package web

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardBrowserAndCachedBoardExpiry(t *testing.T) {
	for _, boundary := range []string{"browser", "board", "duplicate_cookie"} {
		t.Run(boundary, func(t *testing.T) {
			f := fixture(t)
			cookie := f.login(t, f.token(t, "operator"))
			status, code := 401, "UNAUTHORIZED"
			switch boundary {
			case "browser":
				if _, err := f.s.db.Exec(`UPDATE browser_sessions SET expires=?`, time.Now().Add(-time.Second).UnixNano()); err != nil {
					t.Fatal(err)
				}
			case "board":
				b, err := f.s.loadBrowser(newBrowserRequest(cookie))
				if err != nil {
					t.Fatal(err)
				}
				b.Session.ExpiresAt = time.Now().Add(-time.Second)
				if err = f.s.saveSession(t.Context(), b); err != nil {
					t.Fatal(err)
				}
				status, code = 409, "LEASE_LOST"
			}
			for _, method := range []string{"GET", "HEAD"} {
				r := httptest.NewRequest(method, "https://accord.test/web/data/dashboard", nil)
				r.AddCookie(cookie)
				if boundary == "duplicate_cookie" {
					r.AddCookie(cookie)
				}
				w := httptest.NewRecorder()
				f.s.ServeHTTP(w, r)
				if w.Code != status || method == "GET" && !strings.Contains(w.Body.String(), `"code":"`+code+`"`) || method == "HEAD" && w.Body.Len() != 0 {
					t.Fatalf("%s %s: %d %s", boundary, method, w.Code, w.Body)
				}
			}
		})
	}
}

func TestAgentKindRoleConflictsRemainStrict(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	controlAgent(t, f, token, "analyst")
	cookie := f.login(t, token)
	for _, filter := range []string{"role=human", "kind=agent&role=human", "kind=human&role=analyst"} {
		w := request(f.s, "GET", "/web/agents?"+filter, nil, cookie)
		if w.Code != 400 {
			t.Fatalf("ambiguous role/kind normalized: %s %d", filter, w.Code)
		}
	}
	for _, filter := range []string{"kind=all&role=human", "kind=human&role=human", "kind=agent&role=analyst", "kind=all&role=analyst"} {
		page := f.page(t, cookie, "/web/agents?"+filter)
		if len(page.Agents) != 1 {
			t.Fatalf("valid exact role/kind filter %s: %+v", filter, page.Agents)
		}
	}
}
