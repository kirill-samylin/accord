package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"modernc.org/sqlite"
)

// startupRetry retries SQLITE_BUSY (including BUSY_RECOVERY) only before a
// transaction has begun, or for an explicitly idempotent startup read/pragma.
// It must never wrap a business operation, DDL transaction body or Commit:
// uncertain commits are recovered by durable replay, not by blind retries.
func startupRetry(ctx context.Context, fn func(context.Context) error) error {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		if err := bounded.Err(); err != nil {
			return err
		}
		err := fn(bounded)
		if err == nil {
			return nil
		}
		var driverErr *sqlite.Error
		if !errors.As(err, &driverErr) || driverErr.Code()&255 != 5 {
			return err
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-bounded.Done():
			timer.Stop()
			return bounded.Err()
		case <-timer.C:
		}
	}
}
func beginImmediate(ctx context.Context, conn *sql.Conn) (*sql.Tx, error) {
	var tx *sql.Tx
	err := startupRetry(ctx, func(attempt context.Context) error {
		var err error
		// The caller's context owns the transaction lifetime. The bounded retry
		// context only guards admission, not a transaction after it succeeds.
		if err = attempt.Err(); err != nil {
			return err
		}
		tx, err = conn.BeginTx(ctx, nil)
		return err
	})
	return tx, err
}
