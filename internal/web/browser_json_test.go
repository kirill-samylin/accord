package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type browserEnvelope struct {
	OK              bool                           `json:"ok"`
	Page            browserPage                    `json:"page"`
	Redirect        string                         `json:"redirect"`
	Error           struct{ Code, Message string } `json:"error"`
	Draft           map[string]string              `json:"draft"`
	Actions         map[string]string              `json:"actions"`
	FieldErrors     map[string]string              `json:"field_errors"`
	ActionID        string                         `json:"action_id"`
	RetrySameAction bool                           `json:"retry_same_action"`
}

func jsonBrowserRequest(s *Server, method, path string, fields url.Values, cookie *http.Cookie, origins ...string) *httptest.ResponseRecorder {
	var body string
	if fields != nil {
		body = fields.Encode()
	}
	r := httptest.NewRequest(method, "https://accord.test"+path, strings.NewReader(body))
	r.Header.Set("Accept", "application/json")
	if fields != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for _, origin := range origins {
		r.Header.Add("Origin", origin)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func decodeBrowser(t *testing.T, w *httptest.ResponseRecorder, status int) browserEnvelope {
	t.Helper()
	if w.Code != status || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("JSON response: %d %s %s", w.Code, w.Header(), w.Body)
	}
	var out browserEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.OK != (status < 400) {
		t.Fatalf("incorrect ok: %s", w.Body)
	}
	return out
}
func jsonBrowserPage(t *testing.T, f *webFixture, cookie *http.Cookie, path string) browserPage {
	t.Helper()
	return decodeBrowser(t, jsonBrowserRequest(f.s, "GET", path, nil, cookie), 200).Page
}
func jsonBrowserFields(p browserPage, op string, fields url.Values) url.Values {
	if fields == nil {
		fields = url.Values{}
	}
	fields.Set("csrf", p.CSRF)
	fields.Set("action_id", p.Actions[op])
	return fields
}
func TestBrowserJSONLoginRoutesAndProjection(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "json-owner")
	pre := jsonBrowserRequest(f.s, "GET", "/web/login", nil, nil)
	p := decodeBrowser(t, pre, 200).Page
	if p.Name != "login" || len(p.CSRF) != 64 {
		t.Fatalf("preauth %+v", p)
	}
	cookie := pre.Result().Cookies()[0]
	if cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure {
		t.Fatal("preauth flags")
	}
	w := jsonBrowserRequest(f.s, "POST", "/web/login", url.Values{"csrf": {p.CSRF}, "token": {token.Secret}}, cookie, "https://accord.test")
	result := decodeBrowser(t, w, 200)
	if result.Redirect != "/web/" || w.Header().Get("Location") != "" {
		t.Fatalf("login result %s", w.Body)
	}
	for _, value := range w.Result().Cookies() {
		if value.Name == f.s.cookieName {
			cookie = value
		}
	}
	task := f.question(t, token)
	agent := controlAgent(t, f, token, "developer")
	routes := map[string]string{
		"/web/": "overview", "/web/overview": "overview", "/web/tasks": "board", "/web/questions": "board",
		"/web/agents": "agents", "/web/agents/" + agent.SessionID: "agent", "/web/activity": "activity",
		"/web/projects": "projects", "/web/settings": "settings", "/web/stats": "stats", "/web/history": "history",
		"/web/tasks/" + task.Key: "task", "/web/designs": "designs",
	}
	for path, name := range routes {
		w = jsonBrowserRequest(f.s, "GET", path, nil, cookie)
		page := decodeBrowser(t, w, 200).Page
		if page.Name != name || page.Principal.Owner != "json-owner" || len(page.CSRF) != 64 || len(page.Actions["logout"]) != 64 {
			t.Fatalf("page %s: %+v", path, page)
		}
		for _, secret := range []string{token.Secret, cookie.Value, digest(cookie.Value), "pending_describe", "session_hash", "TokenID", "Principal"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("private projection %s", path)
			}
		}
		if w.Header().Get("Vary") != "Accept" || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("cache/CORS headers")
		}
	}
	page := jsonBrowserPage(t, f, cookie, "/web/")
	w = jsonBrowserRequest(f.s, "POST", "/web/logout", jsonBrowserFields(page, "logout", nil), cookie, "https://accord.test")
	if decodeBrowser(t, w, 200).Redirect != "/web/login" {
		t.Fatal("logout redirect")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "GET", "/web/", nil, cookie), 401)
}
func TestBrowserJSONNegotiationAndRawErrors(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "negotiation"))
	for _, accept := range []string{"", "*/*", "application/json;q=0", "application/json;q=0.1,text/html;q=0.9", "application/json;q=invalid"} {
		r := httptest.NewRequest("GET", "https://accord.test/web/", nil)
		r.AddCookie(cookie)
		r.Header.Set("Accept", accept)
		w := httptest.NewRecorder()
		f.s.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), `"ok"`) || f.render.last().CSRF == "" {
			t.Fatalf("SSR representation changed for %q", accept)
		}
	}
	for _, test := range []struct {
		method, path string
		status       int
		code         string
	}{
		{"GET", "/web/missing", 404, "NOT_FOUND"}, {"DELETE", "/web/tasks", 405, "METHOD_NOT_ALLOWED"},
		{"POST", "/web/login", 403, "FORBIDDEN"}, {"GET", "/web/tasks", 401, "UNAUTHORIZED"},
	} {
		w := jsonBrowserRequest(f.s, test.method, test.path, url.Values{"token": {"private-form-value"}}, nil, "https://accord.test")
		if test.method == "GET" {
			w = jsonBrowserRequest(f.s, test.method, test.path, nil, nil)
		}
		if test.path == "/web/missing" {
			w = jsonBrowserRequest(f.s, "GET", test.path, nil, cookie)
		}
		result := decodeBrowser(t, w, test.status)
		if result.Error.Code != test.code || strings.Contains(w.Body.String(), "private-form-value") {
			t.Fatalf("raw error: %s", w.Body)
		}
	}
	w := jsonBrowserRequest(f.s, "HEAD", "/web/tasks", nil, cookie)
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatal("HEAD representation")
	}
}
func TestBrowserJSONOriginCSRFAndRevocation(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "security")
	cookie := f.login(t, token)
	p := jsonBrowserPage(t, f, cookie, "/web/tasks")
	fields := jsonBrowserFields(p, "create", url.Values{"type": {"epic"}, "title": {"preserved"}})
	for _, origins := range [][]string{nil, {"null"}, {"https://evil.test"}, {"https://accord.test", "https://accord.test"}} {
		out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, origins...), 403)
		if out.Draft["title"] != "preserved" {
			t.Fatal("Origin error lost draft")
		}
	}
	fields.Set("csrf", "wrong")
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), 403)
	if err := f.board.RevokeToken(t.Context(), token.ID); err != nil {
		t.Fatal(err)
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "GET", "/web/tasks", nil, cookie), 401)
}
