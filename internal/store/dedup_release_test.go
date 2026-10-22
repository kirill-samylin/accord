package store

import (
	"agent-board/internal/core"
	"fmt"
	"testing"
	"time"
)

func similarEpics(t *testing.T, f *fixture) (core.Task, core.Task) {
	t.Helper()
	create := func(title string) core.Task {
		v, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "epic", Title: title})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	return create("Экспорт JSONL"), create("Экспорт задач JSONL")
}

func TestClaimWarningsSelectLiveWorkBeforeLimitAndRespectOff(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	b := &board{Config: core.Config{DuplicateMode: core.DuplicateWarn}, Tasks: map[string]*core.Task{}}
	for n := 1; n <= 13; n++ {
		key := fmt.Sprintf("T-%d", n)
		task := &core.Task{Key: key, Type: "task", Role: "developer", State: "ready", Title: "Экспорт JSONL", Areas: []string{"ui"}}
		if n > 5 {
			task.State = "in_progress"
			task.Lease = &core.Lease{SessionID: key, ExpiresAt: now.Add(time.Minute)}
		}
		b.Tasks[key] = task
	}
	// An expired lease must not consume a slot either.
	b.Tasks["T-6"].Lease.ExpiresAt = now.Add(-time.Minute)
	current := &core.Task{Key: "T-99", Type: "task", Role: "developer", Title: "Экспорт JSONL", Areas: []string{"ui"}}
	warnings := claimWarnings(b, current, now)
	if len(warnings) != 2 || warnings[0].Rule != "duplicate_check" || len(warnings[0].Similar) != 5 || warnings[0].Similar[0].Key != "T-7" {
		t.Fatal("live holder hidden behind ready work", warnings)
	}
	for _, warning := range warnings {
		if len(warning.Similar) > 5 {
			t.Fatal("unbounded claim warning", warning)
		}
		for _, similar := range warning.Similar {
			if !similar.LiveLease {
				t.Fatal("nonlive work in claim warning", similar)
			}
		}
	}
	b.Config.DuplicateMode = core.DuplicateOff
	warnings = claimWarnings(b, current, now)
	if len(warnings) != 1 || warnings[0].Rule != "area_overlap" || len(warnings[0].Similar) != 5 {
		t.Fatal("off must disable duplicate advice and retain bounded area overlap", warnings)
	}
}

func TestEpicRelatedDismissRetainsPermissionsAndTaskOnlyDependencies(t *testing.T) {
	f := newFixture(t)
	a, b := similarEpics(t, f)
	input := core.LinkInput{RequestID: f.id(), From: a.Key, To: b.Key, Type: "related"}
	stranger := f.secondPerson(t, "bob", "developer")
	_, err := f.s.Link(testContext, stranger, input)
	requireCode(t, err, "FORBIDDEN_ROLE")
	for _, change := range []func(*core.LinkInput){
		func(in *core.LinkInput) { in.Type = "blocks" },
		func(in *core.LinkInput) { in.Type = "contract_of" },
		func(in *core.LinkInput) { in.Type = "spawned_by" },
		func(in *core.LinkInput) { in.Condition = "completed" },
		func(in *core.LinkInput) { in.Generation = 1 },
		func(in *core.LinkInput) { in.To = f.task(t, "developer").Key },
	} {
		invalid := input
		invalid.RequestID = f.id()
		change(&invalid)
		_, err = f.s.Link(testContext, f.coord, invalid)
		requireCode(t, err, "VALIDATION")
	}
	team, err := f.s.ControlTeam(testContext, f.coord)
	if err != nil || team.DuplicatesTotal != 1 {
		t.Fatal("epic pair missing", team.Duplicates, err)
	}
	input.RequestID = f.id()
	linked, err := f.s.Link(testContext, f.coord, input)
	if err != nil || len(linked.Links) != 1 || linked.Links[0].Type != "related" {
		t.Fatal("epic dismiss failed", linked, err)
	}
	team, err = f.s.ControlTeam(testContext, f.coord)
	if err != nil || team.DuplicatesTotal != 0 {
		t.Fatal("dismissed epic pair still visible", team.Duplicates, err)
	}
	input.RequestID, input.Unlink = f.id(), true
	if _, err = f.s.Link(testContext, f.coord, input); err != nil {
		t.Fatal("epic relation cannot be removed", err)
	}
	team, err = f.s.ControlTeam(testContext, f.coord)
	if err != nil || team.DuplicatesTotal != 1 {
		t.Fatal("unlinked epic pair not restored", team.Duplicates, err)
	}
}

