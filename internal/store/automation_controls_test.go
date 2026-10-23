package store

import (
	"agent-board/internal/core"
	"testing"
)

func TestAutomationQuestionHumanAnswerStopAndResume(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	page := m.page(t, id)
	asked, e := m.f.s.WorkflowQuestion(a.ctx, a.actor, core.WorkflowQuestionInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Question: "Which behavior?", Options: []string{"A", "B"}})
	if e != nil {
		t.Fatal(e)
	}
	if asked.Workflow.State != "awaiting_human" || len(asked.Questions) != 1 {
		t.Fatal("question not durable")
	}
	q := asked.Questions[0]
	if _, e = m.f.s.WorkflowAnswer(a.ctx, a.actor, core.WorkflowAnswerInput{RequestID: m.f.id(), WorkflowID: id, Version: asked.Workflow.Version, QuestionID: q.ID, QuestionVersion: q.Version, Answer: "A"}); e == nil {
		t.Fatal("agent answeredhuman")
	}
	stop, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch, WorkerCapabilities: map[string]int{"automation": 1}})
	if e != nil || stop.Command == nil || stop.Command.Kind != "stop" {
		t.Fatal("stop not prioritized", e)
	}
	report := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: stop.WorkerEpoch}, CommandID: stop.Command.ID, DispatchID: stop.Command.DispatchID, RunID: stop.Run.ID, Generation: stop.Run.Generation, Phase: "accepted"}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	report.RequestID = m.f.id()
	report.Phase = "stopped"
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	page = m.page(t, id)
	answered, e := m.f.s.WorkflowAnswer(m.ctx, m.a, core.WorkflowAnswerInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, QuestionID: q.ID, QuestionVersion: q.Version, Answer: "Choose A", Resume: true})
	if e != nil {
		t.Fatal(e)
	}
	if answered.Workflow.State != "queued" || answered.Questions[0].State != "answered" {
		t.Fatal("answer not resumed")
	}
	next := m.tick(t, id)
	if next.Workflow.State != "running" || len(next.Steps) != 1 || next.Steps[0].ManagedRunID != a.poll.Run.ID || next.Steps[0].ManagedGeneration != 2 {
		t.Fatal("native stage didnotcontinue", next.Workflow)
	}
	resumed := m.attempt(t, id)
	if resumed.uuid != a.uuid || resumed.poll.Run.ID != a.poll.Run.ID || resumed.poll.Command.Kind != "resume" || resumed.actor.SessionID == a.actor.SessionID || resumed.claim.Lease.Generation <= a.claim.Lease.Generation {
		t.Fatal("question answer lost exact conversation/fresh fences")
	}
}
func TestAutomationQueuedRequirementsShareWorkspaceSequentiallyAndCancelReleasesDurably(t *testing.T) {
	m := automationSetup(t)
	first := m.submit(t)
	second := m.submit(t)
	one := m.tick(t, first.Workflow.ID)
	two := m.tick(t, second.Workflow.ID)
	if one.Workflow.State != "running" || two.Workflow.State != "queued" || len(two.Steps) != 0 {
		t.Fatal("workspace concurrency", one.Workflow, two.Workflow)
	}
	cancelled, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: one.Workflow.ID, Version: one.Workflow.Version, Kind: "cancel"})
	if e != nil || cancelled.Workflow.State != "cancelled" {
		t.Fatal(e)
	}
	two = m.tick(t, second.Workflow.ID)
	if two.Workflow.State != "running" {
		t.Fatal("reservation survivedprovenundispatchedcancel", two.Workflow)
	}
	for _, task := range one.Steps {
		v, e := m.f.s.Get(m.ctx, m.a, core.GetInput{TaskKey: task.TaskKey})
		if e != nil || v.State != "cancelled" {
			t.Fatal("cancel leftreadytask", e)
		}
	}
}
func TestAutomationPausedGrantRequiresExplicitHumanRefreshAfterPolicyRevision(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	changed, e := m.f.s.AutomationPolicyUpdate(m.ctx, m.a, core.AutomationPolicyUpdateInput{AutomationPolicyCreateInput: core.AutomationPolicyCreateInput{RequestID: m.f.id(), AutomationPolicySettings: m.p.AutomationPolicySettings}, PolicyID: m.p.ID, Version: m.p.Version})
	if e != nil {
		t.Fatal(e)
	}
	page := m.tick(t, w.Workflow.ID)
	if page.Workflow.State != "paused" {
		t.Fatal("policyupdate didnotpause")
	}
	_, e = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, Kind: "resume"})
	if e == nil {
		t.Fatal("implicitregrant")
	}
	approved, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, Kind: "resume", RefreshPolicyID: changed.ID, RefreshPolicyVersion: changed.Version})
	if e != nil || approved.Workflow.Grant.PolicyVersion != changed.Version {
		t.Fatal("explicitrefresh", e)
	}
}

