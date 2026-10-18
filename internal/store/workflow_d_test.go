package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"testing"
	"time"
)

func TestConfiguredHumanCategoryCheckedAtClaimAndDone(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		r := c.Roles["human"]
		r.Members = []string{"alice"}
		r.Categories = map[string]config.Category{"technical": {AnswerBy: json.RawMessage(`["alice"]`), ComplexityMax: "normal"}}
		c.Roles["human"] = r
	})
	_, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "question", Description: "question", Role: "human", Category: "technical"})
	requireCode(t, err, "VALIDATION")
	task, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "question", Description: "question", Role: "human", Category: "technical", Complexity: "high"})
	if err != nil {
		t.Fatal(err)
	}
	human := f.actor(t, "human")
	_, err = f.s.Next(testContext, human, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
	requireCode(t, err, "FORBIDDEN_ROLE")
	normal := "normal"
	task, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key}, Complexity: &normal})
	if err != nil {
		t.Fatal(err)
	}
	claim := f.claim(t, human, task.Key)
	high := "high"
	_, err = f.s.Update(testContext, human, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Complexity: &high})
	if err != nil {
		t.Fatal(err)
	}
	evidence := core.EvidenceOf(map[string]any{"answer": "decision"})
	_, err = f.s.Transition(testContext, human, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &evidence})
	requireCode(t, err, "FORBIDDEN_ROLE")
}
func TestDraftAreasOwnerOverlapAndDuplicates(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		c.Rules.EpicOwner.Mode = "warn"
		c.Rules.AreaOverlap.Mode = "warn"
		c.Rules.DuplicateCheck.Mode = "warn"
		c.Areas = []config.Area{{Name: "src", Aliases: []string{"code"}}, {Name: "only", Aliases: []string{}, SingleEditorRole: "developer"}}
	})
	ready := false
	draft, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "Straße   ändern", Role: "developer", Ready: &ready, Areas: []string{"code"}})
	if err != nil || draft.State != "created" || len(draft.Areas) != 1 || draft.Areas[0] != "src" {
		t.Fatal(draft, err)
	}
	_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: draft.Key}, State: "ready"})
	requireCode(t, err, "VALIDATION")
	description := "implement"
	_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: draft.Key}, Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: draft.Key}, State: "ready"})
	if err != nil {
		t.Fatal(err)
	}
	similar, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "STRASSE ändern", Description: "implement", Role: "developer", Areas: []string{"src/child"}})
	if err != nil || len(similar.Similar) != 1 || similar.Similar[0].Score != 1 || len(similar.Warnings) != 1 {
		t.Fatal(similar, err)
	}
	d := f.actor(t, "developer")
	f.claim(t, d, draft.Key)
	other := f.actor(t, "developer")
	claim := f.claim(t, other, similar.Key)
	if len(claim.Task.Warnings) != 1 || claim.Task.Warnings[0].Rule != "area_overlap" {
		t.Fatal("overlap omitted", claim)
	}
	forbidden, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "role restricted", Description: "implement", Role: "analyst", Areas: []string{"only"}})
	if err != nil {
		t.Fatal(err)
	}
	analyst := f.actor(t, "analyst")
	_, err = f.s.Next(testContext, analyst, core.NextInput{RequestID: f.id(), TaskKey: forbidden.Key})
	requireCode(t, err, "FORBIDDEN_ROLE")
	if titleDice("ΑΒΓ", "αβγ") != 1 || titleDice("ß", "SS") != 1 || titleDice("A\tB", "a b") != 1 {
		t.Fatal("Unicode/short title/whitespace mismatch")
	}
}
func TestDuplicateCancellationAndEpicDAG(t *testing.T) {
	f := newFixture(t)
	source, target := f.task(t, "developer"), f.task(t, "developer")
	linked, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "duplicate_of", Reason: "same issue"})
	if err != nil || linked.State != "cancelled" {
		t.Fatal(linked, err)
	}
	original, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: target.Key})
	if err != nil || original.State != "ready" {
		t.Fatal("duplicate changed target", err)
	}
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: f.epic.Key, To: target.Key, Type: "blocks"})
	requireCode(t, err, "CYCLE")
	otherEpic := f.newEpic(t)
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: f.epic.Key, To: otherEpic.Key, Type: "contract_of"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: otherEpic.Key, To: f.epic.Key, Type: "blocks"})
	requireCode(t, err, "CYCLE")
}
func TestDynamicRoleToolsAndSingletonGeneration(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		role := c.Roles["developer"]
		role.Singleton = true
		role.EvidenceOnDone = []string{"summary"}
		c.Roles["editor"] = role
		c.MCP.ToolsByRole = map[string][]string{}
	})
	editor := f.actor(t, "editor")
	_, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "editor", RunID: "parallel", Machine: f.p.Machine})
	requireCode(t, err, "ROLE_HELD")
	f.task(t, "editor")
	f.clock.advance(2 * time.Hour)
	replacement := f.actor(t, "editor")
	_, err = f.s.Heartbeat(testContext, editor, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
	result, err := f.s.Heartbeat(testContext, replacement, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 2})
	if err != nil || result.RoleGeneration != 2 {
		t.Fatal(result, err)
	}
	f.clock.advance(2 * time.Hour)
	f.configure(t, func(c *config.Config) { c.MCP.ToolsByRole["editor"] = []string{"describe", "heartbeat"} })
	editor = f.actor(t, "editor")
	_, err = f.s.Next(testContext, editor, core.NextInput{RequestID: f.id()})
	requireCode(t, err, "FORBIDDEN_ROLE")
}
