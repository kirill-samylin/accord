package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

// GuardCredential linearizes legacy revocation with a short operation in another
// board. Never call back into this Store or hold this lock across HTTP streaming.
func (s *Store) GuardCredential(ctx context.Context, p core.Principal, call func() error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE metadata SET sequence=sequence WHERE id=1`); err != nil {
		return err
	}
	if err = authenticatePrincipal(ctx, &board{tx: tx}, p, s.clock().UTC()); err != nil {
		return err
	}
	if err = call(); err != nil {
		return err
	}
	return tx.Commit()
}

// ProjectPrincipal creates an inert local identity. Its random bearer is never
// returned; only the registry may map an independently authenticated account to it.
// authorize is used when this database is also the legacy credential authority.
func (s *Store) ProjectPrincipal(ctx context.Context, owner, machine string, expires time.Time, authorize *core.Principal) (core.Principal, error) {
	var p core.Principal
	id, err := identifier()
	if err != nil {
		return p, err
	}
	secret, err := identifier()
	if err != nil {
		return p, err
	}
	p = core.Principal{ID: id, Owner: owner, Machine: machine}
	err = s.adminTransaction(ctx, func(b *board, now time.Time) error {
		if authorize != nil {
			if e := authenticatePrincipal(ctx, b, *authorize, now); e != nil {
				return e
			}
		}
		if !expires.After(now) {
			return core.Fail("unauthorized", "credential expired")
		}
		_, e := b.tx.ExecContext(ctx, `INSERT INTO tokens(id,hash,owner,machine,expires) VALUES(?,?,?,?,?)`, id, tokenHash(secret), owner, machine, expires.UnixNano())
		return e
	})
	return p, err
}

// CheckNoLiveLeases is used while the registry's authorization gate is held.
func (s *Store) CheckNoLiveLeases(ctx context.Context) error {
	_, err := transact(s, ctx, func(b *board, now time.Time) (bool, error) {
		for _, task := range b.Tasks {
			if task.Lease != nil && task.Lease.ExpiresAt.After(now) {
				return false, core.Fail("conflict", "project has active leases")
			}
		}
		return true, nil
	})
	return err
}
