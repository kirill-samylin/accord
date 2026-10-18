package store

import (
	"agent-board/internal/core"
	"reflect"
	"testing"
)

func (f *fixture) newEpic(t *testing.T) core.Task {
	t.Helper()
	v, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "epic", Title: "Another epic"})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) closeEpic(t *testing.T, key string) {
	t.Helper()
	_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: key}, State: "done"})
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) get(t *testing.T, key string) core.Task {
	t.Helper()
	v, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) link(t *testing.T, from, to string) {
	t.Helper()
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: from, To: to, Type: "blocks"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoneConsumerLinkReconcilesAtomicallyAndReplaysOnce(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	consumer := f.task(t, "developer")
	before := f.finish(t, dev, f.claim(t, dev, consumer.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
	ep := f.epic
	f.closeEpic(t, ep.Key)
	f.epic = f.newEpic(t)
	source := f.task(t, "analyst")
	input := core.LinkInput{RequestID: f.id(), From: source.Key, To: consumer.Key, Type: "blocks"}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	// Force failure after domain reconciliation and replay insertion, while
	// persisting its audit event. Every change must roll back together.
	_, err = f.s.db.Exec(`CREATE TRIGGER reject_reopen BEFORE INSERT ON events WHEN instr(NEW.data,'"type":"epic_reopened"')>0 BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Link(testContext, f.coord, input); err == nil {
		t.Fatal("expected injected SQL failure")
	}
	if f.get(t, ep.Key).State != "done" || f.get(t, consumer.Key).Blocked {
		t.Fatal("failed transaction persisted link or reconciliation")
	}
	var saved int
	if err = f.s.db.QueryRow(`SELECT count(*) FROM idempotency WHERE key=?`, f.p.ID+"/"+f.coord.SessionID+"/"+input.RequestID).Scan(&saved); err != nil || saved != 0 {
		t.Fatal("failed mutation persisted replay response")
	}
	if _, err = f.s.db.Exec(`DROP TRIGGER reject_reopen`); err != nil {
		t.Fatal(err)
	}
	linked, err := f.s.Link(testContext, f.coord, input)
	if err != nil {
		t.Fatal(err)
	}
	if !linked.Blocked || f.get(t, ep.Key).State != "open" {
		t.Fatal("new unsatisfied gate left epic done")
	}
	after := f.get(t, consumer.Key)
	if after.State != "done" || !reflect.DeepEqual(after.Evidence, before.Evidence) || !reflect.DeepEqual(after.Attempts, before.Attempts) {
		t.Fatal("reconciliation rewrote historical completion")
	}
	replay, err := f.s.Link(testContext, f.coord, input)
	if err != nil || !reflect.DeepEqual(replay, linked) {
		t.Fatal("replay changed response")
	}
	events, err := f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: page.HighWater})
	if err != nil {
		t.Fatal(err)
	}
	// The completed consumer still invalidates the epic's gate, but does not
	// start an operational waiting interval until explicitly reopened.
	if len(events.Events) != 2 || events.Events[0].Type != "linked" || events.Events[1].Type != "epic_reopened" {
		t.Fatal("rollback/replay duplicated or lost atomic audit")
	}
	if events.Events[0].Data == nil || events.Events[0].Data.Blocked {
		t.Fatal("terminal consumer started an operational blocked interval")
	}
}

func TestReopenSourceCascadesAcrossDoneEpics(t *testing.T) {
	f := newFixture(t)
	analyst := f.actor(t, "analyst")
	source := f.task(t, "analyst")
	f.finish(t, analyst, f.claim(t, analyst, source.Key), core.Evidence{Artifact: []string{"decision"}, Summary: "resolved"}, "")
	epics := []string{f.epic.Key}
	previous := source.Key
	var history []core.Task
	for range 2 {
		f.epic = f.newEpic(t)
		epics = append(epics, f.epic.Key)
		consumer := f.task(t, "analyst")
		f.link(t, previous, consumer.Key)
		history = append(history, f.finish(t, analyst, f.claim(t, analyst, consumer.Key), core.Evidence{Artifact: []string{"decision"}, Summary: "resolved"}, ""))
		previous = consumer.Key
	}
	for _, key := range epics {
		f.closeEpic(t, key)
	}
	_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: source.Key, Reason: "defect"}, State: "ready", EvidenceRef: previous})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range epics {
		if f.get(t, key).State != "open" {
			t.Fatal("cross-epic invalidation did not propagate")
		}
		_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: key}, State: "done"})
		requireCode(t, err, "RULE_VIOLATION")
	}
	for _, original := range history {
		after := f.get(t, original.Key)
		if !after.Blocked || after.State != "done" || !reflect.DeepEqual(after.Attempts, original.Attempts) || !reflect.DeepEqual(after.Evidence, original.Evidence) {
			t.Fatal("cascading invalidation changed completed audit or ignored gate")
		}
	}
}

func TestImplicitReviewDependencyCycleFailsClosed(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
	check := f.verification(t, "reviewer", target.Key, commitA)
	reviewer := f.actor(t, "reviewer")
	f.finish(t, reviewer, f.claim(t, reviewer, check.Key), core.Evidence{VerifiedCommit: commitA, Findings: "checked"}, "approve")
	f.closeEpic(t, f.epic.Key)
	f.link(t, check.Key, target.Key)
	if !f.get(t, check.Key).Blocked || !f.get(t, target.Key).Blocked || f.get(t, f.epic.Key).State != "open" {
		t.Fatal("implicit cycle must invalidate gates without recursion overflow")
	}
}
