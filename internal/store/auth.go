package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

func tokenHash(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}

// CreateToken returns the secret once. Only its SHA-256 digest is persisted.
func (s *Store) CreateToken(ctx context.Context, in core.TokenInput) (core.TokenView, error) {
	var result core.TokenView
	if !bounded(in.Owner, 128) || !bounded(in.Machine, 128) || in.TTL < time.Second || in.TTL > 365*24*time.Hour {
		return result, core.Fail("invalid_input", "owner, machine and TTL in [1s,365d] required")
	}
	id, err := identifier()
	if err != nil {
		return result, err
	}
	secret, err := identifier()
	if err != nil {
		return result, err
	}
	result = core.TokenView{ID: id, Secret: "ab_" + secret, ExpiresAt: s.clock().UTC().Add(in.TTL)}
	_, err = s.db.ExecContext(ctx, `INSERT INTO tokens(id,hash,owner,machine,expires) VALUES(?,?,?,?,?)`, id, tokenHash(result.Secret), in.Owner, in.Machine, result.ExpiresAt.UnixNano())
	return result, err
}
func (s *Store) Authenticate(ctx context.Context, secret string) (core.Principal, error) {
	var p core.Principal
	var expiry int64
	err := s.db.QueryRowContext(ctx, `SELECT id,owner,machine,expires FROM tokens WHERE hash=? AND revoked=0`, tokenHash(secret)).Scan(&p.ID, &p.Owner, &p.Machine, &expiry)
	if err == sql.ErrNoRows {
		return p, core.Fail("unauthorized", "invalid or expired token")
	}
	if err != nil {
		return p, err
	}
	if expiry <= s.clock().UnixNano() {
		return core.Principal{}, core.Fail("unauthorized", "invalid or expired token")
	}
	return p, nil
}

// RevokeToken invalidates future authentication, including idempotent replay.
func (s *Store) RevokeToken(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE tokens SET revoked=1 WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return core.Fail("not_found", "token does not exist")
	}
	return nil
}

func authenticatePrincipal(ctx context.Context, b *board, p core.Principal, now time.Time) error {
	var owner, machine string
	var expires int64
	err := b.tx.QueryRowContext(ctx, `SELECT owner,machine,expires FROM tokens WHERE id=? AND revoked=0`, p.ID).Scan(&owner, &machine, &expires)
	if err == sql.ErrNoRows {
		return core.Fail("unauthorized", "invalid or expired token")
	}
	if err != nil {
		return err
	}
	if expires <= now.UnixNano() || p.Owner != owner || p.Machine != machine {
		return core.Fail("unauthorized", "invalid or expired token")
	}
	return nil
}
