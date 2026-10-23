package store

import (
	"agent-board/internal/core"
	"strings"
	"testing"
)

func TestAutomationCompleteNativeTaskPipelineRequiresActualCheck(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	w = m.tick(t, id)
	if w.Workflow.State != "running" || len(w.Steps) != 1 {
		t.Fatalf("analysis not queued %+v", w.Workflow)
	}
	analyst := m.attempt(t, id)
	plan := m.plan(t, id, analyst)
	m.finish(t, analyst, core.Evidence{Summary: "Examined repository", Artifact: []string{"workflow-plan:" + plan.Plans[0].Hash}}, "")
	m.exit(t, analyst, commitA, "")
	w = m.tick(t, id)
	if len(w.Steps) != 2 || w.Steps[1].Kind != "developer" {
		t.Fatalf("developer not queued %+v", w.Workflow)
	}
	developer := m.attempt(t, id)
	m.finish(t, developer, core.Evidence{Commit: commitB, RunSummary: "Actual source + unit tests committed"}, "")
	m.exit(t, developer, commitB, "")
	w = m.tick(t, id)
	if len(w.Steps) != 3 || w.Steps[2].Kind != "reviewer" {
		t.Fatal("review missing", w.Workflow)
	}
	reviewer := m.attempt(t, id)
	m.finish(t, reviewer, core.Evidence{VerifiedCommit: commitB, Findings: "Reviewed exact source and tests independently"}, "approve")
	m.exit(t, reviewer, commitB, commitB)
	w = m.tick(t, id)
	if len(w.Steps) != 4 || w.Steps[3].Kind != "tester" {
		t.Fatal("QA missing", w.Workflow)
	}
	tester := m.attempt(t, id)
	m.attest(t, tester, "check_reserved", "check-one", core.WorkflowObservations{})
	m.attest(t, tester, "check_started", "check-one", core.WorkflowObservations{})
	clean := true
	exit := 0
	m.attest(t, tester, "check_finished", "check-one", core.WorkflowObservations{ActualHead: commitB, TargetSHA: commitB, SourceClean: &clean, ManifestSHA256: strings.Repeat("b", 64), CheckExitCode: &exit, ChecksPassed: 7, ReportKind: "junit", ReportSHA256: strings.Repeat("c", 64)})
	m.finish(t, tester, core.Evidence{VerifiedCommit: commitB, RunSummary: "7 tests actually passed"}, "passed")
	m.exit(t, tester, commitB, commitB)
	w = m.tick(t, id)
	if w.Workflow.State != "completed" || w.Workflow.ReservedRuns != 0 || w.Workflow.Counters.Invocations != 4 {
		t.Fatalf("workflow not complete %+v", w.Workflow)
	}
	// Scope uses separate native conversations and separate board sessions.
	seen := map[string]bool{}
	for _, step := range w.Steps {
		if seen[step.NativeSessionID] || step.NativeSessionID == "" {
			t.Fatal("native conversation reused")
		}
		seen[step.NativeSessionID] = true
	}
}
func TestAutomationCLIExitAndAgentOutcomeCannotQualifyQA(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	p := m.plan(t, id, a)
	m.finish(t, a, core.Evidence{Summary: "plan", Artifact: []string{"workflow-plan:" + p.Plans[0].Hash}}, "")
	m.exit(t, a, commitA, "")
	m.tick(t, id)
	d := m.attempt(t, id)
	m.finish(t, d, core.Evidence{Commit: commitB, RunSummary: "commit"}, "")
	m.exit(t, d, commitB, "")
	m.tick(t, id)
	r := m.attempt(t, id)
	m.finish(t, r, core.Evidence{VerifiedCommit: commitB, Findings: "source approve"}, "approve")
	m.exit(t, r, commitB, commitB)
	m.tick(t, id)
	q := m.attempt(t, id)
	m.finish(t, q, core.Evidence{VerifiedCommit: commitB, RunSummary: "claimed pass without check"}, "passed")
	m.exit(t, q, commitB, commitB)
	w = m.tick(t, id)
	if w.Workflow.State != "blocked" || w.Workflow.ReasonCode != "test_evidence_missing" && w.Workflow.ReasonCode != "TEST_EVIDENCE_MISSING" {
		t.Fatalf("false QA accepted %+v", w.Workflow)
	}
}
