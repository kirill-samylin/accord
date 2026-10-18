package store

import (
	"context"
	"errors"
	"fmt"
	"modernc.org/sqlite"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-board/internal/core"
)

func schema3PerformanceFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	dropPerformanceFixture(t, f.s)
	if _, err := f.s.db.Exec(`PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	return f
}
func TestPerformanceMigrationAtomicBackupAndBytes(t *testing.T) {
	f := schema3PerformanceFixture(t)
	db, err := openReadOnly(testContext, f.path)
	if err != nil {
		t.Fatal(err)
	}
	var rawEvent, rawReplay string
	if err = db.QueryRow(`SELECT data FROM events ORDER BY cursor DESC LIMIT 1`).Scan(&rawEvent); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT response FROM idempotency ORDER BY key LIMIT 1`).Scan(&rawReplay); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version int
	if err = s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 4 {
		t.Fatal(version, err)
	}
	var afterEvent, afterReplay string
	if err = s.db.QueryRow(`SELECT data FROM events ORDER BY cursor DESC LIMIT 1`).Scan(&afterEvent); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT response FROM idempotency ORDER BY key LIMIT 1`).Scan(&afterReplay); err != nil {
		t.Fatal(err)
	}
	if rawEvent != afterEvent || rawReplay != afterReplay {
		t.Fatal("migration rewrote authoritative bytes")
	}
	backups, err := filepath.Glob(f.path + ".migrations/schema3-*.db")
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	info, err := os.Stat(backups[0])
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
	if err = checkSnapshot(testContext, backups[0]); err != nil {
		t.Fatal(err)
	}
	original, err := openReadOnly(testContext, backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	if err = original.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 3 {
		t.Fatal(version, err)
	}
	var dirty bool
	if err = s.db.QueryRow(`SELECT dirty FROM projection_control`).Scan(&dirty); err != nil || dirty {
		t.Fatal(dirty, err)
	}
	directory := t.TempDir()
	if err = os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = Backup(testContext, f.path, filepath.Join(directory, "snapshot.db")); err != nil {
		t.Fatal(err)
	}
}
func TestPerformanceMigrationDDLFailureAndCancelLeaveSchema3(t *testing.T) {
	for _, mode := range []string{"ddl", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := schema3PerformanceFixture(t)
			s, err := OpenExisting(f.path, core.Config{}, f.clock.read)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if mode == "ddl" {
				if _, err = s.db.Exec(`CREATE TABLE stats_prefix(conflict INTEGER)`); err != nil {
					t.Fatal(err)
				}
			}
			ctx := testContext
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err = migrate(ctx, s.db, f.path, core.Config{}); err == nil {
				t.Fatal("failed migration succeeded")
			}
			var version, count int
			if err = s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 3 {
				t.Fatal(version, err)
			}
			if err = s.db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='events_task_cursor'`).Scan(&count); err != nil || count != 0 {
				t.Fatal("partial schema4 DDL", count, err)
			}
		})
	}
}

func TestPerformanceMigrationStorageFullRollsBack(t *testing.T) {
	f := newFixture(t)
	events := make([]core.Event, 2000)
	at := f.clock.read()
	for i := range events {
		events[i] = core.Event{Type: "commented", At: at.Add(time.Duration(i) * time.Nanosecond), Data: &core.EventData{Role: "developer", Owner: "synthetic"}}
	}
	seedPerformanceEvents(t, f, events)
	dropPerformanceFixture(t, f.s)
	if _, err := f.s.db.Exec(`PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	s, err := OpenExisting(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var pages int64
	if err = s.db.QueryRow(`PRAGMA page_count`).Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(fmt.Sprintf(`PRAGMA max_page_count=%d`, pages)); err != nil {
		t.Fatal(err)
	}
	err = migrate(testContext, s.db, f.path, core.Config{})
	var full *sqlite.Error
	if !errors.As(err, &full) || full.Code()&255 != 13 {
		t.Fatal("expected actual SQLITE_FULL", err)
	}
	var version, count int
	if err = s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 3 {
		t.Fatal(version, err)
	}
	if err = s.db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='projection_control'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial projection after storage failure", count, err)
	}
	backups, err := filepath.Glob(f.path + ".migrations/schema3-*.db")
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	if err = checkSnapshot(testContext, backups[0]); err != nil {
		t.Fatal(err)
	}
}
