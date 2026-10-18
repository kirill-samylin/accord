package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"time"
)

// OpenExisting is for local administrative commands that must never bootstrap
// an empty database because an operator misspelled a path.
func OpenExisting(path string, cfg core.Config, clock core.Clock) (*Store, error) {
	absolute, err := existingDatabase(path)
	if err != nil {
		return nil, err
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 15 * time.Minute
	}
	if cfg.LeaseTTL == 0 {
		cfg.LeaseTTL = 15 * time.Minute
	}
	if cfg.SessionTTL < time.Second || cfg.LeaseTTL < time.Second {
		return nil, core.Fail("invalid_input", "TTL must be at least one second")
	}
	if clock == nil {
		clock = time.Now
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute), RawQuery: "mode=rw"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err = db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err == nil {
		err = verifySchema(ctx, db)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, cfg: cfg, clock: clock}, nil
}

// adminTransaction does not materialize TTL expiry: recovery must close all
// restored attempts with its explicit reason, even when their clocks expired.
func (s *Store) adminTransaction(ctx context.Context, fn func(*board, time.Time) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE metadata SET sequence=sequence WHERE id=1`); err != nil {
		return err
	}
	b, err := loadBoard(ctx, tx)
	if err != nil {
		return err
	}
	before := metricFacts(b)
	now := s.clock().UTC()
	if err = fn(b, now); err != nil {
		return err
	}
	enrichEvents(b, before, now)
	if err = saveBoard(ctx, tx, b); err != nil {
		return err
	}
	return tx.Commit()
}

// RevokeAllTokens is a local administrator operation, unavailable to bearer API.
func (s *Store) RevokeAllTokens(ctx context.Context, reason string) (int, error) {
	if !bounded(reason, 4096) {
		return 0, core.Fail("invalid_input", "revocation reason is required")
	}
	var count int
	err := s.adminTransaction(ctx, func(b *board, now time.Time) error {
		var err error
		count, err = revokeAll(ctx, b, now)
		if err == nil {
			b.Events[len(b.Events)-1].Reason = reason
		}
		return err
	})
	return count, err
}

func revokeAll(ctx context.Context, b *board, now time.Time) (int, error) {
	result, err := b.tx.ExecContext(ctx, `UPDATE tokens SET revoked=1 WHERE revoked=0`)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err == nil {
		emit(b, now, "tokens_revoked_all", "", "")
	}
	return int(count), err
}

// RecoverState must run offline before a restored database is exposed. It
// invalidates credentials and sessions together, preserving fencing and replay
// history. A fresh credential cannot resume any pre-recovery session.
func (s *Store) RecoverState(ctx context.Context, reason string) (core.RecoveryResult, error) {
	var result core.RecoveryResult
	if reason != "restore" {
		return result, core.Fail("invalid_input", "recovery requires reason restore")
	}
	err := s.adminTransaction(ctx, func(b *board, now time.Time) error {
		var err error
		result.RevokedTokens, err = revokeAll(ctx, b, now)
		if err != nil {
			return err
		}
		b.Events[len(b.Events)-1].Reason = reason
		for _, sess := range b.Sessions {
			if sess.ExpiresAt.After(now) {
				sess.ExpiresAt = now
			}
			result.Sessions++
		}
		for _, task := range b.Tasks {
			closedAttempt := false
			for i := range task.Attempts {
				attempt := &task.Attempts[i]
				if attempt.EndedAt == nil {
					attempt.EndedAt = &now
					attempt.Reason = reason
					closedAttempt = true
				}
			}
			if task.Lease == nil {
				if closedAttempt {
					task.Version++
					emit(b, now, "attempt_recovered", task.Key, "")
					b.Events[len(b.Events)-1].Reason = reason
				}
				continue
			}
			session := task.Lease.SessionID
			task.Lease = nil
			task.State = "ready"
			task.ReadyAt = now
			task.Version++
			result.Leases++
			emit(b, now, "lease_recovered", task.Key, session)
			b.Events[len(b.Events)-1].Reason = reason
		}
		emit(b, now, "database_recovered", "", "")
		b.Events[len(b.Events)-1].Reason = reason
		return nil
	})
	return result, err
}
