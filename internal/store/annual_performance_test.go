package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"agent-board/internal/core"
)

// This deliberately expensive developer diagnostic is opt-in. Runtime/VPS
// acceptance belongs to the independent tester; this fixture is synthetic SQL.
func TestPerformanceAnnualMigrationAndOrdinaryAPIs(t *testing.T) {
	if os.Getenv("ACCORD_ANNUAL_TEST") != "1" {
		t.Skip("set ACCORD_ANNUAL_TEST=1 for 200k-event streaming migration diagnostic")
	}
	f := newFixture(t)
	start := f.clock.read().Add(-365 * 24 * time.Hour)
	seed, err := f.s.db.BeginTx(testContext, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Rollback()
	if _, err = seed.Exec(`DELETE FROM events`); err != nil {
		t.Fatal(err)
	}
	insert, err := seed.Prepare(`INSERT INTO events(cursor,data) VALUES(?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer insert.Close()
	fixtureHash := sha256.New()
	for i := 0; i < 200000; i++ {
		at := start.Add(time.Duration(i) * time.Minute)
		d := &core.EventData{Kind: "task", Role: []string{"developer", "reviewer", "tester"}[i%3], Owner: fmt.Sprintf("owner-%d", i%20), EpicKey: f.epic.Key, NewState: "ready"}
		e := core.Event{Type: "commented", TaskKey: fmt.Sprintf("T-annual-%d", i/64%2000), At: at, Data: d}
		switch i % 64 {
		case 0:
			e.Type = "task_created"
			d.Blocked = true
		case 1:
			e.Type = "blocked_changed"
		case 2:
			e.Type = "task_done"
			d.NewState = "done"
			d.FirstDone = true
			d.AttemptEnded = true
			value := int64(i % 10000)
			d.TokensSpent = &value
			claim, done := at.Add(-time.Hour), at
			d.FirstClaimAt, d.FirstDoneAt = &claim, &done
		case 3:
			e.Type = "task_ready"
			d.Returns = 1
		case 4:
			e.Type = "lease_expired"
			d.AttemptEnded = true
		case 5:
			e.Type = "area_warning"
		}
		e.Cursor = int64(i + 1)
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = insert.Exec(e.Cursor, string(raw)); err != nil {
			t.Fatal(err)
		}
		fixtureHash.Write(raw)
	}
	insert.Close()
	if err = seed.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Logf("annual raw events sha256=%x", fixtureHash.Sum(nil))
	tx, err := f.s.db.BeginTx(testContext, nil)
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO tasks VALUES(?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("T-%d", i+100)
		task := core.Task{Key: key, Type: "task", Role: "developer", Title: fmt.Sprintf("annual task %d", i), Description: "synthetic diagnostic", EpicKey: f.epic.Key, CreatedAt: &start, ReadyAt: start, State: "ready", Priority: 3, Version: 1, Requester: f.p.ID}
		raw, err := json.Marshal(task)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = stmt.Exec(key, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	stmt.Close()
	if _, err = tx.Exec(`UPDATE metadata SET sequence=2100 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	dropPerformanceFixture(t, f.s)
	if _, err = f.s.db.Exec(`PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	runtime.GC()
	debug.FreeOSMemory()
	begun := time.Now()
	s, err := Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	migration := time.Since(begun)
	f.s = s
	defer s.Close()
	var control projectionControl
	_, err = transact(s, testContext, func(b *board, _ time.Time) (struct{}, error) {
		var err error
		control, err = readProjection(testContext, b.tx)
		return struct{}{}, err
	})
	if err != nil || control.Compatibility || control.Cursor != 200000 || control.Dirty {
		t.Fatal(control, err)
	}
	t.Logf("synthetic migration200k=%s, projection compatibility=false", migration)
	s.Close()
	begun = time.Now()
	s, err = Open(f.path, core.Config{}, f.clock.read)
	restart := time.Since(begun)
	if err != nil {
		t.Fatal(err)
	}
	f.s = s
	defer s.Close()
	t.Logf("populated schema4 Open restart=%s", restart)
	in := core.StatsInput{From: start.Add(15 * time.Hour), To: start.Add(100 * 24 * time.Hour), GroupBy: "owner"}
	// Public methods perform real auth/config/expiry; no cache is substituted.
	measure := func(name string, fn func() error) {
		t.Helper()
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		begun := time.Now()
		if err := fn(); err != nil {
			t.Fatal(name, err)
		}
		elapsed := time.Since(begun)
		runtime.ReadMemStats(&after)
		t.Logf("%s wall=%s alloc_bytes=%d", name, elapsed, after.TotalAlloc-before.TotalAlloc)
	}
	measure("cold get", func() error { _, err := s.Get(testContext, f.coord, core.GetInput{TaskKey: "T-100"}); return err })
	measure("warm get", func() error { _, err := s.Get(testContext, f.coord, core.GetInput{TaskKey: "T-100"}); return err })
	measure("tasks20", func() error { _, err := s.ListTasks(testContext, f.coord, core.ListInput{Limit: 20}); return err })
	measure("events20", func() error {
		_, err := s.Events(testContext, f.coord, core.EventsInput{TaskKey: "T-annual-1", Types: []string{"task_done", "lease_expired"}, Limit: 20})
		return err
	})
	measure("stats20owners", func() error { _, err := s.Stats(testContext, f.coord, in); return err })
	var created core.Task
	measure("create", func() error {
		var err error
		created, err = s.Create(testContext, f.coord, core.CreateInput{RequestID: "annualcreate", Type: "task", EpicKey: f.epic.Key, Role: "developer", Title: "ordinary annual create", Description: "synthetic"})
		return err
	})
	text := "updated"
	measure("update", func() error {
		_, err := s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: "annualupdate", TaskKey: created.Key}, Description: &text})
		return err
	})
	actor := f.actor(t, "developer")
	var claim core.Claim
	measure("next", func() error {
		var err error
		claim, err = s.Next(testContext, actor, core.NextInput{RequestID: "annualnext", TaskKey: created.Key})
		return err
	})
	lease := core.LeaseInput{RequestID: "annualcomment", TaskKey: created.Key, Generation: claim.Lease.Generation}
	measure("comment", func() error {
		_, err := s.Comment(testContext, actor, core.CommentInput{LeaseInput: lease, Text: "annual"})
		return err
	})
	measure("heartbeat", func() error {
		_, err := s.Heartbeat(testContext, actor, core.HeartbeatInput{RequestID: "annualheartbeat", TaskKey: created.Key, Generation: claim.Lease.Generation})
		return err
	})
	measure("link", func() error {
		_, err := s.Link(testContext, actor, core.LinkInput{RequestID: "annuallink", From: "T-100", To: created.Key, Type: "related", Generation: claim.Lease.Generation})
		return err
	})
	measure("note", func() error {
		_, err := s.Note(testContext, actor, core.NoteInput{RequestID: "annualnote", Action: "create", ScopeType: "role", ScopeID: "developer", SourceTaskKey: created.Key, Text: "annual synthetic"})
		return err
	})
	lease.RequestID = "annualrelease"
	lease.Reason = "diagnostic"
	measure("release", func() error { _, err := s.Release(testContext, actor, lease); return err })
	measure("transition", func() error {
		_, err := s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: "annualcancel", TaskKey: created.Key, Reason: "diagnostic"}, State: "cancelled"})
		return err
	})
	// Annual prefix correctness is independently compared with the old reducer.
	comparePerformanceStats(t, f, []core.StatsInput{in, {From: start, To: start.Add(time.Minute), GroupBy: "role"}, {From: start.Add(time.Minute), To: start.Add(3 * time.Minute), GroupBy: "epic"}}, false)
	s.Close()
	file, err := os.Open(f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, file)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("final synthetic DB sha256=%s bytes=%d", hex.EncodeToString(digest.Sum(nil)), size)
}
