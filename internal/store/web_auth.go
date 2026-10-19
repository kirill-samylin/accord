package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"time"
)

// AuthenticateID revalidates an already authenticated browser session's token.
// The ID is not a credential and must never authorize a public request itself.
func (s *Store) AuthenticateID(ctx context.Context, id string) (core.Principal, time.Time, error) {
	var p core.Principal
	var expiry int64
	err := s.db.QueryRowContext(ctx, `SELECT id,owner,machine,expires FROM tokens WHERE id=? AND revoked=0`, id).Scan(&p.ID, &p.Owner, &p.Machine, &expiry)
	if err == sql.ErrNoRows {
		return p, time.Time{}, core.Fail("unauthorized", "invalid or expired token")
	}
	if err != nil {
		return p, time.Time{}, err
	}
	until := time.Unix(0, expiry).UTC()
	if !until.After(s.clock()) {
		return core.Principal{}, time.Time{}, core.Fail("unauthorized", "invalid or expired token")
	}
	return p, until, nil
}
