package web

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func multipartLogin(s *Server, pre *http.Cookie, csrf, token, file string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("csrf", csrf)
	if token != "" {
		_ = writer.WriteField("token", token)
	}
	if file != "" {
		part, _ := writer.CreateFormFile("token_file", "local-token")
		_, _ = part.Write([]byte(file))
	}
	_ = writer.Close()
	r := httptest.NewRequest("POST", "https://accord.test/web/login", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Origin", "https://accord.test")
	r.AddCookie(pre)
	w := httptest.NewRecorder()
	observeBrowserResponse(s, w, r)
	return w
}
func TestLoginFileSourceBoundsAndPreauthCSRF(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "one")
	preauth := func() (*http.Cookie, string) {
		w := request(f.s, "GET", "/web/login", nil, nil)
		return w.Result().Cookies()[0], f.response.last().CSRF
	}
	pre, csrf := preauth()
	if w := multipartLogin(f.s, pre, "wrong", "", token.Secret); w.Code != 403 {
		t.Fatalf("preauth CSRF %d", w.Code)
	}
	for _, v := range []struct{ token, file string }{{token.Secret, token.Secret}, {"", strings.Repeat("x", 4097)}, {"", ""}} {
		pre, csrf = preauth()
		w := multipartLogin(f.s, pre, csrf, v.token, v.file)
		if w.Code != 400 {
			t.Fatalf("token source bound %d", w.Code)
		}
		if strings.Contains(w.Body.String(), token.Secret) {
			t.Fatal("token echoed")
		}
	}
	pre, csrf = preauth()
	w := multipartLogin(f.s, pre, csrf, "", token.Secret+"\n")
	if w.Code != 200 {
		t.Fatalf("file login %d %s", w.Code, w.Body)
	}
	// The consumed preauth session cannot be replayed to create extra sessions.
	w = multipartLogin(f.s, pre, csrf, "", token.Secret)
	if w.Code != 403 {
		t.Fatalf("preauth replay %d", w.Code)
	}
	var rawHash, tokenID string
	if err := f.s.db.QueryRow(`SELECT hash,token_id FROM browser_sessions LIMIT 1`).Scan(&rawHash, &tokenID); err != nil {
		t.Fatal(err)
	}
	if tokenID != token.ID || rawHash == token.Secret {
		t.Fatal("wrong browser persistence")
	}
}
func TestBrowserAbsoluteExpiryRevokePostAndPendingDescribe(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "one")
	c := f.login(t, token)
	p := f.page(t, c, "/web/")
	if _, err := f.s.db.Exec(`UPDATE browser_sessions SET expires=? WHERE hash=?`, time.Now().Add(-time.Second).UnixNano(), digest(c.Value)); err != nil {
		t.Fatal(err)
	}
	fields := url.Values{"csrf": {p.CSRF}, "action_id": {p.Actions["create"]}, "type": {"epic"}, "title": {"unsent title"}}
	w := request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test")
	if w.Code != 401 || f.response.last().Draft["title"] != "unsent title" {
		t.Fatalf("expired session %d/draft lost", w.Code)
	}
	c = f.login(t, token)
	p = f.page(t, c, "/web/")
	if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	fields.Set("csrf", p.CSRF)
	fields.Set("action_id", p.Actions["create"])
	w = request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test")
	if w.Code != 401 {
		t.Fatalf("revoked POST %d", w.Code)
	}
}
func TestOriginConfigurationFailsClosed(t *testing.T) {
	f := fixture(t)
	for _, origin := range []string{"https://accord.test/path", "https://u:p@accord.test", "http://accord.test", "http://localhost", "https://accord.test?x=1", "https://accord.test#x", "https://", "null"} {
		s, err := New(f.board, Options{Origin: origin, DataDir: filepath.Join(f.dir, "invalid")})
		if err == nil {
			s.Close()
			t.Fatalf("accepted origin %s", origin)
		}
	}
	s, err := New(f.board, Options{Origin: "http://127.0.0.1:1234", DataDir: filepath.Join(f.dir, "dev")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.secure || s.cookieName != "accord_dev_session" {
		t.Fatal("dev cookie contract")
	}
}