func TestEpicDuplicateRestoreDetachesOnlyDuplicateRelation(t *testing.T) {
	f := newFixture(t)
	a, b := similarEpics(t, f)
	// A separate descriptive relation survives restore.
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: f.epic.Key, To: b.Key, Type: "related"})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		marked, err := f.s.Duplicate(testContext, f.coord, core.DuplicateInput{RequestID: f.id(), TaskKey: b.Key, CanonicalKey: a.Key})
		if err != nil || marked.State != "cancelled" || marked.DuplicateOf != a.Key || len(marked.Links) != 2 {
			t.Fatal("epic mark", marked, err)
		}
		in := core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: b.Key}, State: "open"}
		_, err = f.s.Transition(testContext, f.coord, in)
		requireCode(t, err, "INVALID_TRANSITION")
		still, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: b.Key})
		if err != nil || still.DuplicateOf != a.Key || still.State != "cancelled" {
			t.Fatal("invalid restore partially changed duplicate", still, err)
		}
		in.RequestID, in.Reason = f.id(), "Это отдельный эпик"
		restored, err := f.s.Transition(testContext, f.coord, in)
		if err != nil || restored.State != "open" || restored.DuplicateOf != "" || len(restored.Links) != 1 || restored.Links[0].Type != "related" {
			t.Fatal("epic restore", restored, err)
		}
		team, err := f.s.ControlTeam(testContext, f.coord)
		if err != nil || team.DuplicatesTotal != 1 {
			t.Fatal("restored epic pair missing", team.Duplicates, err)
		}
	}
}

func TestTeamUnclaimedMatchesAssignmentReadinessAfterEditAndArchive(t *testing.T) {
	for _, operation := range []string{"edit", "archive"} {
		t.Run(operation, func(t *testing.T) {
			f := newFixture(t)
			profile := createProfile(t, f)
			actor := boundActor(t, f, profile)
			task := assignedTask(t, f, profile)
			team, err := f.s.ControlTeam(testContext, f.coord)
			if err != nil || team.UnclaimedTotal != 1 || len(team.Unclaimed) != 1 || team.Unclaimed[0].Key != task.Key {
				t.Fatal("active assignment not available", team.Unclaimed, err)
			}
			if operation == "edit" {
				changed := settings()
				changed.Instructions = "Updated instructions"
				_, err = f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: f.id(), ProfileID: profile.ID, Version: profile.Version, ProfileSettings: changed})
			} else {
				_, err = f.s.ProfileArchive(testContext, f.coord, core.ProfileArchiveInput{RequestID: f.id(), ProfileID: profile.ID, Version: profile.Version, State: "archived"})
			}
			if err != nil {
				t.Fatal(err)
			}
			team, err = f.s.ControlTeam(testContext, f.coord)
			if err != nil || team.UnclaimedTotal != 0 || len(team.Unclaimed) != 0 {
				t.Fatal("unclaimable assignment listed", team.Unclaimed, err)
			}
			claim, err := f.s.Next(testContext, actor, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
			if err != nil || claim.Task != nil {
				t.Fatal("changed assignment claimable", claim, err)
			}
			_, err = f.s.Assign(testContext, f.coord, core.AssignInput{RequestID: f.id(), TaskKey: task.Key, Version: task.Version, Mode: "role_queue", Reason: "Return independent work"})
			if err != nil {
				t.Fatal(err)
			}
			team, err = f.s.ControlTeam(testContext, f.coord)
			if err != nil || team.UnclaimedTotal != 1 || len(team.Unclaimed) != 1 || team.Unclaimed[0].Key != task.Key {
				t.Fatal("explicit clear did not restore readiness", team.Unclaimed, err)
			}
		})
	}
}
