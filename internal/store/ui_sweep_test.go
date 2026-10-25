package store

import (
	"testing"
	"time"

	"agent-board/internal/core"
)

// Regressions from the logged-in UI sweep of the Accord stand (26.10.2026).

func TestExplicitPriorityZeroIsTopAndOmittedDefaultsToTwo(t *testing.T) {
	f := newFixture(t)
	zero, hundred := 0, 100
	create := func(priority *int) (core.Task, error) {
		return f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "p", Description: "d", Role: "developer", Priority: priority})
	}
	p0, err := create(&zero)
	if err != nil || p0.Priority != 0 {
		t.Fatalf("explicit P0: %+v %v", p0.Priority, err)
	}
	omitted, err := create(nil)
	if err != nil || omitted.Priority != 2 {
		t.Fatalf("omitted priority: %+v %v", omitted.Priority, err)
	}
	if v, err := create(&hundred); err != nil || v.Priority != 100 {
		t.Fatalf("upper bound: %v", err)
	}
	for _, bad := range []int{-1, 101} {
		requireCode(t, func() error { _, e := create(&bad); return e }(), "VALIDATION")
	}
	// The queue serves P0 before the default priority, and a check inherits it.
	dev := f.actor(t, "developer")
	claim, err := f.s.Next(testContext, dev, core.NextInput{RequestID: f.id()})
	if err != nil || claim.Task == nil || claim.Task.Key != p0.Key {
		t.Fatalf("P0 not first: %+v %v", claim.Task, err)
	}
	done := f.finish(t, dev, claim, core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
	check, err := f.s.CheckCreate(testContext, f.coord, core.CheckCreateInput{RequestID: f.id(), TargetKey: done.Key, TargetVersion: done.Version, Role: "reviewer"})
	if err != nil || check.Priority != 0 {
		t.Fatalf("check priority: %d %v", check.Priority, err)
	}
}

func TestOverviewAvailableExcludesBlockedLikeTeamUnclaimed(t *testing.T) {
	f := newFixture(t)
	blocker := f.task(t, "analyst")
	waiting := f.task(t, "developer")
	f.task(t, "architect")
	if _, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: blocker.Key, To: waiting.Key, Type: "blocks"}); err != nil {
		t.Fatal(err)
	}
	overview, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil {
		t.Fatal(err)
	}
	team, err := f.s.ControlTeam(testContext, f.coord)
	if err != nil {
		t.Fatal(err)
	}
	// Three ready tasks by state, one of them blocked by an unfinished dependency.
	if overview.TasksReady != 3 || overview.TasksBlocked != 1 || overview.TasksAvailable != 2 || team.UnclaimedTotal != overview.TasksAvailable {
		t.Fatalf("ready %d blocked %d available %d unclaimed %d", overview.TasksReady, overview.TasksBlocked, overview.TasksAvailable, team.UnclaimedTotal)
	}
}

func TestTeamDoneRecentCreditsCompletingPerson(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	task := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, task.Key), core.Evidence{Commit: commitA, RunSummary: "tested"}, "")
	human := f.actor(t, "human")
	question := f.task(t, "human")
	f.finish(t, human, f.claim(t, human, question.Key), core.Evidence{Answer: "Да"}, "")
	done := func() int {
		team, err := f.s.ControlTeam(testContext, f.coord)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range team.Members {
			if m.Owner == "alice" {
				return m.DoneRecent
			}
		}
		return -1
	}
	if got := done(); got != 2 {
		t.Fatalf("done_recent = %d, want 2", got)
	}
	// Completions older than the window are not "recent".
	err := f.s.adminTransaction(testContext, func(b *board, now time.Time) error {
		for _, key := range []string{task.Key, question.Key} {
			for i := range b.Tasks[key].Attempts {
				old := now.Add(-teamWindow - time.Second)
				b.Tasks[key].Attempts[i].EndedAt = &old
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := done(); got > 0 {
		t.Fatalf("outside the window: %d", got)
	}
}

func TestDraftResumeExplainsHowToStart(t *testing.T) {
	m := automationSetup(t)
	draft, err := m.f.s.WorkflowSubmit(m.ctx, m.a, core.WorkflowSubmitInput{RequestID: m.f.id(), Text: "draft"})
	if err != nil || draft.Workflow.Grant != nil {
		t.Fatalf("draft: %v", err)
	}
	if page := m.page(t, draft.Workflow.ID); page.CanResume || !page.CanControl {
		t.Fatalf("draft can_resume=%v can_control=%v", page.CanResume, page.CanControl)
	}
	for _, kind := range []string{"resume", "replan"} {
		_, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: draft.Workflow.ID, Version: draft.Workflow.Version, Kind: kind})
		requireCode(t, err, "VALIDATION")
		if err.(*core.Error).Message != workflowDraftMessage {
			t.Fatalf("%s message: %v", kind, err)
		}
		err = m.f.s.AutomationPermission(m.ctx, m.a, core.AutomationPermissionInput{WorkflowID: draft.Workflow.ID, Kind: "command", CommandKind: kind})
		requireCode(t, err, "VALIDATION")
	}
	// Starting with an explicit team policy is unchanged.
	started, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: draft.Workflow.ID, Version: draft.Workflow.Version, Kind: "resume", RefreshPolicyID: m.p.ID, RefreshPolicyVersion: m.p.Version})
	if err != nil || started.Workflow.Grant == nil || started.Workflow.State != "queued" {
		t.Fatalf("start with policy: %v", err)
	}
}

func TestProfileTextsRejectControlCharacters(t *testing.T) {
	v := settings()
	v.Description, v.Instructions = "line\nnext\ttab\r\nend", "ok"
	if err := validateProfile(v); err != nil {
		t.Fatalf("multi-line text rejected: %v", err)
	}
	for _, bad := range []string{"bell\x07", "esc\x1b[2J", "lone\rcr", "bad\xff"} {
		v.Description = bad
		requireCode(t, validateProfile(v), "VALIDATION")
		v.Description, v.Instructions = "", bad
		requireCode(t, validateProfile(v), "VALIDATION")
		v.Instructions = ""
	}
}

func TestBrowserActivityNamesSessionOwner(t *testing.T) {
	f := newFixture(t)
	// A session-open event carries no metric owner of its own.
	page, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{Type: "session_opened", SessionID: f.coord.SessionID, Limit: 1})
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("events: %+v %v", page.Events, err)
	}
	if e := page.Events[0]; e.Actor != "alice" {
		t.Fatalf("activity event has no person: %+v", e)
	}
}
