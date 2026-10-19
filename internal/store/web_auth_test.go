package store

import (
	"agent-board/internal/core"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthenticateIDCurrentIdentityExpiryAndRevoke(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s, err := Open(filepath.Join(t.TempDir(), "board.db"), core.Config{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	token, err := s.CreateToken(ctx, core.TokenInput{Owner: "human", Machine: "fixture", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, until, err := s.AuthenticateID(ctx, token.ID)
	if err != nil || p.ID != token.ID || p.Owner != "human" || !until.Equal(token.ExpiresAt) {
		t.Fatalf("identity %#v %s %v", p, until, err)
	}
	if _, _, err = s.AuthenticateID(ctx, token.Secret); err == nil {
		t.Fatal("secret accepted as ID")
	}
	if _, err = s.db.Exec(`UPDATE tokens SET owner='changed',machine='fresh' WHERE id=?`, token.ID); err != nil {
		t.Fatal(err)
	}
	p, _, err = s.AuthenticateID(ctx, token.ID)
	if err != nil || p.Owner != "changed" || p.Machine != "fresh" {
		t.Fatal("cached identity")
	}
	now = now.Add(time.Hour)
	if _, _, err = s.AuthenticateID(ctx, token.ID); err == nil {
		t.Fatal("expired token accepted")
	}
	now = now.Add(-time.Hour)
	if err = s.RevokeToken(ctx, token.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.AuthenticateID(ctx, token.ID); err == nil {
		t.Fatal("revoked token accepted")
	}
}
