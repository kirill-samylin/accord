package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"cmp"
	"slices"
	"time"
)

func foreignEpic(b *board, t *core.Task, sess *core.Session) bool {
	e := b.Tasks[t.EpicKey]
	return e != nil && e.Owner != "" && e.Owner != sess.Owner
}
func ownerExempt(b *board, sess *core.Session) bool {
	return singleton(b, sess.Role) || slices.Contains(b.Config.Workflow.Rules.EpicOwner.SharedRoles, sess.Role)
}
func eligibleClaim(b *board, t *core.Task, sess *core.Session, now time.Time) error {
	if err := humanTaskAllowed(b, t, sess.Owner); err != nil {
		return err
	}
	if b.Config.Workflow.Rules.EpicOwner.Mode == "block" && foreignEpic(b, t, sess) && !ownerExempt(b, sess) {
		return core.Fail("RULE_VIOLATION", "epic belongs to another owner")
	}
	if err := areaRoleAllowed(b, t.Areas, sess.Role); err != nil {
		return err
	}
	return overlapAllowed(b, t, now)
}

func areaRoleAllowed(b *board, paths []string, role string) error {
	if len(paths) == 0 {
		return nil
	}
	for _, area := range b.Config.Workflow.Areas {
		if area.SingleEditorRole == "" || area.SingleEditorRole == role {
			continue
		}
		for _, path := range paths {
			if config.AreasIntersect(area.Name, path) {
				return core.Fail("FORBIDDEN_ROLE", "area requires single editor role "+area.SingleEditorRole)
			}
		}
	}
	return nil
}
func claimRules(b *board, t *core.Task, sess *core.Session, now time.Time) error {
	if err := eligibleClaim(b, t, sess, now); err != nil {
		return err
	}
	if b.Config.Workflow.Rules.EpicOwner.Mode == "warn" && foreignEpic(b, t, sess) {
		ruleWarning(b, t, "epic_owner", "claiming work belonging to another owner", sess.SessionID)
	}
	if err := checkAreaOverlap(b, t, now); err != nil {
		return err
	}
	if epic := b.Tasks[t.EpicKey]; epic != nil && epic.Owner == "" {
		epic.Owner = sess.Owner
		epic.Version++
		emit(b, now, "epic_owner_assigned", epic.Key, sess.SessionID)
	}
	return nil
}
func prepareAreas(b *board, t *core.Task) error {
	seen := map[string]bool{}
	areas := make([]string, 0, len(t.Areas))
	for _, area := range t.Areas {
		canonical, err := b.Config.Snapshot.ResolveArea(area)
		if err != nil {
			return core.Fail("VALIDATION", err.Error())
		}
		if seen[canonical] {
			return core.Fail("VALIDATION", "duplicate canonical area")
		}
		seen[canonical] = true
		areas = append(areas, canonical)
	}
	t.Areas = areas
	return nil
}

// Eligibility checks never emit warnings or assign an epic owner. Queue scans
// may inspect and skip a candidate without changing its persisted state.
func overlapAllowed(b *board, t *core.Task, now time.Time) error {
	if b.Config.Workflow.Rules.AreaOverlap.Mode == "block" {
		if overlaps := areaOverlaps(b, t, now); len(overlaps) > 0 {
			return core.Fail("AREA_OVERLAP", "live task "+overlaps[0].TaskKey+" session:"+overlaps[0].SessionID)
		}
	}
	return nil
}
func checkAreaOverlap(b *board, t *core.Task, now time.Time) error {
	if err := overlapAllowed(b, t, now); err != nil {
		return err
	}
	if b.Config.Workflow.Rules.AreaOverlap.Mode == "warn" {
		t.Warnings = append(t.Warnings, areaOverlaps(b, t, now)...)
	}
	return nil
}
func areaOverlaps(b *board, t *core.Task, now time.Time) []core.RuleWarning {
	if len(t.Areas) == 0 {
		return nil
	}
	warnings := []core.RuleWarning{}
	for _, other := range b.Tasks {
		if other.Key == t.Key || other.Lease == nil || !other.Lease.ExpiresAt.After(now) {
			continue
		}
		otherSession := b.Sessions[other.Lease.SessionID]
		if !sessionLive(b, otherSession, now) {
			continue
		}
		intersect := []string{}
		for _, a := range t.Areas {
			for _, z := range other.Areas {
				if config.AreasIntersect(a, z) {
					intersect = append(intersect, a)
				}
			}
		}
		if len(intersect) > 0 {
			warnings = append(warnings, core.RuleWarning{Rule: "area_overlap", Message: "areas overlap a live lease", TaskKey: other.Key, SessionID: other.Lease.SessionID, Owner: otherSession.Owner, Areas: intersect})
		}
	}
	slices.SortFunc(warnings, func(a, z core.RuleWarning) int { return cmp.Compare(numericKey(a.TaskKey), numericKey(z.TaskKey)) })
	return warnings
}
func reviewTargetAllowed(b *board, role, target string) bool {
	targets := b.Config.Workflow.Roles[role].IndependentFrom
	if len(targets) > 0 {
		return slices.Contains(targets, target)
	}
	if role == "design-reviewer" {
		return target == "developer" || target == "design-owner"
	}
	if role == "reviewer" || role == "security" || role == "tester" {
		return target == "developer"
	}
	return false
}
