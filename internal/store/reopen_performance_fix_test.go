package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"testing"
	"time"
)

// Exercise the parent reopen both alone and after a verifier has already
// reopened the same (or another) epic in this candidate.
func performanceFixClosedParents(t *testing.T, kind string) (*fixture, core.CreateInput, []core.Task) {
	t.Helper()
	f := newFixture(t)
	in := core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Role: "developer", Title: "new work", Description: "implement"}
	if kind != "developer" {
		dev := f.actor(t, "developer")
		target := f.task(t, "developer")
		f.finish(t, dev, f.claim(t, dev, target.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked"}), "")
		in.Role, in.TargetKey, in.TargetCommit = "reviewer", target.Key, commitA
	}
	f.closeEpic(t, f.epic.Key)
	closed := []core.Task{f.get(t, f.epic.Key)}
	if kind == "verifier_cross_epic" {
		parent := f.newEpic(t)
		f.closeEpic(t, parent.Key)
		closed = append(closed, f.get(t, parent.Key))
		in.EpicKey = parent.Key
	}
	return f, in, closed
}

func performanceFixSQLTask(t *testing.T, f *fixture, key string) core.Task {
	t.Helper()
	var raw []byte
	if err := f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, key).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var task core.Task
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatal(err)
	}
	return task
}

