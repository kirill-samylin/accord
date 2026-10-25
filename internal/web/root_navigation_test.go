package web

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestRootAndLegacyPagesNavigateWithoutCreatingBrowserState(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "visitor")
	cookie := f.login(t, token)
	if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.Close(); err != nil {
		t.Fatal(err)
	}
	for path, target := range map[string]string{
		"/": "/app", "/web/": "/app", "/web/login": "/app/login",
		"/web/tasks/T-1?q=test": "/app/tasks/T-1?q=test",
	} {
		for _, method := range []string{"GET", "HEAD"} {
			r := httptest.NewRequest(method, "https://accord.test"+path, nil)
			r.AddCookie(cookie)
			w := httptest.NewRecorder()
			f.s.ServeHTTP(w, r)
			if w.Code != 303 || w.Header().Get("Location") != target || len(w.Result().Cookies()) != 0 {
				t.Fatalf("stateless navigation %s %s: %d %s", method, path, w.Code, w.Header())
			}
		}
	}
}
