package store

import (
	"math"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestAttemptTokensFinalTotalReplayAndReworkStats(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, d, task.Key)
	delta := int64(11)
	in := core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Branch: "feature/c"}, Text: "progress", TokensSpentDelta: &delta}
	_, err := f.s.Comment(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Comment(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	bad := int64(10)
	complete := core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &core.Evidence{Commit: commitA, RunSummary: "passed"}, TokensSpent: &bad}
	_, err = f.s.Transition(testContext, d, complete)
	requireCode(t, err, "VALIDATION")
	good := int64(15)
	complete.RequestID = f.id()
	complete.TokensSpent = &good
	f.clock.advance(10 * time.Second)
	done, err := f.s.Transition(testContext, d, complete)
	if err != nil {
		t.Fatal(err)
	}
	if *done.Attempts[0].TokensSpent != 15 || done.Attempts[0].Branch != "feature/c" {
		t.Fatal("final total or branch lost")
	}
	_, err = f.s.Transition(testContext, d, complete)
	if err != nil {
		t.Fatal(err)
	}
	defect := f.task(t, "coordinator")
	c := f.claim(t, f.coord, defect.Key)
	f.finish(t, f.coord, c, core.Evidence{Summary: "rework needed"}, "")
	reopen := core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Reason: "fix"}, State: "ready", EvidenceRef: defect.Key}
	returned, err := f.s.Transition(testContext, f.coord, reopen)
	if err != nil || returned.ReturnsCount != 1 {
		t.Fatal("return count")
	}
	_, err = f.s.Transition(testContext, f.coord, reopen)
	if err != nil {
		t.Fatal(err)
	}
	f.finish(t, d, f.claim(t, d, task.Key), core.Evidence{Commit: commitB, RunSummary: "passed"}, "")
	stat, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "role"})
	if err != nil {
		t.Fatal(err)
	}
	m := stat.Totals
	if m.FirstCompletions != 2 || m.ReworkCompletions != 1 || m.Returns != 1 || m.TokensSpent != 15 || m.Attempts != 3 || m.MissingTokens != 2 || m.WallDurationSeconds != 10 || !stat.IncompleteData {
		t.Fatalf("stats %+v incomplete=%v", m, stat.IncompleteData)
	}
	if math.Abs(m.MissingTokensShare-2.0/3) > 0.00001 {
		t.Fatal("missing tokens share")
	}
	events, err := f.s.Events(testContext, f.coord, core.EventsInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	deltaEvents, corrections := 0, 0
	for _, e := range events.Events {
		if e.Data != nil && e.Data.TokensSpentDelta != nil {
			deltaEvents++
			if *e.Data.TokensSpentDelta != 11 {
				t.Fatal("delta audit")
			}
		}
		if e.Data != nil && e.Data.CorrectionDelta != nil {
			corrections++
			if *e.Data.CorrectionDelta != 4 || *e.Data.TokensSpentTotal != 15 {
				t.Fatal("final correction audit")
			}
		}
	}
	if deltaEvents != 1 || corrections != 1 {
		t.Fatal("replay duplicated token audit")
	}
	// [from,to) excludes all boundary events.
	empty, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(-time.Minute), GroupBy: "epic"})
	if err != nil || empty.Totals.Attempts != 0 {
		t.Fatal("half-open window failed")
	}
}
func TestTokensZeroNegativeOverflowAndExpiredLease(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, d, task.Key)
	delta := int64(-1)
	in := core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "x", TokensSpentDelta: &delta}
	_, err := f.s.Comment(testContext, d, in)
	requireCode(t, err, "VALIDATION")
	// One attempt is bounded to maxAttemptTokens (10^12): huge values are a
	// client error, never a counter that later breaks statistics.
	delta = math.MaxInt64
	in.RequestID = f.id()
	_, err = f.s.Comment(testContext, d, in)
	requireCode(t, err, "VALIDATION")
	delta = maxAttemptTokens
	in.RequestID = f.id()
	_, err = f.s.Comment(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	delta = maxAttemptTokens + 1
	in.RequestID = f.id()
	_, err = f.s.Comment(testContext, d, in)
	requireCode(t, err, "VALIDATION")
	task = f.get(t, task.Key)
	if *task.Attempts[0].TokensSpent != maxAttemptTokens {
		t.Fatal("failed token mutation persisted")
	}
	// T022-R2-10c: the bound applies to the reported delta only; an attempt
	// whose total already exceeds it (like legacy data) keeps reporting.
	for _, step := range []int64{maxAttemptTokens, 0} {
		delta = step
		in.RequestID = f.id()
		if _, err = f.s.Comment(testContext, d, in); err != nil {
			t.Fatal("delta on an attempt above the bound", step, err)
		}
	}
	if task = f.get(t, task.Key); *task.Attempts[0].TokensSpent != 2*maxAttemptTokens {
		t.Fatal("deltas not accumulated", *task.Attempts[0].TokensSpent)
	}
	f.clock.advance(time.Minute)
	in.RequestID = f.id()
	_, err = f.s.Comment(testContext, d, in)
	requireCode(t, err, "LEASE_LOST")
}
func TestBlockedDurationAndLegacyIncomplete(t *testing.T) {
	f := newFixture(t)
	source := f.task(t, "analyst")
	target := f.task(t, "developer")
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "blocks"})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(12 * time.Second)
	a := f.actor(t, "analyst")
	f.finish(t, a, f.claim(t, a, source.Key), core.Evidence{Artifact: []string{"a"}, Summary: "done"}, "")
	in := core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "owner"}
	s, err := f.s.Stats(testContext, f.coord, in)
	if err != nil || s.Totals.BlockedDurationSeconds != 12 {
		t.Fatalf("blocked duration %+v %v", s.Totals, err)
	}
	// Insert a historical payload exactly as schema1 stored it; no fabricated
	// role/owner/timing is derived from today's task.
	_, err = f.s.db.Exec(`INSERT INTO events(data) VALUES(?)`, `{"type":"task_done","task_key":"T-historical","at":"2026-10-18T12:00:00Z"}`)
	if err != nil {
		t.Fatal(err)
	}
	s, err = f.s.Stats(testContext, f.coord, in)
	if err != nil || !s.IncompleteData || s.IncompleteEvents != 1 || s.Totals.FirstCompletions != 1 {
		t.Fatalf("legacy stats %+v %v", s, err)
	}
}

func TestExplicitZeroTokensAreKnownAndStatsSurvivesStoreReopen(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, d, task.Key)
	zero := int64(0)
	done, err := f.s.Transition(testContext, d, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", TokensSpent: &zero, Evidence: &core.Evidence{Commit: commitA, RunSummary: "passed"}})
	if err != nil || done.Attempts[0].TokensSpent == nil || *done.Attempts[0].TokensSpent != 0 {
		t.Fatal("explicit zero became missing")
	}
	other, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	stat, err := other.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "epic"})
	if err != nil || stat.Totals.Attempts != 1 || stat.Totals.MissingTokens != 0 || stat.Totals.TokensSpent != 0 {
		t.Fatalf("zero after reopen %+v %v", stat, err)
	}
}

func TestEpicCompletionAttributedToOwnEpic(t *testing.T) {
	f := newFixture(t)
	_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"})
	if err != nil {
		t.Fatal(err)
	}
	stat, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "epic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stat.Groups) != 1 || stat.Groups[0].Key != f.epic.Key || stat.Groups[0].Metrics.EpicCompletions != 1 {
		t.Fatalf("epic attribution %+v", stat.Groups)
	}
}
