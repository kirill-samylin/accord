package projects

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistryLegacyPagesNavigateWithoutSessionOrNonce(t *testing.T) {
	f := setup(t)
	project := "p_" + strings.Repeat("a", 32)
	var before int
	if err := f.r.db.QueryRow("SELECT COUNT(*) FROM preauth").Scan(&before); err != nil {
		t.Fatal(err)
	}
	for path, target := range map[string]string{
		"/web/login": "/app/login", "/web/projects": "/app/projects",
		"/web/catalog/projects/" + project:                          "/app/projects/" + project,
		"/web/p/" + project + "/tasks/T-1?q=test":                   "/app/p/" + project + "/tasks/T-1?q=test",
		"/web/p/" + project + "/login":                              "/app/login",
		"/web/p/" + project + "/automation/host-scope?host_id=host": "/app/p/" + project + "/automation/host-scope?host_id=host",
	} {
		for _, method := range []string{"GET", "HEAD"} {
			for _, accept := range []string{"", "application/json;q=0", "application/json;q=0.1,text/html;q=0.9"} {
				r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, nil)
				r.Header.Set("Accept", accept)
				w := httptest.NewRecorder()
				f.handler.ServeHTTP(w, r)
				if w.Code != 303 || w.Header().Get("Location") != target || len(w.Result().Cookies()) != 0 {
					t.Fatalf("registry navigation %s %s Accept%q: %d %s", method, path, accept, w.Code, w.Header())
				}
			}
		}
	}
	for _, path := range []string{"/web/missing", "/web/catalog/projects/not-a-project", "/web/p/foreign/tasks/T-1", "/web/p/" + project + "/catalog/projects/" + project, "/web/static/app.js"} {
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil))
		if w.Code != 404 || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
			t.Fatalf("registry unknown path %s: %d %s", path, w.Code, w.Header())
		}
	}
	var after int
	if err := f.r.db.QueryRow("SELECT COUNT(*) FROM preauth").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatal("legacy navigation issued preauth nonce")
	}
}

type unreadRegistryLegacyBody struct{ reads int }

func (b *unreadRegistryLegacyBody) Read([]byte) (int, error) {
	b.reads++
	return 0, errors.New("legacy body must remain unread")
}
func (*unreadRegistryLegacyBody) Close() error { return nil }

func TestRegistryNonJSONSubmissionsCannotMutateOrReadBody(t *testing.T) {
	f := setup(t)
	cookies := f.login(t, f.token.Secret)
	for _, path := range []string{"/web/login", "/web/logout", "/web/catalog/create", "/web/p/" + f.r.defaultID + "/create", "/web/p/" + f.r.defaultID + "/designs"} {
		for _, origin := range []string{"http://127.0.0.1:8080", "https://foreign.test"} {
			body := &unreadRegistryLegacyBody{}
			r := httptest.NewRequest("POST", "http://127.0.0.1:8080"+path, body)
			r.Header.Set("Origin", origin)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for _, cookie := range cookies {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			f.handler.ServeHTTP(w, r)
			want := http.StatusNotAcceptable
			if origin != "http://127.0.0.1:8080" {
				want = http.StatusForbidden
			}
			if w.Code != want || body.reads != 0 || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
				t.Fatalf("registry legacy POST %s Origin%s: %d reads%d %s", path, origin, w.Code, body.reads, w.Header())
			}
		}
	}
	// The blocked old logout keeps the actual global JSON browser session valid.
	if w := f.request("GET", "/web/projects", cookies, nil); w.Code != 200 {
		t.Fatalf("session mutated: %d", w.Code)
	}
}
