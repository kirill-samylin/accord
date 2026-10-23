package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestAutomationPausedCheckMayReportOwnedExitButCannotLaunchAnother(t *testing.T) {
	m := automationSetup(t)
	id := automationToReview(t, m)
	r := m.attempt(t, id)
	m.finish(t, r, core.Evidence{VerifiedCommit: commitB, Findings: "source checked"}, "approve")
	m.exit(t, r, commitB, commitB)
	m.tick(t, id)
	a := m.attempt(t, id)
	m.attest(t, a, "check_reserved", "owned", core.WorkflowObservations{})
	m.attest(t, a, "check_started", "owned", core.WorkflowObservations{})
	page := m.page(t, id)
	if _, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"}); e != nil {
		t.Fatal(e)
	}
	authority, _ := AutomationAuthorityFromContext(m.ctx)
	authority.Writable = false
	m.ctx = WithAutomationAuthority(m.ctx, authority)
	code := 130
	finished := m.attest(t, a, "check_finished", "owned", core.WorkflowObservations{CheckExitCode: &code})
	if finished.Attestation.Status != "completed" {
		t.Fatal("ownedexitnotpersisted")
	}
	binding := a.poll.Run.Automation
	_, e := m.f.s.WorkflowAttest(m.ctx, m.a, core.WorkflowAttestInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch}, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, TaskKey: a.claim.Task.Key, TaskGeneration: a.claim.Lease.Generation, NativeSessionID: a.uuid, Kind: "check_reserved", CheckRequestID: "new", PresetID: m.check.PresetID, PresetVersion: m.check.LocalVersion, PresetDigest: m.check.LocalDigest})
	if e == nil {
		t.Fatal("pausednewchecklaunched")
	}
}
func TestAutomationCapabilityCannotReclaimExpiredTaskLease(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	if a.claim.Lease.Generation != a.poll.Run.Automation.TaskGeneration {
		t.Fatal("startup expectedgenerationincorrect")
	}
	for i := 0; i < 3; i++ {
		m.f.clock.advance(20 * time.Second)
		if _, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch, WorkerCapabilities: map[string]int{"automation": 1}}); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := m.f.s.Next(a.ctx, a.actor, core.NextInput{RequestID: m.f.id(), TaskKey: a.claim.Task.Key}); e == nil {
		t.Fatal("sameattemptstolenewleasegeneration")
	}
}
