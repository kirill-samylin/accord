package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestManagedTerminalPreservesOrdinaryForeignOtherRunOtherGenerationAndDone(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	boundView, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *p.Run))
	if err != nil {
		t.Fatal(err)
	}
	bound := core.Actor{Principal: m.f.p, SessionID: boundView.SessionID}
	releaseTask := assignedTask(t, m.f, m.p)
	m.f.claim(t, bound, releaseTask.Key)
	newSession := func(principal core.Principal, runID string) core.Actor {
		t.Helper()
		view, e := m.f.s.Describe(testContext, principal, core.OpenSession{RequestID: m.f.id(), Role: "developer", RunID: runID, Machine: principal.Machine, ProfileID: m.p.ID, ProfileVersion: 1})
		if e != nil {
			t.Fatal(e)
		}
		return core.Actor{Principal: principal, SessionID: view.SessionID}
	}
	ordinary := newSession(m.f.p, p.Run.ID)
	otherRun := newSession(m.f.p, "other-run")
	otherGeneration := newSession(m.f.p, p.Run.ID)
	foreignToken, err := m.f.s.CreateToken(testContext, core.TokenInput{Owner: "foreign", Machine: "foreign", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	foreignPrincipal, err := m.f.s.Authenticate(testContext, foreignToken.Secret)
	if err != nil {
		t.Fatal(err)
	}
	foreign := newSession(foreignPrincipal, p.Run.ID)
	// Durable fixtures distinguish authoritative binding scope from untrusted
	// RunID and from records belonging to another owner/execution generation.
	for _, v := range []struct {
		actor      core.Actor
		run        string
		generation int64
	}{{otherRun, "other-run", 1}, {otherGeneration, p.Run.ID, 2}, {foreign, p.Run.ID, 1}} {
		if _, err = m.f.s.db.Exec(`UPDATE sessions SET data=json_set(data,'$.managed_run_id',?,'$.managed_generation',?) WHERE key=?`, v.run, v.generation, v.actor.SessionID); err != nil {
			t.Fatal(err)
		}
	}
	untouched := map[string]core.Actor{}
	for _, actor := range []core.Actor{ordinary, otherRun, otherGeneration, foreign} {
		task := assignedTask(t, m.f, m.p)
		m.f.claim(t, actor, task.Key)
		untouched[task.Key] = actor
	}
	doneView, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *p.Run))
	if err != nil {
		t.Fatal(err)
	}
	doneActor := core.Actor{Principal: m.f.p, SessionID: doneView.SessionID}
	doneTask := assignedTask(t, m.f, m.p)
	doneClaim := m.f.claim(t, doneActor, doneTask.Key)
	done := m.f.finish(t, doneActor, doneClaim, core.Evidence{Commit: commitA, RunSummary: "Verified native evidence"}, "")
	m.report(t, p, "started")
	m.report(t, p, "completed")
	for key, actor := range untouched {
		task, e := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: key})
		if e != nil || task.Lease == nil || task.Lease.SessionID != actor.SessionID || task.Attempts[0].EndedAt != nil {
			t.Fatal("unrelated lease released", key, e)
		}
		if sess := storedSession(t, m.f, actor); !sess.ExpiresAt.After(m.f.clock.read()) {
			t.Fatal("unrelated session expired", actor.SessionID)
		}
	}
	after, err := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: done.Key})
	if err != nil || after.State != "done" || after.Evidence.Commit != commitA || after.Generation != done.Generation || after.Version != done.Version {
		t.Fatal("completed evidence changed", err)
	}
}

func TestManagedTerminalLeaseMismatchRollsBackAllReleasesAndRunState(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	tasks := []core.Task{}
	for i := 0; i < 2; i++ {
		view, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *p.Run))
		if err != nil {
			t.Fatal(err)
		}
		task := assignedTask(t, m.f, m.p)
		m.f.claim(t, core.Actor{Principal: m.f.p, SessionID: view.SessionID}, task.Key)
		tasks = append(tasks, task)
	}
	m.report(t, p, "started")
	if _, err := m.f.s.db.Exec(`UPDATE tasks SET data=json_set(data,'$.lease.generation',999) WHERE key=?`, tasks[1].Key); err != nil {
		t.Fatal(err)
	}
	_, err := m.f.s.ManagedReport(testContext, m.a, m.reportInput(p, "completed"))
	requireCode(t, err, "CONFLICT")
	first, err := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: tasks[0].Key})
	if err != nil || first.State != "in_progress" || first.Lease == nil || first.Attempts[0].EndedAt != nil {
		t.Fatal("failed handoff partly released prior task", err)
	}
	run, err := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: p.Run.ID})
	if err != nil || run.Run.State != "running" {
		t.Fatal("invalid lease committed terminal run state", err)
	}
}
