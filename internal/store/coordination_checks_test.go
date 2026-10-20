package store

import (
	"agent-board/internal/core"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func createCheck(t *testing.T, f *fixture, target core.Task, role string) core.Task {
	t.Helper()
	v, err := f.s.CheckCreate(testContext, f.coord, core.CheckCreateInput{RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version, Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestCheckCreateAtomicReviewQAAndEpicGates(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	target = f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "Product checks"}, "")
	request := core.CheckCreateInput{RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version, Role: "reviewer"}
	review, err := f.s.CheckCreate(testContext, f.coord, request)
	if err != nil || review.Review.TargetGeneration != target.Generation || review.Review.TargetCommit != commitA {
		t.Fatal(err)
	}
	for _, attempt := range []core.CheckCreateInput{{RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version, Role: "tester"}, {RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version - 1, Role: "reviewer"}, {RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version, Role: "reviewer", ProfileID: "bad", ProfileVersion: 1}} {
		var before int
		f.s.db.QueryRow(`SELECT count(*) FROM tasks`).Scan(&before)
		if _, e := f.s.CheckCreate(testContext, f.coord, attempt); e == nil {
			t.Fatal("invalid check create accepted")
		}
		var after int
		f.s.db.QueryRow(`SELECT count(*) FROM tasks`).Scan(&after)
		if after != before {
			t.Fatal("partial check")
		}
	}
	reviewer := f.actor(t, "reviewer")
	f.finish(t, reviewer, f.claim(t, reviewer, review.Key), core.Evidence{VerifiedCommit: commitA, Findings: "Approved"}, "approve")
	qa := createCheck(t, f, target, "tester")
	if len(qa.Links) != 1 || qa.Links[0].From != review.Key || qa.Links[0].Condition != "approved" {
		t.Fatal("QA missing atomic review gate", qa)
	}
	// A newer pending review invalidates the old approved prerequisite immediately.
	newer := createCheck(t, f, target, "reviewer")
	blocked, err := f.s.Coordination(testContext, f.coord, core.CoordinationInput{TaskKey: qa.Key})
	if err != nil || len(blocked.Blockers) == 0 || blocked.Dependencies[0].Satisfied {
		t.Fatal("latest pending bypass", blocked, err)
	}
	tester := f.actor(t, "tester")
	next, err := f.s.Next(testContext, tester, core.NextInput{RequestID: f.id(), TaskKey: qa.Key})
	if err != nil || next.Task != nil {
		t.Fatal("QA claimed stale review", err)
	}
	f.finish(t, reviewer, f.claim(t, reviewer, newer.Key), core.Evidence{VerifiedCommit: commitA, Findings: "Approved new cycle"}, "approve")
	// Replace the previous edge explicitly; old approval history remains.
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: review.Key, To: qa.Key, Type: "blocks", Unlink: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: newer.Key, To: qa.Key, Type: "blocks", Condition: "approved"})
	if err != nil {
		t.Fatal(err)
	}
	f.finish(t, tester, f.claim(t, tester, qa.Key), core.Evidence{VerifiedCommit: commitA, RunSummary: "QA passed"}, "passed")
	epic, err := f.s.Coordination(testContext, f.coord, core.CoordinationInput{TaskKey: f.epic.Key})
	if err != nil || epic.EpicClosable == nil || !*epic.EpicClosable {
		t.Fatal("epic not closable", epic, err)
	}
	if _, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"}); err != nil {
		t.Fatal(err)
	}
	// Exact replay survives a later target metadata change; a new stale request does not.
	title := "Edited after check"
	_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: target.Key}, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.s.CheckCreate(testContext, f.coord, request)
	if err != nil || replay.Key != review.Key {
		t.Fatal("check replay lost", err)
	}
	request.RequestID = f.id()
	_, err = f.s.CheckCreate(testContext, f.coord, request)
	requireCode(t, err, "CONFLICT")
}
func TestCheckCurrentNegativeIsNotStaleAndCreationRaceIsVersionSafe(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	target = f.finish(t, dev, f.claim(t, dev, target.Key), core.Evidence{Commit: commitA, RunSummary: "checks"}, "")
	review := createCheck(t, f, target, "reviewer")
	reviewer := f.actor(t, "reviewer")
	f.finish(t, reviewer, f.claim(t, reviewer, review.Key), core.Evidence{VerifiedCommit: commitA, Findings: "Defect"}, "changes_requested")
	out, err := f.s.Coordination(testContext, f.coord, core.CoordinationInput{TaskKey: target.Key})
	if err != nil || len(out.Checks) != 1 || !out.Checks[0].Current || !out.Checks[0].Latest || out.Checks[0].Successful {
		t.Fatal("negative labelled stale or successful", out, err)
	}
	_, err = f.s.CheckCreate(testContext, f.coord, core.CheckCreateInput{RequestID: f.id(), TargetKey: target.Key, TargetVersion: target.Version, Role: "tester"})
	requireCode(t, err, "RULE_VIOLATION")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	created := make(chan core.Task, 1)
	createID, reopenID := f.id(), f.id()
	wg.Add(2)
	go func() {
		defer wg.Done()
		v, e := f.s.CheckCreate(testContext, f.coord, core.CheckCreateInput{RequestID: createID, TargetKey: target.Key, TargetVersion: target.Version, Role: "reviewer"})
		created <- v
		errs <- e
	}()
	go func() {
		defer wg.Done()
		_, e := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: reopenID, TaskKey: target.Key, Reason: "Fix defects"}, State: "ready", EvidenceRef: review.Key})
		errs <- e
	}()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			requireCode(t, e, "CONFLICT")
		}
	}
	maybe := <-created
	if maybe.Key != "" {
		out, e := f.s.Coordination(testContext, f.coord, core.CoordinationInput{TaskKey: maybe.Key})
		if e != nil || len(out.Blockers) == 0 {
			t.Fatal("race-created review became runnable", out, e)
		}
	}
}
func TestAdministrativeTransitionCannotFallbackFromStaleGeneration(t *testing.T) {
	f := newFixture(t)
	human := f.actor(t, "human")
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	claim := f.claim(t, dev, task.Key)
	if _, err := f.s.Release(testContext, dev, core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "handoff"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.s.Transition(testContext, human, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "stale cancel"}, State: "cancelled"})
	requireCode(t, err, "LEASE_LOST")
	cancelled, err := f.s.Transition(testContext, human, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Reason: "explicit admin cancel"}, State: "cancelled"})
	if err != nil || cancelled.State != "cancelled" {
		t.Fatal(err)
	}
}

