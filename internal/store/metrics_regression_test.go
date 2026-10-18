package store

import (
	"testing"
	"time"

	"agent-board/internal/core"
)

func participant(t *testing.T, f *fixture, owner, role string) core.Actor {
	t.Helper()
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: owner, Machine: "synthetic-" + owner, TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: f.id(), Role: role, RunID: f.id(), Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: p, SessionID: v.SessionID}
}

func TestStatsParticipantOwnsAttemptAndCurrentLease(t *testing.T) {
	f := newFixture(t)
	bob := participant(t, f, "bob", "developer")
	doneTask := f.task(t, "developer")
	claim := f.claim(t, bob, doneTask.Key)
	tokens := int64(42)
	_, err := f.s.Transition(testContext, bob, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: doneTask.Key, Generation: claim.Lease.Generation}, State: "done", TokensSpent: &tokens, Evidence: &core.Evidence{Commit: commitA, RunSummary: "done"}})
	if err != nil {
		t.Fatal(err)
	}
	active := f.task(t, "developer")
	f.claim(t, bob, active.Key)
	f.task(t, "developer") // No executor exists for this ready task.
	other, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	stat, err := other.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]core.StatsMetrics{}
	var summed core.StatsMetrics
	for _, g := range stat.Groups {
		groups[g.Key] = g.Metrics
		summed.TokensSpent += g.Metrics.TokensSpent
		summed.Attempts += g.Metrics.Attempts
		summed.FirstCompletions += g.Metrics.FirstCompletions
		summed.CurrentQueue.Ready += g.Metrics.CurrentQueue.Ready
		summed.CurrentQueue.InProgress += g.Metrics.CurrentQueue.InProgress
	}
	if m := groups["bob"]; m.TokensSpent != 42 || m.Attempts != 1 || m.FirstCompletions != 1 || m.CurrentQueue.InProgress != 1 {
		t.Fatalf("executor facts lost: %+v", groups)
	}
	if groups[""].CurrentQueue.Ready != 1 || groups["alice"].TokensSpent != 0 || groups["alice"].FirstCompletions != 0 || groups["alice"].CurrentQueue != (core.QueueSummary{}) {
		t.Fatalf("unassigned queue attributed to epic owner: %+v", groups)
	}
	if summed.TokensSpent != stat.Totals.TokensSpent || summed.Attempts != stat.Totals.Attempts || summed.FirstCompletions != stat.Totals.FirstCompletions || summed.CurrentQueue != stat.Totals.CurrentQueue {
		t.Fatalf("groups do not sum to totals: %+v %+v", summed, stat.Totals)
	}
	page, err := other.Events(testContext, f.coord, core.EventsInput{TaskKey: doneTask.Key})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range page.Events {
		if event.Type == "task_done" && (event.Data == nil || event.Data.Owner != "bob") {
			t.Fatalf("persisted completion has wrong participant: %+v", event)
		}
	}
}

func TestStatsClosedAttemptParticipantForReleaseExpiryAndRecovery(t *testing.T) {
	for _, mode := range []string{"release", "expiry", "recovery"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			bob := participant(t, f, "bob", "developer")
			task := f.task(t, "developer")
			claim := f.claim(t, bob, task.Key)
			tokens := int64(42)
			_, err := f.s.Comment(testContext, bob, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "progress", TokensSpentDelta: &tokens})
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "release":
				_, err = f.s.Release(testContext, bob, core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "handoff"})
			case "expiry":
				f.clock.advance(time.Minute)
			case "recovery":
				_, err = f.s.RecoverState(testContext, "restore")
				f.coord = participant(t, f, "alice", "coordinator")
			}
			if err != nil {
				t.Fatal(err)
			}
			stat, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Second), GroupBy: "owner"})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, g := range stat.Groups {
				if g.Key == "bob" {
					found = g.Metrics.Attempts == 1 && g.Metrics.TokensSpent == 42 && g.Metrics.CurrentQueue == (core.QueueSummary{})
				}
				if g.Key == "alice" && (g.Metrics.Attempts != 0 || g.Metrics.TokensSpent != 0) {
					t.Fatal("closed attempt attributed to coordinator")
				}
			}
			if !found || stat.Totals.TokensSpent != 42 || stat.Totals.Attempts != 1 {
				t.Fatalf("closed attempt participant lost: %+v", stat)
			}
		})
	}
}

func TestCancelledBlockedSpanClipsAndRestartsOnRestore(t *testing.T) {
	f := newFixture(t)
	start := f.clock.read()
	source, target := f.task(t, "analyst"), f.task(t, "developer")
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "blocks"})
	if err != nil {
		t.Fatal(err)
	}
	transition := func(state string) {
		t.Helper()
		_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: target.Key, Reason: "change scope"}, State: state})
		if err != nil {
			t.Fatal(err)
		}
	}
	f.clock.advance(10 * time.Second)
	transition("cancelled")
	f.clock.advance(20 * time.Second)
	for _, tc := range []struct{ from, to, want int }{{0, 30, 10}, {0, 5, 5}, {5, 20, 5}, {10, 30, 0}, {20, 30, 0}} {
		stat, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: start.Add(time.Duration(tc.from) * time.Second), To: start.Add(time.Duration(tc.to) * time.Second), GroupBy: "role"})
		if err != nil || stat.Totals.BlockedDurationSeconds != float64(tc.want) {
			t.Fatalf("window %+v: duration=%v err=%v", tc, stat.Totals.BlockedDurationSeconds, err)
		}
	}
	if !f.get(t, target.Key).Blocked {
		t.Fatal("terminal metrics weakened dependency gate predicate")
	}
	transition("ready")
	f.clock.advance(5 * time.Second)
	transition("cancelled")
	f.clock.advance(10 * time.Second)
	other, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	stat, err := other.Stats(testContext, f.coord, core.StatsInput{From: start, To: f.clock.read(), GroupBy: "owner"})
	if err != nil || stat.Totals.BlockedDurationSeconds != 15 {
		t.Fatalf("restored span not closed or persisted: %+v %v", stat, err)
	}
}
