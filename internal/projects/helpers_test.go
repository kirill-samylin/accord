package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
	"agent-board/internal/web"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	r                *Registry
	store            *store.Store
	browser          *web.Server
	handler          http.Handler
	token            core.TokenView
	who              Identity
	root, db, webDir string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	if e := os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	db := filepath.Join(root, "board.db")
	webDir := filepath.Join(root, "web")
	cfg := core.Config{Project: "existing-project", LeaseTTL: time.Minute, SessionTTL: time.Minute}
	s, e := store.Open(db, cfg, nil)
	if e != nil {
		t.Fatal(e)
	}
	browser, e := web.New(s, web.Options{Origin: "http://127.0.0.1:8080", DataDir: webDir})
	if e != nil {
		t.Fatal(e)
	}
	token, e := s.CreateToken(context.Background(), core.TokenInput{Owner: "operator", Machine: "machine", TTL: 24 * time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	r, e := Open(Options{DataDir: filepath.Join(root, "registry"), DefaultDB: db, DefaultWeb: webDir, Default: s, Browser: browser, Origin: "http://127.0.0.1:8080", Config: cfg})
	if e != nil {
		t.Fatal(e)
	}
	who, e := r.Authenticate(context.Background(), token.Secret)
	if e != nil {
		t.Fatal(e)
	}
	f := &fixture{r: r, store: s, browser: browser, token: token, who: who, root: root, db: db, webDir: webDir}
	f.handler = transport.NewHandlerWithOptions(s, r.Ready, transport.HandlerOptions{ProjectRouter: r, RequestsPerSecond: 100000, Burst: 100000})
	t.Cleanup(func() { r.Close(); browser.Close(); s.Close() })
	return f
}
func (f *fixture) create(t *testing.T, who Identity, name string) Project {
	t.Helper()
	id, e := random("p_", 16)
	if e != nil {
		t.Fatal(e)
	}
	p, e := f.r.createProject(context.Background(), who, id, name, "", "")
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func (f *fixture) native(t *testing.T, method, path, token, session, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session", session)
	out := httptest.NewRecorder()
	f.handler.ServeHTTP(out, req)
	return out
}
func (f *fixture) request(method, path string, cookies []*http.Cookie, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Accept", "application/json")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://127.0.0.1:8080")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	out := httptest.NewRecorder()
	f.handler.ServeHTTP(out, req)
	return out
}
func (f *fixture) login(t *testing.T, secret string) []*http.Cookie {
	t.Helper()
	page := f.request("GET", "/web/login", nil, nil)
	if page.Code != 200 {
		t.Fatalf("login page %d %s", page.Code, page.Body.String())
	}
	var data struct {
		Page struct {
			CSRF string `json:"csrf"`
		}
	}
	if e := json.Unmarshal(page.Body.Bytes(), &data); e != nil {
		t.Fatal(e)
	}
	out := f.request("POST", "/web/login", page.Result().Cookies(), url.Values{"csrf": {data.Page.CSRF}, "token": {secret}})
	if out.Code != 200 {
		t.Fatalf("login %d %s", out.Code, out.Body.String())
	}
	return out.Result().Cookies()
}
func (f *fixture) describe(t *testing.T, pid, secret, rid, role string) core.SessionView {
	t.Helper()
	path := "/api/v1/session"
	if pid != "" {
		path = "/api/v1/p/" + pid + "/session"
	}
	out := f.native(t, "POST", path, secret, "", `{"request_id":"`+rid+`","role":"`+role+`","run_id":"run","machine":"machine"}`)
	if out.Code != 200 {
		t.Fatalf("describe %d %s", out.Code, out.Body.String())
	}
	var view core.SessionView
	if e := json.Unmarshal(out.Body.Bytes(), &view); e != nil {
		t.Fatal(e)
	}
	return view
}
