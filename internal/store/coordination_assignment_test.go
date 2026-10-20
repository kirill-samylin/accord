package store

import (
	"agent-board/internal/core"
	"reflect"
	"sync"
	"testing"
	"time"
)

func boundActor(t *testing.T, f *fixture, p core.AgentProfile) core.Actor {
	t.Helper()
	v, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: p.Role, RunID: f.id(), Machine: f.p.Machine, ProfileID: p.ID, ProfileVersion: p.Version})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: f.p, SessionID: v.SessionID}
}
func assignedTask(t *testing.T, f *fixture, p core.AgentProfile) core.Task {
	t.Helper()
	v, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Role: p.Role, Title: "Assigned", Description: "Explicit instructions", ProfileID: p.ID, ProfileVersion: p.Version})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestAssignmentRoutingConcurrentClaimAndLiveLeaseSurvivesProfileChange(t *testing.T) {
	f := newFixture(t)
	p := createProfile(t, f)
	other := createProfile(t, f)
	a, b := boundActor(t, f, p), boundActor(t, f, p)
	unbound := f.actor(t, "developer")
	foreign := boundActor(t, f, other)
	task := assignedTask(t, f, p)
	for _, actor := range []core.Actor{unbound, foreign} {
		out, err := f.s.Next(testContext, actor, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
		if err != nil || out.Task != nil || out.Queue.Ready != 0 || out.Queue.Waiting != 1 {
			t.Fatal("routing bypass", out, err)
		}
	}
	second, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	var wg sync.WaitGroup
	claims := make(chan core.Claim, 2)
	errs := make(chan error, 2)
	for i, actor := range []core.Actor{a, b} {
		wg.Add(1)
		store := f.s
		if i == 1 {
			store = second
		}
		id := f.id()
		go func() {
			defer wg.Done()
			out, e := store.Next(testContext, actor, core.NextInput{RequestID: id, TaskKey: task.Key})
			claims <- out
			errs <- e
		}()
	}
	wg.Wait()
	close(claims)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var won core.Claim
	count := 0
	for claim := range claims {
		if claim.Task != nil {
			won = claim
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate lease", count)
	}
	holder := a
	if won.Lease.SessionID == b.SessionID {
		holder = b
	}
	_, err = f.s.Assign(testContext, f.coord, core.AssignInput{RequestID: f.id(), TaskKey: task.Key, Version: won.Task.Version, Mode: "role_queue", Reason: "steal"})
	requireCode(t, err, "CONFLICT")
	_, err = f.s.ProfileArchive(testContext, f.coord, core.ProfileArchiveInput{RequestID: f.id(), ProfileID: p.ID, Version: p.Version, State: "archived"})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(time.Second)
	if _, err = f.s.Heartbeat(testContext, holder, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: won.Lease.Generation}); err != nil {
		t.Fatal("archive revoked lease", err)
	}
	done := f.finish(t, holder, won, core.Evidence{Commit: commitA, RunSummary: "same bound instructions"}, "")
	if done.Assignment.ProfileVersion != 1 {
		t.Fatal("assignment mutated")
	}
}
func TestAssignmentVersionEditClearReplayAndImmutableAudit(t *testing.T) {
	f := newFixture(t)
	p := createProfile(t, f)
	a := boundActor(t, f, p)
	task := assignedTask(t, f, p)
	changed := settings()
	changed.Instructions = "new instructions"
	newer, err := f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: f.id(), ProfileID: p.ID, Version: p.Version, ProfileSettings: changed})
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.s.Next(testContext, a, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
	if err != nil || out.Task != nil {
		t.Fatal("old snapshot claimed changed profile", err)
	}
	explain, err := f.s.Coordination(testContext, f.coord, core.CoordinationInput{TaskKey: task.Key})
	if err != nil || len(explain.RoutingReasons) != 1 || explain.RoutingReasons[0].Code != "assignment_changed" {
		t.Fatal(explain, err)
	}
	in := core.AssignInput{RequestID: f.id(), TaskKey: task.Key, Version: task.Version, Mode: "profile", ProfileID: newer.ID, ProfileVersion: newer.Version, Reason: "Use updated instructions"}
	updated, err := f.s.Assign(testContext, f.coord, in)
	if err != nil || updated.Version != task.Version+1 {
		t.Fatal(err)
	}
	exact, err := f.s.Assign(testContext, f.coord, in)
	if err != nil || !reflect.DeepEqual(updated, exact) {
		t.Fatal("replay differs", err)
	}
	in.Reason = "changed"
	_, err = f.s.Assign(testContext, f.coord, in)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	_, err = f.s.Assign(testContext, f.coord, core.AssignInput{RequestID: f.id(), TaskKey: task.Key, Version: task.Version, Mode: "role_queue", Reason: "stale"})
	requireCode(t, err, "CONFLICT")
	cleared, err := f.s.Assign(testContext, f.coord, core.AssignInput{RequestID: f.id(), TaskKey: task.Key, Version: updated.Version, Mode: "role_queue", Reason: "Return to role queue"})
	if err != nil || cleared.Assignment != nil || cleared.Generation != task.Generation {
		t.Fatal(err)
	}
	events, err := f.s.Events(testContext, f.coord, core.EventsInput{TaskKey: task.Key, Types: []string{"task_assigned"}})
	if err != nil || len(events.Events) != 2 {
		t.Fatal(err)
	}
	change := events.Events[0].Data.AssignmentChange
	if change == nil || change.Previous.ProfileVersion != 1 || change.Next.ProfileVersion != 2 || change.Reason != "Use updated instructions" {
		t.Fatal("assignment audit lost", events)
	}
	if events.Events[1].Data.AssignmentChange.Next != nil {
		t.Fatal("clear audit fabricated target")
	}
	f.s.Close()
	f.s, err = Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f.claim(t, a, task.Key)
}
func TestAssignmentValidationRejectsPartialPairRolesAndTerminalWorkAtomically(t *testing.T) {
	f := newFixture(t)
	p := createProfile(t, f)
	for _, change := range []func(*core.CreateInput){func(v *core.CreateInput) { v.ProfileVersion = 0 }, func(v *core.CreateInput) { v.ProfileID = "" }, func(v *core.CreateInput) { v.Role = "tester" }, func(v *core.CreateInput) { v.Role = "human" }, func(v *core.CreateInput) { v.Type = "epic" }} {
		in := core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Role: "developer", Title: "Invalid", Description: "Do not create", ProfileID: p.ID, ProfileVersion: 1}
		change(&in)
		var before int
		f.s.db.QueryRow(`SELECT count(*) FROM tasks`).Scan(&before)
		if _, err := f.s.Create(testContext, f.coord, in); err == nil {
			t.Fatal("invalid assignment accepted")
		}
		var after int
		f.s.db.QueryRow(`SELECT count(*) FROM tasks`).Scan(&after)
		if before != after {
			t.Fatal("partial create")
		}
	}
	task := f.task(t, "developer")
	for _, in := range []core.AssignInput{{Mode: "role_queue", ProfileID: p.ID, ProfileVersion: 1}, {Mode: "profile", ProfileID: p.ID}, {Mode: ""}, {Mode: "role_queue", Reason: ""}} {
		in.RequestID = f.id()
		in.TaskKey = task.Key
		in.Version = task.Version
		if in.Mode != "role_queue" || in.ProfileID != "" {
			in.Reason = "invalid"
		}
		if _, err := f.s.Assign(testContext, f.coord, in); err == nil {
			t.Fatal("invalid assign accepted", in)
		}
	}
}
