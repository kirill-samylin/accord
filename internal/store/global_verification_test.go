package store

import (
	"agent-board/internal/core"
	"testing"
)

func TestCrossEpicVerificationObligationsRequireCurrentPositive(t *testing.T) {
	for _, role := range []string{"reviewer", "tester"} {
		for _, late := range []bool{false, true} {
			name := role + "/before_close"
			if late {
				name = role + "/after_close"
			}
			t.Run(name, func(t *testing.T) {
				f := newFixture(t)
				dev := f.actor(t, "developer")
				target := f.task(t, "developer")
				f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
				targetEpic := f.epic.Key
				if late {
					f.closeEpic(t, targetEpic)
				}
				f.epic = f.newEpic(t)
				check := f.verification(t, role, target.Key, commitA)
				if f.get(t, targetEpic).State != "open" {
					t.Fatal("late cross-epic verification did not reopen target epic")
				}
				assertCannotClose := func() {
					t.Helper()
					_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: targetEpic}, State: "done"})
					requireCode(t, err, "RULE_VIOLATION")
				}
				assertCannotClose()
				verifier := f.actor(t, role)
				negative, positive := "changes_requested", "approve"
				if role == "tester" {
					negative, positive = "failed", "passed"
				}
				evidence := core.Evidence{VerifiedCommit: commitA, Findings: "checked", RunSummary: "tested"}
				f.finish(t, verifier, f.claim(t, verifier, check.Key), evidence, negative)
				assertCannotClose()
				replacement := f.verification(t, role, target.Key, commitA)
				f.finish(t, verifier, f.claim(t, verifier, replacement.Key), evidence, positive)
				f.closeEpic(t, targetEpic)
				f.closeEpic(t, f.epic.Key)
				// A same-SHA new developer generation invalidates remote approvals.
				_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: target.Key, Reason: "defect"}, State: "ready", EvidenceRef: check.Key})
				if err != nil {
					t.Fatal(err)
				}
				if f.get(t, f.epic.Key).State != "open" {
					t.Fatal("stale remote verifier left its epic done")
				}
				f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "retested"}, "")
				assertCannotClose()
				fresh := f.verification(t, role, target.Key, commitA)
				f.finish(t, verifier, f.claim(t, verifier, fresh.Key), evidence, positive)
				f.closeEpic(t, targetEpic)
				f.closeEpic(t, f.epic.Key)
			})
		}
	}
}

func TestLateCrossEpicCheckReopensEvenWithExistingCurrentPositive(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
	targetEpic := f.epic.Key
	f.epic = f.newEpic(t)
	// Force creation-order keys across the lexical T-9/T-10 boundary.
	for range 5 {
		f.newEpic(t)
	}
	check := f.verification(t, "reviewer", target.Key, commitA)
	if check.Key != "T-9" {
		t.Fatal("regression setup must put older approval at T-9")
	}
	reviewer := f.actor(t, "reviewer")
	f.finish(t, reviewer, f.claim(t, reviewer, check.Key), core.Evidence{VerifiedCommit: commitA, Findings: "checked"}, "approve")
	f.closeEpic(t, targetEpic)
	f.closeEpic(t, f.epic.Key)
	pending := f.verification(t, "reviewer", target.Key, commitA)
	if pending.Key != "T-10" {
		t.Fatal("regression setup must put new assignment at T-10")
	}
	if f.get(t, targetEpic).State != "open" {
		t.Fatal("new verification must explicitly reopen the target epic")
	}
	assertCannotClose := func() {
		t.Helper()
		_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: targetEpic}, State: "done"})
		requireCode(t, err, "RULE_VIOLATION")
	}
	assertCannotClose()
	f.finish(t, reviewer, f.claim(t, reviewer, pending.Key), core.Evidence{VerifiedCommit: commitA, Findings: "new defect"}, "changes_requested")
	assertCannotClose()
	replacement := f.verification(t, "reviewer", target.Key, commitA)
	f.finish(t, reviewer, f.claim(t, reviewer, replacement.Key), core.Evidence{VerifiedCommit: commitA, Findings: "fixed"}, "approve")
	f.closeEpic(t, targetEpic)
	f.closeEpic(t, f.epic.Key)
}

func TestSupersededVerificationRequiresExplicitDependencyReplacement(t *testing.T) {
	for _, role := range []string{"reviewer", "tester"} {
		t.Run(role, func(t *testing.T) {
			f := newFixture(t)
			dev := f.actor(t, "developer")
			target := f.task(t, "developer")
			f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
			f.epic = f.newEpic(t)
			verificationEpic := f.epic
			old := f.verification(t, role, target.Key, commitA)
			verifier := f.actor(t, role)
			positive, negative := "approve", "changes_requested"
			if role == "tester" {
				positive, negative = "passed", "failed"
			}
			evidence := core.Evidence{VerifiedCommit: commitA, Findings: "checked", RunSummary: "tested"}
			f.finish(t, verifier, f.claim(t, verifier, old.Key), evidence, positive)
			f.epic = f.newEpic(t)
			consumer := f.task(t, "human")
			f.link(t, old.Key, consumer.Key)
			human := f.actor(t, "human")
			claim := f.claim(t, human, consumer.Key)
			_, err := f.s.Release(testContext, human, core.LeaseInput{RequestID: f.id(), TaskKey: consumer.Key, Generation: claim.Lease.Generation, Reason: "defer"})
			if err != nil {
				t.Fatal(err)
			}
			f.epic = verificationEpic
			pending := f.verification(t, role, target.Key, commitA)
			assertBlocked := func() {
				t.Helper()
				v, err := f.s.Next(testContext, human, core.NextInput{RequestID: f.id(), TaskKey: consumer.Key})
				if err != nil || v.Task != nil || !f.get(t, consumer.Key).Blocked {
					t.Fatal("superseded approval incorrectly opened downstream gate")
				}
			}
			assertBlocked()
			f.finish(t, verifier, f.claim(t, verifier, pending.Key), evidence, negative)
			assertBlocked()
			fresh := f.verification(t, role, target.Key, commitA)
			f.finish(t, verifier, f.claim(t, verifier, fresh.Key), evidence, positive)
			assertBlocked() // A link names its exact source, never an implicit replacement.
			_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: old.Key, To: consumer.Key, Type: "blocks", Unlink: true})
			if err != nil {
				t.Fatal(err)
			}
			f.link(t, fresh.Key, consumer.Key)
			f.claim(t, human, consumer.Key)
			f.closeEpic(t, verificationEpic.Key)
		})
	}
}
