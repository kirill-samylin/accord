package store

import (
	"agent-board/internal/core"
	"testing"
)

func (f *fixture) verification(t *testing.T, role, target, commit string) core.Task {
	t.Helper()
	v, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: role, Description: "Verify", Role: role, TargetKey: target, TargetCommit: commit})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestFailedReviewAndTestDoNotOpenGate(t *testing.T) {
	for _, scenario := range []struct{ role, outcome string }{{"reviewer", "changes_requested"}, {"reviewer", "blocked"}, {"tester", "failed"}, {"tester", "not_run"}, {"tester", "not_applicable"}} {
		t.Run(scenario.role+scenario.outcome, func(t *testing.T) {
			f := newFixture(t)
			a := f.actor(t, "developer")
			target := f.task(t, "developer")
			f.finish(t, a, f.claim(t, a, target.Key), core.Evidence{Commit: commitA, RunSummary: "tests passed"}, "")
			check := f.verification(t, scenario.role, target.Key, commitA)
			verifier := f.actor(t, scenario.role)
			e := core.Evidence{VerifiedCommit: commitA, Findings: "Defect", RunSummary: "checked"}
			f.finish(t, verifier, f.claim(t, verifier, check.Key), e, scenario.outcome)
			consumer := f.task(t, "analyst")
			_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: check.Key, To: consumer.Key, Type: "blocks", Condition: "completed"})
			if err != nil {
				t.Fatal(err)
			}
			claim, err := f.s.Next(testContext, f.actor(t, "analyst"), core.NextInput{RequestID: f.id(), TaskKey: consumer.Key})
			if err != nil {
				t.Fatal(err)
			}
			if claim.Task != nil {
				t.Fatal("negative verification opened gate")
			}
			_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"})
			requireCode(t, err, "RULE_VIOLATION")
		})
	}
}
func TestApprovalBecomesStaleAfterReopenSameCommit(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tests"}, "")
	check := f.verification(t, "reviewer", target.Key, commitA)
	reviewer := f.actor(t, "reviewer")
	f.finish(t, reviewer, f.claim(t, reviewer, check.Key), core.Evidence{VerifiedCommit: commitA, Findings: "No findings"}, "approve")
	consumer := f.task(t, "human")
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: check.Key, To: consumer.Key, Type: "blocks"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Transition(testContext, reviewer, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: target.Key, Reason: "Defect"}, State: "ready", EvidenceRef: check.Key})
	if err != nil {
		t.Fatal(err)
	}
	f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "Retested"}, "")
	get, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: consumer.Key})
	if err != nil {
		t.Fatal(err)
	}
	if !get.Blocked {
		t.Fatal("old approve regained validity for same SHA")
	}
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: check.Key, To: consumer.Key, Type: "blocks", Unlink: true})
	if err != nil {
		t.Fatal(err)
	}
	human := f.actor(t, "human")
	f.finish(t, human, f.claim(t, human, consumer.Key), core.Evidence{Answer: "Resolved"}, "")
	_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"})
	requireCode(t, err, "RULE_VIOLATION")
	fresh := f.verification(t, "reviewer", target.Key, commitA)
	f.finish(t, reviewer, f.claim(t, reviewer, fresh.Key), core.Evidence{VerifiedCommit: commitA, Findings: "Fixed"}, "approve")
	epic, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"})
	if err != nil {
		t.Fatal(err)
	}
	if epic.State != "done" {
		t.Fatal("epic did not close")
	}
}
func TestVerifierCannotUseWrongCommitAndProvenanceIsChecked(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	done := f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tests"}, "")
	check := f.verification(t, "reviewer", target.Key, commitA)
	reviewer := f.actor(t, "reviewer")
	claim := f.claim(t, reviewer, check.Key)
	_, err := f.s.Transition(testContext, reviewer, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: check.Key, Generation: claim.Lease.Generation}, State: "done", Outcome: "approve", Evidence: &core.Evidence{VerifiedCommit: commitB, Findings: "No findings"}})
	requireCode(t, err, "RULE_VIOLATION")
	// Exercise the provenance invariant independently of the fixed-role session API.
	b := &board{Tasks: map[string]*core.Task{done.Key: &done}}
	done.Attempts = append(done.Attempts, core.Attempt{SessionID: reviewer.SessionID, Commit: commitA})
	err = completion(b, claim.Task, &core.Session{SessionView: core.SessionView{SessionID: reviewer.SessionID}}, core.TransitionInput{Outcome: "approve", Evidence: &core.Evidence{VerifiedCommit: commitA, Findings: "No findings"}})
	requireCode(t, err, "RULE_VIOLATION")
	f.finish(t, reviewer, claim, core.Evidence{VerifiedCommit: commitA, Findings: "Independent"}, "approve")
}
func TestReviewCreatedBeforeTargetClaimBindsOnFirstCheckClaim(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	check := f.verification(t, "reviewer", target.Key, commitA)
	reviewer := f.actor(t, "reviewer")
	empty, err := f.s.Next(testContext, reviewer, core.NextInput{RequestID: f.id(), TaskKey: check.Key})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Task != nil {
		t.Fatal("incomplete target available for review")
	}
	f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tests"}, "")
	claim := f.claim(t, reviewer, check.Key)
	if claim.Task.Review.TargetGeneration != 1 {
		t.Fatal("review did not bind target generation")
	}
	f.finish(t, reviewer, claim, core.Evidence{VerifiedCommit: commitA, Findings: "Independent"}, "approve")
}
