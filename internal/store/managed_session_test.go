package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func managedDescribeInput(m *managedFixture, run core.ManagedRun) core.OpenSession {
	return core.OpenSession{RequestID: m.f.id(), Role: run.Profile.Role, RunID: run.ID, Machine: m.f.p.Machine, ProfileID: run.Profile.ID, ProfileVersion: run.Profile.Version, ManagedRunID: run.ID, ManagedGeneration: run.Generation}
}
func TestManagedDescribeContinuesImmutableSnapshotAfterEditArchiveWithoutWeakeningTaskGates(t *testing.T) {
	m := managedSetup(t)
	task := assignedTask(t, m.f, m.p)
	m.start(t)
	p := m.poll(t)
	first := managedDescribeInput(m, *p.Run)
	_, err := m.f.s.Describe(testContext, m.f.p, first)
	requireCode(t, err, "CONFLICT")
	m.report(t, p, "accepted")
	opened, err := m.f.s.Describe(testContext, m.f.p, first)
	if err != nil || opened.ManagedRunID != p.Run.ID || opened.Profile.Instructions != m.p.Instructions {
		t.Fatal("managed snapshot binding", err)
	}
	m.report(t, p, "started")
	done := m.report(t, p, "completed")
	settings := m.p.ProfileSettings
	settings.Instructions = "A new revision"
	updated, err := m.f.s.ProfileUpdate(testContext, m.f.coord, core.ProfileUpdateInput{RequestID: m.f.id(), ProfileID: m.p.ID, Version: 1, ProfileSettings: settings})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.f.s.ProfileArchive(testContext, m.f.coord, core.ProfileArchiveInput{RequestID: m.f.id(), ProfileID: m.p.ID, Version: updated.Version, State: "archived"}); err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: done.Run.ID, Version: done.Run.Version, Kind: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	next := m.poll(t)
	m.report(t, next, "accepted")
	in := managedDescribeInput(m, *next.Run)
	resumed, err := m.f.s.Describe(testContext, m.f.p, in)
	if err != nil || resumed.SessionID == opened.SessionID || resumed.ManagedGeneration != 2 || resumed.Profile.Version != 1 || resumed.Profile.Instructions != m.p.Instructions {
		t.Fatal("resume lost immutable snapshot/new session", err)
	}
	ordinary := in
	ordinary.RequestID = m.f.id()
	ordinary.ManagedRunID = ""
	ordinary.ManagedGeneration = 0
	_, err = m.f.s.Describe(testContext, m.f.p, ordinary)
	requireCode(t, err, "CONFLICT")
	_, err = m.f.s.Describe(testContext, m.f.p, first)
	requireCode(t, err, "CONFLICT")
	claim, err := m.f.s.Next(testContext, core.Actor{Principal: m.f.p, SessionID: resumed.SessionID}, core.NextInput{RequestID: m.f.id(), TaskKey: task.Key})
	if err != nil || claim.Task != nil {
		t.Fatal("managed binding bypassed current task assignment gate", err)
	}
}
func TestManagedDescribeRejectsWrongOwnerRoleSnapshotGenerationAndExpiredWorkerLease(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	in := managedDescribeInput(m, *p.Run)
	for _, change := range []func(*core.OpenSession){func(v *core.OpenSession) { v.ManagedGeneration++ }, func(v *core.OpenSession) { v.ManagedGeneration = 0 }, func(v *core.OpenSession) { v.ProfileVersion++ }, func(v *core.OpenSession) { v.ProfileID = "foreign" }, func(v *core.OpenSession) { v.Role = "reviewer" }, func(v *core.OpenSession) { v.RunID = "foreign" }} {
		bad := in
		bad.RequestID = m.f.id()
		change(&bad)
		if _, err := m.f.s.Describe(testContext, m.f.p, bad); err == nil {
			t.Fatal("invalid managed profile binding accepted")
		}
	}
	other, err := m.f.s.CreateToken(testContext, core.TokenInput{Owner: "other", Machine: m.f.p.Machine, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	actor, err := m.f.s.Authenticate(testContext, other.Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.Describe(testContext, actor, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	_, err = m.f.s.Describe(WithManagedEpoch(testContext, 1), m.f.p, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	m.f.clock.advance(31 * time.Second)
	_, err = m.f.s.Describe(testContext, m.f.p, in)
	requireCode(t, err, "CONFLICT")
}
