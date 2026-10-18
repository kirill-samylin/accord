package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestSecurityDesignAndCustomVerifierObligationsStayGlobal(t *testing.T) {
	f := newFixture(t)
	template, result := config.Parse(bytes.ReplaceAll(config.DavinciTemplate(), []byte("OWNER"), []byte("alice")), config.Options{})
	if !result.Valid {
		t.Fatal(result)
	}
	resetConfig(t, f, func(c *config.Config) {
		d := template.Decode()
		for _, role := range []string{"security", "design-reviewer", "design-owner"} {
			c.Roles[role] = d.Roles[role]
		}
		custom := c.Roles["reviewer"]
		custom.EvidenceOnDone = []string{"result", "verified_commit", "summary"}
		custom.IndependentFrom = []string{"developer"}
		c.Roles["audit"] = custom
		c.MCP.ToolsByRole = map[string][]string{}
	})
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, target.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked"}), "")
	targetEpic := f.epic.Key
	verificationEpic := f.newEpic(t)
	f.epic = verificationEpic
	for _, role := range []string{"security", "design-reviewer", "audit"} {
		task := f.verification(t, role, target.Key, commitA)
		actor := f.actor(t, role)
		claim := f.claim(t, actor, task.Key)
		fields := map[string]any{"verified_commit": commitA, "findings": "reviewed", "screens": []string{"screen.png"}, "summary": "audited"}
		negative, positive := "changes_requested", "approve"
		if role == "audit" {
			negative, positive = "failed", "passed"
		}
		f.finish(t, actor, claim, core.EvidenceOf(fields), negative)
		_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: targetEpic}, State: "done"})
		requireCode(t, err, "RULE_VIOLATION")
		replacement := f.verification(t, role, target.Key, commitA)
		f.finish(t, actor, f.claim(t, actor, replacement.Key), core.EvidenceOf(fields), positive)
	}
	f.closeEpic(t, targetEpic)
	late := f.verification(t, "security", target.Key, commitA)
	got, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: targetEpic})
	if err != nil || got.State != "open" {
		t.Fatal("new global check did not reopen epic", got, err)
	}
	security := f.actor(t, "security")
	claim := f.claim(t, security, late.Key)
	evidence := core.EvidenceOf(map[string]any{"verified_commit": commitA, "verdict": "approve", "findings": "current"})
	_, err = f.s.Transition(testContext, security, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: late.Key, Generation: claim.Lease.Generation}, State: "done", Outcome: "changes_requested", Evidence: &evidence})
	requireCode(t, err, "VALIDATION")
}
func TestWarnExtraEvidenceCannotOpenCompletedGateAndExactNumbers(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		r := c.Roles["developer"]
		r.EvidenceOnDone = append(r.EvidenceOnDone, "pr")
		c.Roles["developer"] = r
		c.Rules.EvidenceRequired.Mode = "warn"
	})
	dev := f.actor(t, "developer")
	source := f.task(t, "developer")
	consumer := f.task(t, "developer")
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: source.Key, To: consumer.Key, Type: "blocks", Condition: "completed"})
	if err != nil {
		t.Fatal(err)
	}
	evidence := core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked", "tokens_spent": json.Number("9007199254740993")})
	done := f.finish(t, dev, f.claim(t, dev, source.Key), evidence, "")
	if done.EvidenceComplete == nil || *done.EvidenceComplete || len(done.Warnings) != 1 {
		t.Fatal(done)
	}
	if done.Attempts[0].TokensSpent == nil || *done.Attempts[0].TokensSpent != 9007199254740993 {
		t.Fatal("precise evidence total lost")
	}
	claim, err := f.s.Next(testContext, dev, core.NextInput{RequestID: f.id(), TaskKey: consumer.Key})
	if err != nil || claim.Task != nil {
		t.Fatal("incomplete evidence opened gate", claim, err)
	}
}
func TestHandoffCASRejectsLiveWorkAndNeverTransfersLease(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc", TTL: 24 * 60 * 60 * 1000000000}); err != nil {
		t.Fatal(err)
	}
	dev := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, dev, task.Key)
	_, err := f.s.HandoffEpic(testContext, f.epic.Key, "bob", f.epic.Version, "handoff", "admin")
	requireCode(t, err, "LEASE_HELD")
	_, err = f.s.Release(testContext, dev, core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "handoff"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.s.HandoffEpic(testContext, f.epic.Key, "bob", f.epic.Version, "handoff", "admin")
	if err != nil || out.Owner != "bob" {
		t.Fatal(out, err)
	}
	_, err = f.s.HandoffEpic(testContext, f.epic.Key, "alice", f.epic.Version, "stale", "admin")
	requireCode(t, err, "CONFLICT")
}

func TestRemovingInactiveVerifierDoesNotEraseHistoricalGate(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		r := c.Roles["reviewer"]
		c.Roles["audit"] = r
		c.MCP.ToolsByRole = map[string][]string{}
	})
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	f.finish(t, dev, f.claim(t, dev, target.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked"}), "")
	check := f.verification(t, "audit", target.Key, commitA)
	audit := f.actor(t, "audit")
	f.finish(t, audit, f.claim(t, audit, check.Key), core.EvidenceOf(map[string]any{"verified_commit": commitA, "findings": "checked"}), "approve")
	f.clock.advance(2 * time.Hour)
	f.configure(t, func(c *config.Config) { delete(c.Roles, "audit"); delete(c.MCP.ToolsByRole, "audit") })
	f.coord = f.actor(t, "coordinator")
	consumer := f.task(t, "developer")
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: check.Key, To: consumer.Key, Type: "blocks", Condition: "approved"})
	if err != nil {
		t.Fatal(err)
	}
	dev = f.actor(t, "developer")
	if claim, err := f.s.Next(testContext, dev, core.NextInput{RequestID: f.id(), TaskKey: consumer.Key}); err != nil || claim.Task == nil {
		t.Fatal("historical verifier lost gate after deletion", claim, err)
	}
}
