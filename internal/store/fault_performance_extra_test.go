package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func performanceExtraDB(t *testing.T, f *fixture) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", f.path+"?_txlock=immediate&_pragma=busy_timeout(50)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPerformanceExtraCandidateFailureIsAtomicAndRetryable(t *testing.T) {
	for _, table := range []string{"idempotency", "tasks", "sessions", "events", "projection_control"} {
		t.Run(table, func(t *testing.T) {
			f := newFixture(t)
			dev := f.actor(t, "developer")
			task := f.task(t, "developer")
			claim := f.claim(t, dev, task.Key)
			db := performanceExtraDB(t, f)
			action := "INSERT"
			if table == "projection_control" {
				action = "UPDATE"
			}
			if _, err := db.Exec(`CREATE TRIGGER performance_extra_fault BEFORE ` + action + ` ON ` + table + ` BEGIN SELECT RAISE(ABORT,'performance-extra-fault'); END`); err != nil {
				t.Fatal(err)
			}
			before, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: task.Key})
			if err != nil {
				t.Fatal(err)
			}
			f.clock.advance(5 * time.Second) // Candidate would renew both leases.
			durable := performanceExtraDump(t, f)
			delta := int64(7)
			in := core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "one committed comment", TokensSpentDelta: &delta}
			if _, err = f.s.Comment(testContext, dev, in); err == nil || !strings.Contains(err.Error(), "performance-extra-fault") {
				t.Fatal("injected failure not reached", err)
			}
			if performanceExtraDump(t, f) != durable {
				t.Fatal("failure persisted partial entity/session/lease/audit/replay/projection")
			}
			after, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: task.Key})
			if err != nil || performanceExtraJSON(t, after) != performanceExtraJSON(t, before) {
				t.Fatal("failed candidate leaked into next cached read", after, err)
			}
			if _, err = db.Exec(`DROP TRIGGER performance_extra_fault`); err != nil {
				t.Fatal(err)
			}
			committed, err := f.s.Comment(testContext, dev, in)
			if err != nil || len(committed.Comments) != 1 || committed.Attempts[0].TokensDelta != delta || committed.Lease == nil || !committed.Lease.ExpiresAt.After(before.Lease.ExpiresAt) {
				t.Fatal("same request could not retry once after rollback", committed, err)
			}
			durable = performanceExtraDump(t, f)
			replay, err := f.s.Comment(testContext, dev, in)
			if err != nil || performanceExtraJSON(t, replay) != performanceExtraJSON(t, committed) || performanceExtraDump(t, f) != durable {
				t.Fatal("valid retry replay changed state or emitted another event", err)
			}
		})
	}
}

func TestPerformanceExtraCanceledGateOwnerAndConcurrentClose(t *testing.T) {
	f := newFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var enterOnce, releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	// The public injected clock is read after BeginTx; block there to establish
	// an owned operation without reaching into the gate or retained connection.
	s, err := OpenExisting(f.path, core.Config{}, func() time.Time {
		enterOnce.Do(func() { close(entered); <-release })
		return f.clock.read()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	durable := performanceExtraDump(t, f)
	ctx, cancel := context.WithCancel(testContext)
	defer cancel()
	owner := make(chan error, 1)
	go func() {
		_, err := s.Get(ctx, f.coord, core.GetInput{TaskKey: f.epic.Key})
		owner <- err
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("operation did not reach injected clock")
	}
	canceled, cancelWaiting := context.WithCancel(testContext)
	cancelWaiting()
	waiting := make(chan error, 16)
	for range 16 {
		go func() {
			_, err := s.Get(canceled, f.coord, core.GetInput{TaskKey: f.epic.Key})
			waiting <- err
		}()
	}
	for range 16 {
		select {
		case err := <-waiting:
			if !errors.Is(err, context.Canceled) {
				t.Fatal("canceled caller did not leave occupied gate", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("canceled caller stuck behind gate owner")
		}
	}
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case err := <-closed:
		t.Fatal("Close returned before owned operation finished", err)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-owner:
		// database/sql may race its asynchronous rollback with our next query.
		if !errors.Is(err, context.Canceled) && !errors.Is(err, sql.ErrTxDone) {
			t.Fatal("canceled owner returned a successful candidate", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("owner failed to leave canceled transaction")
	}
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close left retained writer/pool blocked")
	}
	if performanceExtraDump(t, f) != durable {
		t.Fatal("cancellation published durable candidate")
	}
	if _, err = s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key}); err == nil {
		t.Fatal("closed Store accepted new work")
	}
	if _, err = f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "epic", Title: "after close"}); err != nil {
		t.Fatal("canceled Store left SQLite writer occupied", err)
	}
}

func TestPerformanceExtraCanceledBeforeAdmissionDoesNotReserveReplay(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, dev, task.Key)
	in := core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "retry canceled request"}
	durable := performanceExtraDump(t, f)
	ctx, cancel := context.WithCancel(testContext)
	cancel()
	for range 16 {
		if _, err := f.s.Comment(ctx, dev, in); !errors.Is(err, context.Canceled) {
			t.Fatal("canceled context admitted before gate/begin", err)
		}
	}
	if performanceExtraDump(t, f) != durable {
		t.Fatal("canceled admission persisted mutation or replay")
	}
	got, err := f.s.Comment(testContext, dev, in)
	if err != nil || len(got.Comments) != 1 {
		t.Fatal("canceled admission left gate or replay reserved", got, err)
	}
}

// Compare durable logical contents, including projection/control/replay tables;
// trigger DDL itself is intentionally excluded. No cache fields are consulted.
func performanceExtraDump(t *testing.T, f *fixture) string {
	t.Helper()
	db := performanceExtraDB(t, f)
	defer db.Close()
	tables, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for tables.Next() {
		var name string
		if err := tables.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := tables.Err(); err != nil {
		t.Fatal(err)
	}
	tables.Close()
	out := map[string][]string{}
	for _, name := range names {
		// Names originate from the schema; quote identifiers even in this private DB.
		rows, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(name, `"`, `""`) + `"`)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			out[name] = append(out[name], string(raw))
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		sort.Strings(out[name])
	}
	return performanceExtraJSON(t, out)
}
