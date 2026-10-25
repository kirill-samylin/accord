package store

import (
	"agent-board/internal/core"
	"testing"
)

func (m *automationFixture) exitInput(a automationAttempt) core.WorkflowAttestInput {
	binding := a.poll.Run.Automation
	return core.WorkflowAttestInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch}, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, TaskKey: a.claim.Task.Key, TaskGeneration: a.claim.Lease.Generation, NativeSessionID: a.uuid, Kind: "run_exit"}
}
func (m *automationFixture) stopAttempt(t *testing.T, id string, a automationAttempt) automationAttempt {
	t.Helper()
	page := m.page(t, id)
	if _, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"}); e != nil {
		t.Fatal(e)
	}
	poll, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch, WorkerCapabilities: map[string]int{"automation": 1}})
	if e != nil || poll.Command == nil || poll.Command.Kind != "stop" {
		t.Fatal("stop dispatch", e)
	}
	in := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: poll.WorkerEpoch}, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, Phase: "accepted"}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, in); e != nil {
		t.Fatal(e)
	}
	in.RequestID = m.f.id()
	in.Phase = "stopped"
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, in); e != nil {
		t.Fatal(e)
	}
	proof := a
	proof.poll = poll
	return proof
}
func TestAutomationHumanResumePreservesExactConversationRunAndFreshTaskFence(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	first := m.attempt(t, id)
	stop := m.stopAttempt(t, id, first)
	if _, e := m.f.s.WorkflowAttest(m.ctx, m.a, m.exitInput(first)); e == nil {
		t.Fatal("started originalcommand usedasstopproof")
	}
	ack, e := m.f.s.WorkflowAttest(m.ctx, m.a, m.exitInput(stop))
	if e != nil {
		t.Fatal("exact stop terminalproof rejected", e)
	}
	if ack.Attestation.ControlEpoch != first.poll.Run.Automation.ControlEpoch || ack.ControlEpoch <= ack.Attestation.ControlEpoch || ack.Attestation.RunID != stop.poll.Run.ID || ack.Attestation.Generation != stop.poll.Run.Generation {
		t.Fatal("terminal ACK lost immutable original tuple/current metadata distinction", ack)
	}
	page := m.tick(t, id)
	if page.Workflow.State != "paused" {
		t.Fatal(page.Workflow)
	}
	if _, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "resume"}); e != nil {
		t.Fatal(e)
	}
	next := m.tick(t, id)
	if next.Workflow.State != "running" || len(next.Steps) != 1 || next.Steps[0].ManagedRunID != first.poll.Run.ID || next.Steps[0].ManagedGeneration != 2 || next.Steps[0].AttemptNo != 2 || next.Workflow.Counters.Invocations != 1 || next.Steps[0].Continuations != 0 {
		t.Fatal("resume replacedcontext", next.Workflow)
	}
	resumed := m.attempt(t, id)
	if resumed.uuid != first.uuid || resumed.poll.Run.ID != first.poll.Run.ID || resumed.poll.Command.Kind != "resume" || resumed.actor.SessionID == first.actor.SessionID || resumed.claim.Lease.Generation <= first.claim.Lease.Generation {
		t.Fatal("resume lostnativecontext/newfences")
	}
}
func TestAutomationRunExitRequiresAuthoritativeTerminalBeforeCreationAndReplay(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	in := m.exitInput(a)
	if _, e := m.f.s.WorkflowAttest(m.ctx, m.a, in); e == nil {
		t.Fatal("running start receipt acceptedasexit")
	}
	before := m.page(t, id)
	if len(before.Attestations) != 0 {
		t.Fatal("premature proof persisted")
	}
	report := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch}, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, Phase: "completed"}
	if _, e := m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	accepted, e := m.f.s.WorkflowAttest(m.ctx, m.a, in)
	if e != nil || accepted.Attestation.Kind != "run_exit" {
		t.Fatal("terminal proof rejected", e)
	}
	if _, e = m.f.s.WorkflowAttest(m.ctx, m.a, in); e != nil {
		t.Fatal("exact proof retry", e)
	}
	report.RequestID = m.f.id()
	report.Phase = "delivery_unknown"
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	if _, e = m.f.s.WorkflowAttest(m.ctx, m.a, in); e == nil {
		t.Fatal("needsrecovery replay trustedoldpositive")
	}
}
func TestAutomationTerminalGateRejectsAcceptedAndUnknownCommandStates(t *testing.T) {
	for _, spec := range []struct{ state, status, phase, kind string }{{"starting", "accepted", "", "start"}, {"running", "applied", "", "start"}, {"needs_recovery", "delivery_unknown", "completed", "start"}, {"stopped", "applied", "", "start"}} {
		t.Run(spec.state+spec.status, func(t *testing.T) {
			r := &core.ManagedRun{ID: "run", Generation: 1, State: spec.state}
			c := &managedCommandRecord{Command: core.ManagedCommand{RunID: "run", Generation: 1, Status: spec.status, Kind: spec.kind}, TerminalPhase: spec.phase}
			if automationExitReportMatches(r, c) {
				t.Fatal("nonterminal command qualified")
			}
		})
	}
}
