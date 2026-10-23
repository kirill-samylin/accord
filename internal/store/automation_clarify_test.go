package store

import (
	"agent-board/internal/core"
	"testing"
)

func TestAutomationClarifyAfterDoneRequiresExplicitReplan(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	page := m.page(t, id)
	clarified, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "clarify", Text: "Clarify the acceptance before advancing."})
	if e != nil {
		t.Fatal(e)
	}
	if clarified.Steps[0].DeliveredMessages != 0 {
		t.Fatal("queued clarification shownapplied")
	}
	plan := m.plan(t, id, a)
	m.finish(t, a, core.Evidence{Summary: "oldturncompleted", Artifact: []string{"workflow-plan:" + plan.Plans[0].Hash}}, "")
	m.exit(t, a, commitA, "")
	paused := m.tick(t, id)
	if paused.Workflow.State != "paused" || paused.Workflow.ReasonCode != "clarification_after_completion" || len(paused.Steps) != 1 {
		t.Fatal("clarification silentlyadvanced", paused.Workflow)
	}
	if _, e = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: paused.Workflow.Version, Kind: "resume"}); e == nil {
		t.Fatal("resume bypassedclarification")
	}
	replanned, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: paused.Workflow.Version, Kind: "replan", Text: "Accept prior analysis as history; revise requirement and acceptance."})
	if e != nil || replanned.Workflow.State != "queued" {
		t.Fatal(e)
	}
	next := m.tick(t, id)
	if next.Workflow.State != "running" || len(next.Steps) != 2 || next.Steps[1].Kind != "analyst" {
		t.Fatal("newanalysisnotqueued", next.Workflow)
	}
}
func TestAutomationClarifyUnfinishedUsesExactConversationNextTurnAndObservedDelivery(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	first := m.attempt(t, id)
	page := m.page(t, id)
	if _, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "clarify", Text: "Inspect the retry behavior too."}); e != nil {
		t.Fatal(e)
	}
	m.exit(t, first, commitA, "")
	queued := m.tick(t, id)
	if len(queued.Steps) != 1 || queued.Steps[0].ManagedGeneration != 2 || queued.Steps[0].DeliveredMessages != 0 || queued.Workflow.Counters.Invocations != 2 {
		t.Fatal("continuationnotqueued", queued.Workflow)
	}
	second := m.attempt(t, id)
	if second.poll.Run.ID != first.poll.Run.ID || second.uuid != first.uuid || second.actor.SessionID == first.actor.SessionID {
		t.Fatal("resume lostexactcontext/sessionisolation")
	}
	delivered := m.page(t, id)
	if delivered.Steps[0].DeliveredMessages != 1 || delivered.Workflow.Counters.Started != 2 {
		t.Fatal("observedturnnotmarked")
	}
	plan := m.plan(t, id, second)
	m.finish(t, second, core.Evidence{Summary: "newturn incorporated clarification", Artifact: []string{"workflow-plan:" + plan.Plans[0].Hash}}, "")
	m.exit(t, second, commitA, "")
	next := m.tick(t, id)
	if len(next.Steps) != 2 || next.Steps[1].Kind != "developer" {
		t.Fatal("newturn didnotadvance", next.Workflow)
	}
}
