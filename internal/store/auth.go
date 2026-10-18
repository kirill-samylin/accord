package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	err = s.adminTransaction(ctx, func(b *board, now time.Time) error {
		roles, err := tokenRoles(b, in)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(roles)
		if err != nil {
			return err
		}
		if _, err := b.tx.ExecContext(ctx, `INSERT INTO tokens(id,hash,owner,machine,expires,roles) VALUES(?,?,?,?,?,?)`, id, tokenHash(result.Secret), in.Owner, in.Machine, result.ExpiresAt.UnixNano(), string(raw)); err != nil {
			return err
		}
		emit(b, now, "token_created", "", "")
		b.Events[len(b.Events)-1].TokenID = id
		return nil
	})
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
	return s.adminTransaction(ctx, func(b *board, now time.Time) error {
		var revoked bool
		err := b.tx.QueryRowContext(ctx, `SELECT revoked FROM tokens WHERE id=?`, id).Scan(&revoked)
		if err == sql.ErrNoRows {
			return core.Fail("not_found", "token does not exist")
		}
		if err != nil || revoked {
			return err
		}
		if _, err = b.tx.ExecContext(ctx, `UPDATE tokens SET revoked=1 WHERE id=?`, id); err != nil {
			return err
		}
		emit(b, now, "token_revoked", "", "")
		b.Events[len(b.Events)-1].TokenID = id
		return nil
	})
}

// ListTokens exposes identity and validity only; neither secrets nor hashes.
func (s *Store) ListTokens(ctx context.Context) ([]core.TokenInfo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,owner,machine,expires,revoked,roles,scopes_version FROM tokens ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []core.TokenInfo{}
	for rows.Next() {
		var token core.TokenInfo
		var expires int64
		var raw []byte
		if err = rows.Scan(&token.ID, &token.Owner, &token.Machine, &expires, &token.Revoked, &raw, &token.ScopesVersion); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &token.Roles); err != nil {
			return nil, err
		}
		token.ExpiresAt = time.Unix(0, expires).UTC()
		result = append(result, token)
	}
	return result, rows.Err()
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
