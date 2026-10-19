package web

import (
	"context"
	"testing"
)

func TestRootOpensLoginAndRetainsSignedInNavigation(t *testing.T) {
	f := fixture(t)
	anonymous := request(f.s, "GET", "/", nil, nil)
	if anonymous.Code != 303 || anonymous.Header().Get("Location") != "/web/login" {
		t.Fatal("fresh visitor did not reach login")
	}
	token := f.token(t, "visitor")
	cookie := f.login(t, token)
	signedIn := request(f.s, "GET", "/", nil, cookie)
	if signedIn.Code != 303 || signedIn.Header().Get("Location") != "/web/" {
		t.Fatal("signed in visitor did not reach home")
	}
	if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	revoked := request(f.s, "GET", "/", nil, cookie)
	if revoked.Code != 303 || revoked.Header().Get("Location") != "/web/login" {
		t.Fatal("revoked browser did not reach login")
	}
	if err := f.s.db.Close(); err != nil {
		t.Fatal(err)
	}
	failed := request(f.s, "GET", "/", nil, cookie)
	if failed.Code != 503 {
		t.Fatalf("storage failure masked as login: %d", failed.Code)
	}
}