func performanceFixRestart(t *testing.T, f *fixture) {
	t.Helper()
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := OpenExisting(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f.s = s
	t.Cleanup(func() { s.Close() })
}

func performanceFixAssertReopened(t *testing.T, f *fixture, closed []core.Task) {
	t.Helper()
	for _, old := range closed {
		cached, persisted := f.get(t, old.Key), performanceFixSQLTask(t, f, old.Key)
		if cached.State != "open" || persisted.State != "open" || cached.Version != old.Version+1 || persisted.Version != old.Version+1 {
			t.Fatalf("reopen %s: cache=%s/%d SQL=%s/%d want=open/%d", old.Key, cached.State, cached.Version, persisted.State, persisted.Version, old.Version+1)
		}
		var count int
		if err := f.s.db.QueryRow(`SELECT count(*) FROM events WHERE json_extract(data,'$.type')='epic_reopened' AND json_extract(data,'$.task_key')=?`, old.Key).Scan(&count); err != nil || count != 1 {
			t.Fatalf("reopen audit %s count=%d err=%v", old.Key, count, err)
		}
	}
}

func TestPerformanceFixCreateReopenDurableAndReplay(t *testing.T) {
	for _, kind := range []string{"developer", "verifier_same_epic", "verifier_cross_epic"} {
		t.Run(kind, func(t *testing.T) {
			f, in, closed := performanceFixClosedParents(t, kind)
			worker := f.actor(t, in.Role)
			created, err := f.s.Create(testContext, f.coord, in)
			if err != nil {
				t.Fatal(err)
			}
			performanceFixAssertReopened(t, f, closed)
			want := performanceExtraJSON(t, created)
			performanceExtraPoison(&created)
			f.clock.advance(time.Second)
			durable := performanceExtraDump(t, f)
			replay, err := f.s.Create(testContext, f.coord, in)
			if err != nil || performanceExtraJSON(t, replay) != want || performanceExtraDump(t, f) != durable {
				t.Fatal("create replay renewed a session, changed bytes or repeated audit", err)
			}
			performanceFixRestart(t, f)
			performanceFixAssertReopened(t, f, closed)
			replay, err = f.s.Create(testContext, f.coord, in)
			if err != nil || performanceExtraJSON(t, replay) != want || performanceExtraDump(t, f) != durable {
				t.Fatal("restart lost exact durable replay", err)
			}
			claim, err := f.s.Next(testContext, worker, core.NextInput{RequestID: f.id(), TaskKey: replay.Key})
			if err != nil || claim.Task == nil || claim.Task.Key != replay.Key {
				t.Fatal("durable reopened epic did not supply its child after restart", claim, err)
			}
		})
	}
}

func TestPerformanceFixRejectedCreateKeepsDoneEpic(t *testing.T) {
	for _, kind := range []string{"developer", "verifier_same_epic", "verifier_cross_epic"} {
		for _, rejection := range []string{"area", "metadata"} {
			t.Run(kind+"/"+rejection, func(t *testing.T) {
				f, in, closed := performanceFixClosedParents(t, kind)
				if rejection == "area" {
					in.Areas = []string{"../outside"}
				} else {
					in.Complexity = "invalid"
				}
				durable := performanceExtraDump(t, f)
				_, err := f.s.Create(testContext, f.coord, in)
				requireCode(t, err, "VALIDATION")
				if performanceExtraDump(t, f) != durable {
					t.Fatal("rejected create persisted business state, audit, sequence, replay or projection")
				}
				for _, old := range closed {
					if performanceExtraJSON(t, f.get(t, old.Key)) != performanceExtraJSON(t, old) {
						t.Fatal("rejected candidate contaminated retained parent", old.Key)
					}
				}
				if performanceExtraDump(t, f) != durable {
					t.Fatal("read after rejection persisted contaminated state")
				}
				performanceFixRestart(t, f)
				for _, old := range closed {
					if performanceExtraJSON(t, f.get(t, old.Key)) != performanceExtraJSON(t, old) {
						t.Fatal("restart changed rejected parent", old.Key)
					}
				}
				if performanceExtraDump(t, f) != durable {
					t.Fatal("restart after rejection changed durable state")
				}
				in.Areas, in.Complexity = nil, ""
				if _, err := f.s.Create(testContext, f.coord, in); err != nil {
					t.Fatal("rejection reserved request_id", err)
				}
				performanceFixAssertReopened(t, f, closed)
			})
		}
	}
}

func TestPerformanceFixRejectedCreateCommitsOnlyExpiry(t *testing.T) {
	f, in, closed := performanceFixClosedParents(t, "developer")
	f.epic = f.newEpic(t)
	worker := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, worker, task.Key)
	readCoordinator := func() string {
		var raw string
		if err := f.s.db.QueryRow(`SELECT data FROM sessions WHERE key=?`, f.coord.SessionID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	coordBefore := readCoordinator()
	var replaysBefore, sequenceBefore int64
	if err := f.s.db.QueryRow(`SELECT count(*) FROM idempotency`).Scan(&replaysBefore); err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.QueryRow(`SELECT sequence FROM metadata WHERE id=1`).Scan(&sequenceBefore); err != nil {
		t.Fatal(err)
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	in.Areas = []string{"../outside"}
	f.clock.advance(time.Minute) // Equality is already expired, independently of Create.
	_, err = f.s.Create(testContext, f.coord, in)
	requireCode(t, err, "VALIDATION")
	if performanceExtraJSON(t, f.get(t, closed[0].Key)) != performanceExtraJSON(t, closed[0]) {
		t.Fatal("expiry boundary retained rejected parent reopen")
	}
	expired := performanceFixSQLTask(t, f, task.Key)
	if expired.Lease != nil || expired.State != "ready" || expired.Generation != claim.Lease.Generation || len(expired.Attempts) != 1 || expired.Attempts[0].EndedAt == nil || expired.Attempts[0].Reason != "lease_expired" {
		t.Fatal("allowed expiry was not committed independently", expired)
	}
	var replaysAfter, sequenceAfter int64
	if err = f.s.db.QueryRow(`SELECT count(*) FROM idempotency`).Scan(&replaysAfter); err != nil {
		t.Fatal(err)
	}
	if err = f.s.db.QueryRow(`SELECT sequence FROM metadata WHERE id=1`).Scan(&sequenceAfter); err != nil {
		t.Fatal(err)
	}
	if replaysAfter != replaysBefore || sequenceAfter != sequenceBefore || readCoordinator() != coordBefore {
		t.Fatal("rejected candidate renewed coordinator session or persisted replay/sequence")
	}
	events, err := f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: page.HighWater, Limit: 200})
	if err != nil || len(events.Events) != 1 || events.Events[0].Type != "lease_expired" || events.Events[0].TaskKey != task.Key {
		t.Fatal("rejected create leaked audit beyond the allowed expiry", events, err)
	}
}
