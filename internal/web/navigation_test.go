package web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegacyNavigationAllowlistAndJSONNegotiation(t *testing.T) {
	f := fixture(t)
	project := "p_" + strings.Repeat("a", 32)
	for path, target := range map[string]string{
		"/web/login":                   "/app/login",
		"/web/p/" + project + "/login": "/app/login",
		"/web/p/" + project:            "/app/p/" + project,
		"/web/automation/host-scope?host_id=host&workspace=%2Ftmp":  "/app/automation/host-scope?host_id=host&workspace=%2Ftmp",
		"/web/p/" + project + "/automation/host-scope?host_id=host": "/app/p/" + project + "/automation/host-scope?host_id=host", "/web/projects": "/app/projects",
		"/web/tasks/E-1": "/app/tasks/E-1", "/web/automation/workflows/wf_123": "/app/automation/workflows/wf_123",
		"/web/p/" + project + "/tasks/T-1?q=hello%20world": "/app/p/" + project + "/tasks/T-1?q=hello%20world",
		"/web/catalog/projects/" + project:                 "/app/projects/" + project,
	} {
		for _, method := range []string{"GET", "HEAD"} {
			r := httptest.NewRequest(method, "https://accord.test"+path, nil)
			w := httptest.NewRecorder()
			f.s.ServeHTTP(w, r)
			if w.Code != 303 || w.Header().Get("Location") != target || len(w.Result().Cookies()) != 0 {
				t.Fatalf("navigation %s %s: %d %s", method, path, w.Code, w.Header())
			}
		}
	}
	for _, path := range []string{"/web/missing", "/web/catalog/projects/not-a-project", "/web/p/" + project + "/catalog/projects/" + project, "/web/static/app.js", "/web/static/app.css", "/web/tasks/T-1/update", "/web/p/foreign/tasks/T-1", "/web/tasks/%2f%2fevil.example", "/web/tasks/%2e%2e"} {
		w := httptest.NewRecorder()
		f.s.ServeHTTP(w, httptest.NewRequest("GET", "https://accord.test"+path, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
			t.Fatalf("unknown path %s: %d %s", path, w.Code, w.Header())
		}
	}
	for _, path := range []string{"/web/data/dashboard", "/web/designs/asset/download", "/web/designs/asset/preview", "/api/v1/designs"} {
		r := httptest.NewRequest("GET", "https://accord.test"+path, nil)
		if _, ok := BrowserPageDestination(r); ok {
			t.Fatalf("data route redirected: %s", path)
		}
	}
	pre := decodeBrowser(t, jsonBrowserRequest(f.s, "GET", "/web/login", nil, nil), 200)
	if len(pre.Page.CSRF) != 64 {
		t.Fatal("JSON login contract lost preauth nonce")
	}
}

type unreadLegacyBody struct{ reads int }

func (b *unreadLegacyBody) Read([]byte) (int, error) {
	b.reads++
	return 0, errors.New("body must not be read")
}
func (*unreadLegacyBody) Close() error { return nil }

var _ io.ReadCloser = (*unreadLegacyBody)(nil)

func TestNonJSONMutationsRefuseBeforeParsingOrWriting(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "json-only"))
	for _, path := range []string{"/web/login", "/web/create", "/web/logout", "/web/designs"} {
		for _, origin := range []string{"https://accord.test", "https://foreign.test"} {
			body := &unreadLegacyBody{}
			r := httptest.NewRequest(http.MethodPost, "https://accord.test"+path, body)
			r.Header.Set("Origin", origin)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.AddCookie(cookie)
			w := httptest.NewRecorder()
			f.s.ServeHTTP(w, r)
			want := http.StatusNotAcceptable
			if origin != "https://accord.test" {
				want = http.StatusForbidden
			}
			if w.Code != want || body.reads != 0 || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
				t.Fatalf("legacy submission %s %s: %d reads%d %s", path, origin, w.Code, body.reads, w.Header())
			}
		}
	}
	// A rejected legacy logout cannot revoke the authenticated JSON session.
	decodeBrowser(t, jsonBrowserRequest(f.s, "GET", "/web/", nil, cookie), 200)
}