func TestCoordinationExplanationBudgetCountsUniqueReasonsAndMarksTruncated(t *testing.T) {
	b := &board{Tasks: map[string]*core.Task{}}
	consumer := &core.Task{Key: "T-1000", Type: "task", Role: "developer", State: "ready"}
	b.Tasks[consumer.Key] = consumer
	for i := 1; i <= 202; i++ {
		key := fmt.Sprintf("T-%d", i)
		b.Tasks[key] = &core.Task{Key: key, Type: "task", State: "ready"}
		for _, kind := range []string{"blocks", "contract_of"} {
			consumer.Links = append(consumer.Links, core.Link{From: key, To: consumer.Key, Type: kind})
		}
	}
	out := core.TaskCoordination{Blockers: []core.CoordinationReason{}}
	explainBlocked(b, consumer, map[string]bool{}, &out.Blockers)
	normalizeCoordination(&out)
	if !out.Truncated || len(out.Blockers) != 200 {
		t.Fatal("budget hid omitted unique blockers", len(out.Blockers), out.Truncated)
	}
}

func TestCoordinationClosureReasonBudgetIsDeterministic(t *testing.T) {
	var first []core.CoordinationReason
	for _, reverse := range []bool{false, true} {
		var items []core.CoordinationReason
		for n := 1; n <= 300; n++ {
			i := n
			if reverse {
				i = 301 - n
			}
			appendCoordReason(&items, reason("unfinished_work", "pending", fmt.Sprintf("T-%d", i)))
		}
		actual, truncated := normalizeReasons(items)
		if !truncated || len(actual) != 200 || actual[0].TaskKey != "T-1" || actual[199].TaskKey != "T-200" {
			t.Fatal("unstable bounded subset", actual)
		}
		if first == nil {
			first = actual
		} else if !reflect.DeepEqual(first, actual) {
			t.Fatal("same reasons depend on traversal order")
		}
	}
}

func TestCoordinationRequiredRolesKeepBothReasonsDeterministically(t *testing.T) {
	var first []core.CoordinationReason
	for _, reverse := range []bool{false, true} {
		roles := []string{"reviewer", "tester"}
		if reverse {
			roles = []string{"tester", "reviewer"}
		}
		var reasons []core.CoordinationReason
		for _, role := range roles {
			appendCoordReason(&reasons, reason("verification_required", "Need "+role, "T-2"))
		}
		actual, _ := normalizeReasons(reasons)
		if len(actual) != 2 {
			t.Fatal("required role disappeared")
		}
		if first == nil {
			first = actual
		} else if !reflect.DeepEqual(first, actual) {
			t.Fatal("required role messages depend on order")
		}
	}
}
