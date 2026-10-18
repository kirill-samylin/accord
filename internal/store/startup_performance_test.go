package store

import (
	"agent-board/internal/core"
	"context"
	"errors"
	"testing"
	"time"
)

func TestPerformanceImmediateLockContextAndNoopVersion(t *testing.T) {
	f := newFixture(t)
	conn, err := f.s.db.Conn(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tx, err := beginImmediate(testContext, conn)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	other, err := f.s.db.Conn(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err = other.ExecContext(testContext, `PRAGMA busy_timeout=1`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(testContext, 20*time.Millisecond)
	defer cancel()
	if blocked, err := beginImmediate(ctx, other); err == nil {
		blocked.Rollback()
		t.Fatal("BeginTx did not hold the SQLite writer lock")
	}
	tx.Rollback()
	var before, after int64
	if err = other.QueryRowContext(testContext, `PRAGMA data_version`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key}); err != nil {
		t.Fatal(err)
	}
	if err = other.QueryRowContext(testContext, `PRAGMA data_version`).Scan(&after); err != nil || before != after {
		t.Fatalf("no-op changed external data_version %d -> %d: %v", before, after, err)
	}
	ordinary := errors.New("database is locked (261)")
	calls := 0
	if err = startupRetry(testContext, func(context.Context) error { calls++; return ordinary }); !errors.Is(err, ordinary) || calls != 1 {
		t.Fatal("retry classified an error string as SQLite BUSY", calls, err)
	}
}
