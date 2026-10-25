package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type designFixture struct {
	s     *Server
	board *store.Store
	b     *browserSession
	token string
	dir   string
}

func designFixtureNew(t *testing.T) *designFixture {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	board, err := store.Open(filepath.Join(dir, "board.db"), core.Config{Project: "design-test", SessionTTL: time.Hour, LeaseTTL: time.Minute}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { board.Close() })
	token, err := board.CreateToken(context.Background(), core.TokenInput{Owner: "uploader", Machine: "pc", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	session, err := board.Describe(context.Background(), principal, core.OpenSession{RequestID: "test-design-session", Role: "human", RunID: "design-test", Machine: "pc"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(board, Options{Origin: "https://accord.example", DataDir: filepath.Join(dir, "web")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	b := &browserSession{Hash: strings.Repeat("a", 64), CSRF: strings.Repeat("b", 64), TokenID: principal.ID, Principal: principal, Session: session, Expires: time.Now().Add(time.Hour)}
	return &designFixture{s: s, board: board, b: b, token: token.Secret, dir: filepath.Join(dir, "web")}
}

// ResponseController requires real deadline support. This test writer implements
// that narrow capability while leaving transfer behavior to the actual handler.
type designDeadlineWriter struct{ *httptest.ResponseRecorder }

func (*designDeadlineWriter) SetReadDeadline(time.Time) error  { return nil }
func (*designDeadlineWriter) SetWriteDeadline(time.Time) error { return nil }

func (f *designFixture) upload(t *testing.T, id string, in designUpload, data []byte) (DesignAsset, error) {
	t.Helper()
	if id == "" {
		var err error
		id, err = f.s.issueAction(context.Background(), f.b, "upload")
		if err != nil {
			t.Fatal(err)
		}
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, field := range []struct{ k, v string }{{"csrf", f.b.CSRF}, {"action_id", id}, {"title", in.Title}, {"scope", in.Scope}, {"scope_key", in.ScopeKey}, {"epic_key", in.EpicKey}, {"design_id", in.DesignID}} {
		if err := writer.WriteField(field.k, field.v); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.CreateFormFile("file", in.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "https://accord.example/web/designs", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return f.s.uploadDesign(&designDeadlineWriter{httptest.NewRecorder()}, request, f.b)
}
func designInput() designUpload {
	return designUpload{Title: "Дизайн", Scope: "project", Filename: "source.html"}
}
func designRequireStatus(t *testing.T, err error, status int) {
	t.Helper()
	got, _, _ := designErrorInfo(err)
	if err == nil || got != status {
		t.Fatalf("status: got %v/%d want %d", err, got, status)
	}
}

func TestDesignVersionsReplayRecoveryAndAuth(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	id, err := f.s.issueAction(ctx, f.b, "upload")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("<!DOCTYPE html><title>one</title><script>alert(1)</script>")
	first, err := f.upload(t, id, designInput(), data)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || first.MediaType != "text/html" || first.PreviewURL != "" || first.Size != int64(len(data)) {
		t.Fatalf("bad metadata: %+v", first)
	}
	replay, err := f.upload(t, id, designInput(), data)
	if err != nil || replay.AssetID != first.AssetID {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	_, err = f.upload(t, id, designInput(), []byte("<!DOCTYPE html><title>other</title>"))
	designRequireStatus(t, err, 409)
	changed := designInput()
	changed.Title = "other"
	_, err = f.upload(t, id, changed, data)
	designRequireStatus(t, err, 409)
	version := designInput()
	version.DesignID = first.DesignID
	second, err := f.upload(t, "", version, []byte("<!DOCTYPE html><title>two</title>"))
	if err != nil || second.Version != 2 || second.AssetID == first.AssetID {
		t.Fatalf("version: %+v %v", second, err)
	}
	page, err := f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{DesignID: first.DesignID, Limit: 1})
	if err != nil || len(page.Assets) != 1 || page.NextCursor == "" || page.Assets[0].Version != 2 {
		t.Fatalf("page: %+v %v", page, err)
	}
	next, err := f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{DesignID: first.DesignID, Limit: 1, Cursor: page.NextCursor})
	if err != nil || len(next.Assets) != 1 || next.Assets[0].Version != 1 {
		t.Fatalf("next: %+v %v", next, err)
	}
	_, err = f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{Limit: 1, Cursor: page.NextCursor})
	designRequireStatus(t, err, 400)
	if err = f.s.designs.recover(ctx); err != nil {
		t.Fatal(err)
	}
	get, err := f.s.GetDesign(ctx, f.b.actor(), first.AssetID)
	if err != nil || get.SHA256 != first.SHA256 {
		t.Fatalf("get after recovery: %+v %v", get, err)
	}
	if err = f.board.RevokeToken(ctx, f.b.TokenID); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.GetDesign(ctx, f.b.actor(), first.AssetID)
	designRequireStatus(t, err, 401)
	_, err = f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{})
	designRequireStatus(t, err, 401)
	_, err = f.upload(t, id, designInput(), data)
	designRequireStatus(t, err, 401)
}

func TestDesignPreviewAndAttachment(t *testing.T) {
	f := designFixtureNew(t)
	var raster bytes.Buffer
	if err := png.Encode(&raster, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	in := designInput()
	in.Filename = "photo.png"
	a, err := f.upload(t, "", in, raster.Bytes())
	if err != nil || a.PreviewURL == "" || a.MediaType != "image/png" {
		t.Fatalf("PNG: %+v %v", a, err)
	}
	v, err := f.s.designs.get(context.Background(), a.AssetID)
	if err != nil {
		t.Fatal(err)
	}
	for _, preview := range []bool{false, true} {
		w := &designDeadlineWriter{httptest.NewRecorder()}
		f.s.serveDesign(w, httptest.NewRequest("GET", "/", nil), v, preview)
		if w.Code != 200 {
			t.Fatalf("download status: %d", w.Code)
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "sandbox") || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("missing security headers")
		}
		if preview {
			if w.Header().Get("Content-Type") != "image/jpeg" {
				t.Fatal("unsafe preview")
			}
			if _, _, err = image.Decode(bytes.NewReader(w.Body.Bytes())); err != nil {
				t.Fatal(err)
			}
		} else if !bytes.Equal(w.Body.Bytes(), raster.Bytes()) || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
			t.Fatal("original must be attachment")
		}
	}
	_, err = f.upload(t, "", in, raster.Bytes()[:len(raster.Bytes())/2])
	designRequireStatus(t, err, 400)
	if err = os.Chmod(filepath.Join(f.dir, "blobs", a.AssetID), 0644); err != nil {
		t.Fatal(err)
	}
	w := &designDeadlineWriter{httptest.NewRecorder()}
	f.s.serveDesign(w, httptest.NewRequest("GET", "/", nil), v, false)
	if w.Code != 404 {
		t.Fatalf("unsafe permissions read: %d", w.Code)
	}
}

func TestDesignStreamLimitsAndCancel(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	for _, delta := range []int64{0, 1} {
		_, size, err := f.s.designs.stream(ctx, io.Discard, io.LimitReader(designZeroReader{}, maxDesignFile+delta))
		if delta == 0 {
			if err != nil || size != maxDesignFile {
				t.Fatalf("exact limit: %d %v", size, err)
			}
		} else {
			designRequireStatus(t, err, 413)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err := f.s.designs.stream(cancelled, io.Discard, strings.NewReader("x"))
	if err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
}

type designZeroReader struct{}

func (designZeroReader) Read(b []byte) (int, error) { clear(b); return len(b), nil }
