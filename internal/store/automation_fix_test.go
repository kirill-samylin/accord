package store

import (
	"agent-board/internal/core"
	"testing"
)

func automationToReview(t *testing.T, m *automationFixture) string {
	t.Helper()
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	p := m.plan(t, id, a)
	m.finish(t, a, core.Evidence{Summary: "plan", Artifact: []string{"workflow-plan:" + p.Plans[0].Hash}}, "")
	m.exit(t, a, commitA, "")
	m.tick(t, id)
	d := m.attempt(t, id)
	m.finish(t, d, core.Evidence{Commit: commitB, RunSummary: "implementation"}, "")
	m.exit(t, d, commitB, "")
	m.tick(t, id)
	return id
}
func TestAutomationNegativeReviewReopensSameDeveloperWithNewGeneration(t *testing.T) {
	m := automationSetup(t)
	id := automationToReview(t, m)
	r := m.attempt(t, id)
	target := r.claim.Task.Review.TargetKey
	oldGeneration := r.claim.Task.Review.TargetGeneration
	m.finish(t, r, core.Evidence{VerifiedCommit: commitB, Findings: "Regression in implementation; add meaningful coverage."}, "changes_requested")
	m.exit(t, r, commitB, commitB)
	page := m.tick(t, id)
	if page.Workflow.State != "running" || page.Workflow.Counters.FixTotal != 1 || len(page.Steps) != 4 || page.Steps[3].Kind != "developer" || page.Steps[3].TaskKey != target {
		t.Fatalf("fix not enqueued %+v", page.Workflow)
	}
	d := m.attempt(t, id)
	if d.claim.Lease.Generation <= oldGeneration {
		t.Fatal("fix didnotacquirefreshgeneration")
	}
	if _, e := m.f.s.Transition(r.ctx, r.actor, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: m.f.id(), TaskKey: target, Generation: oldGeneration}, State: "done", Evidence: &core.Evidence{Commit: commitA, RunSummary: "forged"}}); e == nil {
		t.Fatal("oldverifier completeddeveloper")
	}
}
func TestAutomationWorkerSHAProofCannotBeAgentClaimOrDirtyCheckout(t *testing.T) {
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
	m.finish(t, d, core.Evidence{Commit: commitB, RunSummary: "claimednewSHA"}, "")
	m.exit(t, d, commitA, "")
	page := m.tick(t, id)
	if page.Workflow.State != "blocked" || len(page.Steps) != 2 {
		t.Fatal("mismatchedSHA advanced")
	}
}