func TestAutomationArchivedSchedulerStopsWithoutDispatch(t *testing.T) {
	m := automationSetup(t)
	queued := m.submit(t)
	running := m.tick(t, queued.Workflow.ID)
	attempt := m.attempt(t, queued.Workflow.ID)
	authority, _ := AutomationAuthorityFromContext(m.ctx)
	authority.Writable = false
	archived := WithAutomationAuthority(m.ctx, authority)
	candidates, e := m.f.s.AutomationCandidates(archived, 32)
	if e != nil || len(candidates) != 1 {
		t.Fatal("archivedcandidates", e)
	}
	current := m.page(t, queued.Workflow.ID)
	result, e := m.f.s.AutomationTick(archived, core.WorkflowAdvanceInput{WorkflowID: current.Workflow.ID, ExpectedVersion: current.Workflow.Version, ControlEpoch: current.Workflow.ControlEpoch})
	if e != nil || result.Workflow.State != "pausing" {
		t.Fatal("archivednotstopping", e, result.Workflow)
	}
	after := m.page(t, queued.Workflow.ID)
	if len(after.Steps) != len(running.Steps) {
		t.Fatal("archiveddispatchednewstep")
	}
	if _, e = m.f.s.Heartbeat(attempt.ctx, attempt.actor, core.HeartbeatInput{RequestID: m.f.id(), TaskKey: attempt.claim.Task.Key, Generation: attempt.claim.Lease.Generation}); e == nil {
		t.Fatal("archivedcapstillactive")
	}
}

func TestAutomationArchivedStopPlaneAcceptsOnlyStopAndTerminalProof(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	m.tick(t, w.Workflow.ID)
	a := m.attempt(t, w.Workflow.ID)
	authority, _ := AutomationAuthorityFromContext(m.ctx)
	authority.Writable = false
	archived := WithAutomationAuthority(m.ctx, authority)
	page := m.page(t, w.Workflow.ID)
	if _, e := m.f.s.AutomationTick(archived, core.WorkflowAdvanceInput{WorkflowID: w.Workflow.ID, ExpectedVersion: page.Workflow.Version, ControlEpoch: page.Workflow.ControlEpoch}); e != nil {
		t.Fatal(e)
	}
	poll, e := m.f.s.ManagedPoll(archived, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch})
	if e != nil || poll.Command == nil || poll.Command.Kind != "stop" {
		t.Fatal("archivedstopplane", e)
	}
	report := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: poll.WorkerEpoch}, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, Phase: "accepted"}
	if _, e = m.f.s.ManagedReport(archived, m.a, report); e != nil {
		t.Fatal("archivedstopack", e)
	}
	report.RequestID = m.f.id()
	report.Phase = "stopped"
	if _, e = m.f.s.ManagedReport(archived, m.a, report); e != nil {
		t.Fatal("archivedstopproof", e)
	}
}
