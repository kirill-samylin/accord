package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type captureRenderer struct {
	mu   sync.Mutex
	page Page
	name string
}

func (c *captureRenderer) Render(w http.ResponseWriter, status int, name string, p Page) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.page = p
	c.name = name
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(p)
}
func (c *captureRenderer) last() Page { c.mu.Lock(); defer c.mu.Unlock(); return c.page }

type webFixture struct {
	s      *Server
	board  *store.Store
	render *captureRenderer
	now    time.Time
	dir    string
}

func fixture(t *testing.T) *webFixture {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	f := &webFixture{now: time.Now(), dir: dir, render: new(captureRenderer)}
	f.board, err = store.Open(filepath.Join(dir, "board.db"), core.Config{Project: "fixture", SessionTTL: time.Minute, LeaseTTL: time.Minute}, func() time.Time { return f.now })
	if err != nil {
		t.Fatal(err)
	}
	f.s, err = New(f.board, Options{Origin: "https://accord.test", DataDir: filepath.Join(dir, "web"), Renderer: f.render})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.s.Close(); f.board.Close() })
	return f
}
func (f *webFixture) token(t *testing.T, owner string) core.TokenView {
	t.Helper()
	v, err := f.board.CreateToken(context.Background(), core.TokenInput{Owner: owner, Machine: "fixture", TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func request(s *Server, method, path string, fields url.Values, cookie *http.Cookie, origins ...string) *httptest.ResponseRecorder {
	var body string
	if fields != nil {
		body = fields.Encode()
	}
	r := httptest.NewRequest(method, "https://accord.test"+path, strings.NewReader(body))
	if fields != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for _, o := range origins {
		r.Header.Add("Origin", o)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func (f *webFixture) login(t *testing.T, token core.TokenView, old ...*http.Cookie) *http.Cookie {
	t.Helper()
	w := request(f.s, "GET", "/web/login", nil, nil)
	if w.Code != 200 {
		t.Fatalf("login get %d %s", w.Code, w.Body)
	}
	csrf := f.render.last().CSRF
	pre := w.Result().Cookies()[0]
	r := httptest.NewRequest("POST", "https://accord.test/web/login", strings.NewReader(url.Values{"csrf": {csrf}, "token": {token.Secret}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://accord.test")
	r.AddCookie(pre)
	for _, c := range old {
		r.AddCookie(c)
	}
	w = httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatalf("login %d %s", w.Code, w.Body)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == f.s.cookieName {
			return c
		}
	}
	t.Fatal("no browser cookie")
	return nil
}
func (f *webFixture) page(t *testing.T, c *http.Cookie, path string) Page {
	t.Helper()
	w := request(f.s, "GET", path, nil, c)
	if w.Code != 200 {
		t.Fatalf("page %s %d %s", path, w.Code, w.Body)
	}
	return f.render.last()
}
func (f *webFixture) post(c *http.Cookie, path string, p Page, op string, fields url.Values) *httptest.ResponseRecorder {
	if fields == nil {
		fields = url.Values{}
	}
	fields.Set("csrf", p.CSRF)
	id := p.Actions[op]
	if id == "" && p.Task != nil {
		id = p.Actions[op+":"+p.Task.Key]
	}
	fields.Set("action_id", id)
	return request(f.s, "POST", path, fields, c, "https://accord.test")
}
func (f *webFixture) native(t *testing.T, token core.TokenView) core.Actor {
	t.Helper()
	p, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	id, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.board.Describe(context.Background(), p, core.OpenSession{Role: "human", Machine: p.Machine, RunID: id, RequestID: id})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: p, SessionID: v.SessionID}
}
func (f *webFixture) question(t *testing.T, token core.TokenView) core.Task {
	t.Helper()
	a := f.native(t, token)
	epic, err := f.board.Create(context.Background(), a, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Project"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := f.board.Create(context.Background(), a, core.CreateInput{RequestID: "question", Type: "task", Role: "human", EpicKey: epic.Key, Title: "Question", Description: "Context"})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestBrowserCookieRotationRevokeAndLogout(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "one")
	first := f.login(t, token)
	if !first.Secure || !first.HttpOnly || first.SameSite != http.SameSiteStrictMode || first.Path != "/" || first.Domain != "" || first.Name != "__Host-accord_session" {
		t.Fatalf("cookie flags %#v", first)
	}
	p := f.page(t, first, "/web/")
	w := request(f.s, "GET", "/api/v1/designs", nil, first)
	if w.Code != 401 {
		t.Fatalf("cookie authorizes native API: %d", w.Code)
	}
	w = request(f.s, "GET", "/web/", nil, &http.Cookie{Name: f.s.cookieName, Value: token.ID})
	if w.Code != 401 {
		t.Fatalf("ID authorizes cookie: %d", w.Code)
	}
	second := f.login(t, token, first)
	if first.Value == second.Value {
		t.Fatal("session fixation")
	}
	if p.CSRF == f.page(t, second, "/web/").CSRF {
		t.Fatal("CSRF not rotated")
	}
	if w = request(f.s, "GET", "/web/", nil, first); w.Code != 401 {
		t.Fatal("old session survived login")
	}
	p = f.page(t, second, "/web/")
	if w = f.post(second, "/web/logout", p, "logout", nil); w.Code != 303 {
		t.Fatalf("logout %d", w.Code)
	}
	if w = request(f.s, "GET", "/web/", nil, second); w.Code != 401 {
		t.Fatal("logout session survived")
	}
	third := f.login(t, token)
	if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	if w = request(f.s, "GET", "/web/", nil, third); w.Code != 401 {
		t.Fatalf("revoked cookie %d", w.Code)
	}
}
func TestBrowserOriginCSRFAndDraftBoundaries(t *testing.T) {
	f := fixture(t)
	c := f.login(t, f.token(t, "one"))
	p := f.page(t, c, "/web/")
	fields := url.Values{"csrf": {p.CSRF}, "action_id": {p.Actions["create"]}, "type": {"epic"}, "title": {"<script>draft</script>"}}
	for _, origins := range [][]string{nil, {"null"}, {""}, {"https://evil.test"}, {"https://accord.test", "https://accord.test"}} {
		w := request(f.s, "POST", "/web/tasks", fields, c, origins...)
		if w.Code != 403 {
			t.Fatalf("origin %#v: %d", origins, w.Code)
		}
		if f.render.last().Draft["title"] != fields.Get("title") {
			t.Fatal("draft lost")
		}
	}
	fields.Set("csrf", "wrong")
	if w := request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test"); w.Code != 403 {
		t.Fatal("bad CSRF accepted")
	}
	fields.Set("csrf", p.CSRF)
	w := request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test")
	if w.Code != 303 {
		t.Fatalf("valid create %d %s", w.Code, w.Body)
	}
	w = request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test")
	if w.Code != 303 {
		t.Fatalf("replay %d", w.Code)
	}
	fields.Set("title", "changed")
	if w = request(f.s, "POST", "/web/tasks", fields, c, "https://accord.test"); w.Code != 409 {
		t.Fatalf("different replay %d", w.Code)
	}
}
func TestFailedLoginLimiterReservesConcurrentAttempts(t *testing.T) {
	f := fixture(t)
	w := request(f.s, "GET", "/web/login", nil, nil)
	csrf := f.render.last().CSRF
	pre := w.Result().Cookies()[0]
	var wg sync.WaitGroup
	codes := make(chan int, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("POST", "https://accord.test/web/login", strings.NewReader(url.Values{"csrf": {csrf}, "token": {"invalid"}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", "https://accord.test")
			r.Header.Set("X-Forwarded-For", time.Now().String())
			r.Header.Set("X-Accord-Client-IP", time.Now().String())
			r.AddCookie(pre)
			w := httptest.NewRecorder()
			f.s.ServeHTTP(w, r)
			codes <- w.Code
		}()
	}
	wg.Wait()
	close(codes)
	unauthorized, limited := 0, 0
	for code := range codes {
		switch code {
		case 401:
			unauthorized++
		case 429:
			limited++
		default:
			t.Fatalf("unexpected code %d", code)
		}
	}
	if unauthorized != 5 || limited != 11 {
		t.Fatalf("attempt reservation %d failures %d limited", unauthorized, limited)
	}
}
