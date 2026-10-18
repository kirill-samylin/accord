package store

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func validRole(b *board, name string) bool { _, ok := b.Config.Workflow.Roles[name]; return ok }
func singleton(b *board, name string) bool { return b.Config.Workflow.Roles[name].Singleton }
func verifier(b *board, name string) bool {
	r, ok := b.Config.Workflow.Roles[name]
	if !ok {
		return false
	}
	if name == "reviewer" || name == "tester" || name == "security" || name == "design-reviewer" {
		return true
	}
	if len(r.IndependentFrom) == 0 {
		return false
	}
	return verifierOutcome(r) != ""
}
func verifierOutcome(r config.Role) string {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if json.Unmarshal(r.EvidenceSchema, &schema) != nil {
		return ""
	}
	_, sha := schema.Properties["verified_commit"]
	required := append(schema.Required, r.EvidenceOnDone...)
	verdict := slices.Contains(required, "verdict")
	result := slices.Contains(required, "result")
	if !sha || verdict == result {
		return ""
	}
	if verdict {
		return "verdict"
	}
	return "result"
}
func outcomeField(b *board, name string) string {
	if name == "tester" {
		return "result"
	}
	if name == "reviewer" || name == "security" || name == "design-reviewer" {
		return "verdict"
	}
	return verifierOutcome(b.Config.Workflow.Roles[name])
}
func toolAllowed(b *board, role, tool string) error {
	if !validRole(b, role) || !slices.Contains(b.Config.Workflow.MCP.ToolsByRole[role], tool) {
		return core.Fail("FORBIDDEN_ROLE", "role may not call "+tool)
	}
	return nil
}
func roleAuthorized(ctx context.Context, b *board, p core.Principal, role string) error {
	if !validRole(b, role) {
		return core.Fail("VALIDATION", "unknown configured role")
	}
	var raw []byte
	if err := b.tx.QueryRowContext(ctx, `SELECT roles FROM tokens WHERE id=? AND revoked=0`, p.ID).Scan(&raw); err != nil {
		return err
	}
	var roles []string
	if err := json.Unmarshal(raw, &roles); err != nil {
		return err
	}
	if !slices.Contains(roles, role) {
		return core.Fail("FORBIDDEN_ROLE", "token lacks requested role scope")
	}
	r := b.Config.Workflow.Roles[role]
	if r.Kind == "human" {
		if !slices.Contains(b.Config.Workflow.Auth.Local.Users, p.Owner) {
			return core.Fail("FORBIDDEN_ROLE", "human owner is not allowlisted")
		}
		if len(r.Members) > 0 && !slices.Contains(r.Members, p.Owner) {
			return core.Fail("FORBIDDEN_ROLE", "human owner is not a role member")
		}
	}
	return nil
}
func humanTaskAllowed(b *board, t *core.Task, owner string) error {
	r := b.Config.Workflow.Roles[t.Role]
	if r.Kind != "human" {
		return nil
	}
	if !slices.Contains(b.Config.Workflow.Auth.Local.Users, owner) || len(r.Members) > 0 && !slices.Contains(r.Members, owner) {
		return core.Fail("FORBIDDEN_ROLE", "human owner is not permitted")
	}
	if len(r.Categories) == 0 {
		return nil
	}
	category, ok := r.Categories[t.Category]
	if !ok {
		return core.Fail("VALIDATION", "configured human category required")
	}
	levels := map[string]int{"low": 1, "normal": 2, "high": 3}
	if t.Complexity == "" || levels[t.Complexity] > levels[category.ComplexityMax] {
		return core.Fail("FORBIDDEN_ROLE", "human complexity exceeds category limit")
	}
	if string(category.AnswerBy) != `"any"` {
		var allowed []string
		if err := json.Unmarshal(category.AnswerBy, &allowed); err != nil {
			return err
		}
		if !slices.Contains(allowed, owner) {
			return core.Fail("FORBIDDEN_ROLE", "owner cannot answer this category")
		}
	}
	return nil
}
func requireCreate(b *board, sess *core.Session, kind string) error {
	if !slices.Contains(b.Config.Workflow.Roles[sess.Role].CanCreate, kind) {
		return core.Fail("FORBIDDEN_ROLE", "role cannot create "+kind)
	}
	return nil
}
func canCancel(b *board, sess *core.Session, t *core.Task) bool {
	return b.Config.Workflow.Roles[sess.Role].CanCancel || t.Requester == sess.PrincipalID
}
func reopenAllowed(b *board, role, target string) bool {
	allowed := b.Config.Workflow.Reopen[role]
	return slices.Contains(allowed, "*") || slices.Contains(allowed, target)
}
func sessionTiming(b *board, v *core.SessionView) {
	v.ConfigVersion = b.Config.ConfigVersion
	v.Project = b.Config.Project
	v.SessionTTLSeconds = b.Config.SessionTTL.Seconds()
	v.RoleTTLSeconds = b.Config.RoleTTL.Seconds()
	v.TaskTTLSeconds = b.Config.LeaseTTL.Seconds()
	v.HeartbeatEverySeconds = min(b.Config.SessionTTL, b.Config.RoleTTL, b.Config.LeaseTTL).Seconds() / 3
}
func renew(b *board, t *core.Task, now time.Time) {
	if t.Lease != nil {
		t.Lease.ExpiresAt = now.Add(b.Config.LeaseTTL)
	}
}

func sessionLive(b *board, v *core.Session, now time.Time) bool {
	return v != nil && v.ExpiresAt.After(now) && (!singleton(b, v.Role) || (v.RoleExpiresAt == nil || v.RoleExpiresAt.After(now)) && v.RoleGeneration == b.Roles[v.Role])
}

// Historical completions keep their server-bound verifier kind even if an
// inactive role is removed from the current configuration.
func taskOutcomeField(b *board, t *core.Task) string {
	if t.Review != nil && t.Review.OutcomeField != "" {
		return t.Review.OutcomeField
	}
	if t.Role == "reviewer" || t.Role == "security" || t.Role == "design-reviewer" {
		return "verdict"
	}
	if t.Role == "tester" {
		return "result"
	}
	return outcomeField(b, t.Role)
}
