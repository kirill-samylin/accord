package web

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type browserDeadlineWriter struct {
	*httptest.ResponseRecorder
	read, write time.Time
}

func (w *browserDeadlineWriter) SetReadDeadline(v time.Time) error  { w.read = v; return nil }
func (w *browserDeadlineWriter) SetWriteDeadline(v time.Time) error { w.write = v; return nil }
func jsonUploadRequest(t *testing.T, f *webFixture, cookie *http.Cookie, page browserPage, data []byte) *browserDeadlineWriter {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, pair := range [][2]string{{"csrf", page.CSRF}, {"action_id", page.Actions["upload"]}, {"title", "Uploaded design"}, {"scope", "project"}, {"scope_key", "fixture"}} {
		if err := form.WriteField(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	file, err := form.CreateFormFile("file", "design.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "https://accord.test/web/designs", &body)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Origin", "https://accord.test")
	r.Header.Set("Content-Type", form.FormDataContentType())
	r.AddCookie(cookie)
	w := &browserDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	observeBrowserResponse(f.s, w, r)
	return w
}
func TestBrowserJSONUploadMetadataDeadlineReplayAndBinary(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "upload")
	cookie := f.login(t, token)
	p := jsonBrowserPage(t, f, cookie, "/web/designs")
	data := []byte("<!doctype html><title>Design sample</title><p>Content</p>")
	w := jsonUploadRequest(t, f, cookie, p, data)
	out := decodeBrowser(t, w.ResponseRecorder, 200)
	if !strings.HasPrefix(out.Redirect, "/web/designs?design_id=") {
		t.Fatalf("upload redirect %s", w.Body)
	}
	if time.Until(w.read) < 119*time.Second || time.Until(w.read) > 120*time.Second || !w.read.Equal(w.write) {
		t.Fatal("upload deadline not forwarded through JSON representation")
	}
	if maxDesignFile != 50<<20 {
		t.Fatal("upload size contract changed")
	}
	replay := decodeBrowser(t, jsonUploadRequest(t, f, cookie, p, data).ResponseRecorder, 200)
	if replay.Redirect != out.Redirect {
		t.Fatal("upload replay changed material")
	}
	listed := jsonBrowserPage(t, f, cookie, out.Redirect)
	if len(listed.Designs) != 1 {
		t.Fatalf("upload replay duplicated metadata %+v", listed.Designs)
	}
	asset := listed.Designs[0]
	if asset.Size != int64(len(data)) || asset.Filename != "design.html" || asset.Version != 1 || asset.MediaType != "text/html" || !strings.HasPrefix(asset.DownloadURL, "/web/designs/") || len(asset.SHA256) != 64 {
		t.Fatalf("metadata %+v", asset)
	}
	serialized := jsonBrowserRequest(f.s, "GET", out.Redirect, nil, cookie).Body.String()
	for _, private := range []string{f.dir, cookie.Value, token.Secret, "metadata_hash", "blob", "staging", "request_id"} {
		if strings.Contains(serialized, private) {
			t.Fatalf("private upload metadata field %q", private)
		}
	}
	decodeBrowser(t, jsonUploadRequest(t, f, cookie, p, []byte("<!doctype html><title>Changed</title>")).ResponseRecorder, 409)
	r := httptest.NewRequest("GET", "https://accord.test"+asset.DownloadURL, nil)
	r.AddCookie(cookie)
	r.Header.Set("Accept", "application/json")
	binary := &browserDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	observeBrowserResponse(f.s, binary, r)
	if binary.Code != 200 || binary.Body.String() != string(data) || binary.Header().Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(binary.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatal("binary transfer negotiated JSON")
	}
	if binary.Header().Get("Vary") != "" {
		t.Fatal("binary has page negotiation")
	}
}
func TestBrowserJSONUploadValidationRotatesActionAndKeepsDraft(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "invalid-upload"))
	p := jsonBrowserPage(t, f, cookie, "/web/designs")
	out := decodeBrowser(t, jsonUploadRequest(t, f, cookie, p, []byte("not a supported document")).ResponseRecorder, 400)
	if out.Draft["title"] != "Uploaded design" || out.Actions["upload"] == p.Actions["upload"] || out.Actions["upload"] == "" || out.RetrySameAction {
		t.Fatalf("upload validation %+v", out)
	}
	bad := p
	bad.CSRF = "bad"
	decodeBrowser(t, jsonUploadRequest(t, f, cookie, bad, []byte("<!doctype html><title>test</title>")).ResponseRecorder, 403)
}

func TestBinaryRoutesRetainAuthenticationStatusWithoutLegacyRenderer(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "binary-auth")
	cookie := f.login(t, token)
	page := jsonBrowserPage(t, f, cookie, "/web/designs")
	data := []byte("<!doctype html><title>private design</title>")
	out := decodeBrowser(t, jsonUploadRequest(t, f, cookie, page, data).ResponseRecorder, 200)
	assets := jsonBrowserPage(t, f, cookie, out.Redirect).Designs
	if len(assets) != 1 {
		t.Fatal("missing uploaded design")
	}
	paths := []string{assets[0].DownloadURL, strings.TrimSuffix(assets[0].DownloadURL, "/download") + "/preview"}
	for _, revoked := range []bool{false, true} {
		if revoked {
			if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
				t.Fatal(err)
			}
		}
		for _, path := range paths {
			for _, accept := range []string{"", "application/json"} {
				r := httptest.NewRequest("GET", "https://accord.test"+path, nil)
				r.Header.Set("Accept", accept)
				if revoked {
					r.AddCookie(cookie)
				}
				w := httptest.NewRecorder()
				f.s.ServeHTTP(w, r)
				if w.Code != 401 || w.Header().Get("Location") != "" || strings.Contains(w.Body.String(), "private design") {
					t.Fatalf("binary auth revoked%v Accept%q: %d %s", revoked, accept, w.Code, w.Body)
				}
				r = httptest.NewRequest("POST", "https://accord.test"+path, nil)
				r.Header.Set("Accept", accept)
				r.Header.Set("Origin", "https://foreign.test")
				w = httptest.NewRecorder()
				f.s.ServeHTTP(w, r)
				if w.Code != 403 {
					t.Fatalf("binary foreign Origin: %d", w.Code)
				}
			}
		}
	}
}
