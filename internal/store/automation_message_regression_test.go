package store

import (
	"agent-board/internal/core"
	"strings"
	"testing"
	"time"
)

// Reviewer findings reach only the developer of the reviewed task, labelled as
// gate findings; the next independent reviewer never receives them as orders.
func TestAutomationFindingsReachOnlyDeveloperOfSameTaskLabelledAsFindings(t *testing.T) {
	m := automationSetup(t)
	id := automationToReview(t, m)
	r := m.attempt(t, id)
	target := r.claim.Task.Review.TargetKey
	m.finish(t, r, core.Evidence{VerifiedCommit: commitB, Findings: "Regression in retry loop; add coverage."}, "changes_requested")
	m.exit(t, r, commitB, commitB)
	page := m.tick(t, id)
	if len(page.Workflow.MessageMeta) != 1 || page.Workflow.MessageMeta[0].Source != core.WorkflowMessageReviewerFindings || page.Workflow.MessageMeta[0].TaskKey != target || page.Workflow.MessageMeta[0].DefectKey != r.claim.Task.Key {
		t.Fatal("findings not labelled", page.Workflow.MessageMeta)
	}
	d := m.attempt(t, id)
	prompt := d.poll.Command.Text
	if !strings.Contains(prompt, "Independent reviewer/tester findings for task "+target) || !strings.Contains(prompt, "Regression in retry loop") || strings.Contains(prompt, "Human clarifications") {
		t.Fatal("developer prompt mislabels findings", prompt)
	}
	m.finish(t, d, core.Evidence{Commit: commitA, RunSummary: "fixed"}, "")
	m.exit(t, d, commitA, "")
	m.tick(t, id)
	again := m.attempt(t, id)
	if again.poll.Run.Profile.Role != "reviewer" || strings.Contains(again.poll.Command.Text, "Regression in retry loop") {
		t.Fatal("findings leaked into the next independent review", again.poll.Run.Profile.Role)
	}
}

// Legacy rows without labels: server-authored fix/test-design/answer/replan
// prefixes are classified; anything else keeps the previous human delivery.
func TestAutomationLegacyUnlabelledMessagesKeepHumanSemanticsOnlyForHumanText(t *testing.T) {
	b := &board{Tasks: map[string]*core.Task{"E-9": {Key: "E-9", Role: "tester", Review: &core.Review{TargetKey: "E-2"}}}}
	w := &core.Workflow{Messages: []string{"Keep the API stable.", workflowFixPrefix + "E-9: two tests failed", workflowTestDesignPrefix + "apply patch", "Ответ человека wq_1: yes", workflowFixPrefix + "E-404: unknown"}}
	got := workflowMessageMetas(b, w)
	want := []core.WorkflowMessageMeta{{Source: core.WorkflowMessageHumanClarification}, {Source: core.WorkflowMessageTesterFindings, TaskKey: "E-2", DefectKey: "E-9"}, {Source: core.WorkflowMessageTestDesign, TaskKey: "E-2", DefectKey: "E-9"}, {Source: core.WorkflowMessageHumanAnswer}, {Source: core.WorkflowMessageHumanClarification}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d: %+v want %+v", i, got[i], want[i])
		}
	}
	developer := workflowPromptMessages(b, w, &core.WorkflowStep{Kind: "developer", TaskKey: "E-2"})
	reviewer := workflowPromptMessages(b, w, &core.WorkflowStep{Kind: "reviewer", TaskKey: "E-10"})
	if !strings.Contains(developer, "two tests failed") || strings.Contains(reviewer, "two tests failed") || !strings.Contains(reviewer, "Keep the API stable.") || !strings.Contains(reviewer, "E-404: unknown") {
		t.Fatal("legacy delivery", developer, reviewer)
	}
	// Appending to a legacy row pads labels so indices stay aligned.
	workflowAppendMessage(w, core.WorkflowMessageMeta{Source: core.WorkflowMessageHumanClarification}, "new")
	if len(w.MessageMeta) != len(w.Messages) || workflowMessageMetas(b, w)[1].Source != core.WorkflowMessageTesterFindings {
		t.Fatal("legacy labels misaligned after append", w.MessageMeta)
	}
}

// Production defect A: a workflow queued behind a busy workspace must not bump
// its version on every scheduler scan, or every human command races it.
func TestAutomationQueuedBehindBusyWorkspaceKeepsStableVersion(t *testing.T) {
	m := automationSetup(t)
	first := m.submit(t)
	second := m.submit(t)
	m.tick(t, first.Workflow.ID)
	waiting := m.tick(t, second.Workflow.ID)
	if waiting.Workflow.State != "queued" || waiting.Workflow.ReasonCode != "CONFLICT" || waiting.Workflow.ReasonMessage == "" {
		t.Fatal("second workflow not waiting with a reason", waiting.Workflow.State, waiting.Workflow.ReasonCode)
	}
	for i := 0; i < 5; i++ {
		m.f.clock.advance(2 * time.Second)
		v, err := m.f.s.AutomationTick(m.ctx, core.WorkflowAdvanceInput{WorkflowID: second.Workflow.ID, ExpectedVersion: waiting.Workflow.Version, ControlEpoch: waiting.Workflow.ControlEpoch})
		if err != nil || v.Changed || v.Workflow.Version != waiting.Workflow.Version {
			t.Fatal("queued retry churned the workflow version", err, v.Changed, v.Workflow.Version)
		}
	}
	if _, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: second.Workflow.ID, Version: waiting.Workflow.Version, Kind: "clarify", Text: "Also log retries."}); err != nil {
		t.Fatal("human command with a page from before the scans was rejected", err)
	}
}

// A native failure pauses the workflow with the worker's own redacted error.
func TestAutomationNativeFailureReasonCarriesBoundedRunError(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, "fixture-worker", "failed", "local workspace or provider identity changed; token=abc123")); e != nil {
		t.Fatal(e)
	}
	page := m.tick(t, id)
	if page.Workflow.State != "paused" || page.Workflow.ReasonCode != "native_failed" || !strings.Contains(page.Workflow.ReasonMessage, "provider identity changed") || strings.Contains(page.Workflow.ReasonMessage, "abc123") {
		t.Fatal("failure cause not exposed safely", page.Workflow.ReasonCode, page.Workflow.ReasonMessage)
	}
	list, err := m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{})
	if err != nil || len(list.Workflows) != 1 || list.Workflows[0].ReasonMessage != page.Workflow.ReasonMessage {
		t.Fatal("list DTO lost reason_message", err)
	}
	long := workflowReasonText(strings.Repeat("ж", 600))
	if len(long) > 512 || !strings.HasPrefix(strings.Repeat("ж", 600), long) {
		t.Fatal("reason text not bounded on a rune boundary", len(long))
	}
}
