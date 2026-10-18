package store

import (
	"agent-board/internal/core"
	"sort"
	"strconv"
	"strings"
	"time"
)

// verificationCurrent checks provenance only. Historical negative/stale checks
// remain auditable, but may be replaced by a new current positive cycle.
func verificationCurrent(b *board, t *core.Task) bool {
	if t.Review == nil {
		return false
	}
	target := b.Tasks[t.Review.TargetKey]
	if target == nil || target.State != "done" || target.Evidence.Text("commit") != t.Review.TargetCommit || target.Generation != t.Review.TargetGeneration {
		return false
	}
	return taskOutcomeField(b, t) == "verdict" && t.Review.Outcome == "approve" || taskOutcomeField(b, t) == "result" && t.Review.Outcome == "passed"
}

func successful(b *board, t *core.Task, condition string) bool {
	return gateSuccessful(b, t, condition, map[string]bool{})
}

func gateSuccessful(b *board, t *core.Task, condition string, visiting map[string]bool) bool {
	if t == nil || t.State != "done" || visiting[t.Key] || t.EvidenceComplete != nil && !*t.EvidenceComplete {
		return false
	}
	if t.Review != nil {
		if !verificationCurrent(b, t) || !activeVerification(b, t) || taskOutcomeField(b, t) == "verdict" && condition == "passed" || taskOutcomeField(b, t) == "result" && condition == "approved" {
			return false
		}
	} else if condition != "" && condition != "completed" {
		return false
	}
	return !gateBlocked(b, t, visiting)
}

func blocked(b *board, t *core.Task) bool {
	return gateBlocked(b, t, map[string]bool{})
}

// The newest applicable assignment supersedes older results of the same role.
// A pending or negative check cannot be bypassed by a previous approve/pass.
// T-N is the monotonic server creation sequence, not a lexical ordering.
func hasCurrentVerification(b *board, targetKey, role string) bool {
	latest := latestVerification(b, targetKey, role)
	return latest != nil && successful(b, latest, "")
}

func activeVerification(b *board, t *core.Task) bool {
	latest := latestVerification(b, t.Review.TargetKey, t.Role)
	return latest != nil && latest.Key == t.Key
}

func latestVerification(b *board, targetKey, role string) *core.Task {
	target := b.Tasks[targetKey]
	if target == nil || target.State != "done" {
		return nil
	}
	var latest *core.Task
	var latestOrder int64
	for _, candidate := range b.Tasks {
		r := candidate.Review
		if candidate.Role != role || r == nil || r.TargetKey != targetKey || r.TargetCommit != target.Evidence.Text("commit") || r.TargetGeneration != 0 && r.TargetGeneration != target.Generation {
			continue
		}
		number, ok := strings.CutPrefix(candidate.Key, "T-")
		order, err := strconv.ParseInt(number, 10, 64)
		if !ok || err != nil || order <= 0 {
			return nil
		}
		if order > latestOrder {
			latest, latestOrder = candidate, order
		}
	}
	return latest
}

// Follow dependencies even through completed tasks: keeping a child's done
// history must not make its invalidated result satisfy downstream gates.
// Implicit review-target edges can form cycles; they fail closed as well.
func gateBlocked(b *board, t *core.Task, visiting map[string]bool) bool {
	if visiting[t.Key] {
		return true
	}
	visiting[t.Key] = true
	defer delete(visiting, t.Key)
	if t.Review != nil {
		target := b.Tasks[t.Review.TargetKey]
		if target == nil || target.State != "done" || target.Evidence.Text("commit") != t.Review.TargetCommit || (t.Review.TargetGeneration != 0 && target.Generation != t.Review.TargetGeneration) || gateBlocked(b, target, visiting) {
			return true
		}
	}
	if t.Type == "epic" {
		for _, child := range b.Tasks {
			if child.EpicKey == t.Key && child.State != "cancelled" && !(child.State == "done" && child.Review != nil && (!verificationCurrent(b, child) || !activeVerification(b, child))) && !gateSuccessful(b, child, "", visiting) {
				return true
			}
		}
	}
	for _, link := range t.Links {
		if link.To == t.Key && (link.Type == "blocks" || link.Type == "contract_of") && !gateSuccessful(b, b.Tasks[link.From], link.Condition, visiting) {
			return true
		}
	}
	return false
}

// epicClosable is shared by explicit closure and reconciliation. Obligations
// belong to developer targets globally, regardless of a verifier's epic.
func epicClosable(b *board, key string) error {
	if epic := b.Tasks[key]; epic != nil {
		for _, link := range epic.Links {
			if link.To == key && (link.Type == "blocks" || link.Type == "contract_of") && !successful(b, b.Tasks[link.From], link.Condition) {
				return core.Fail("RULE_VIOLATION", "epic dependencies are unsatisfied")
			}
		}
	}
	targets := map[string]bool{}
	for _, child := range b.Tasks {
		if child.EpicKey != key {
			continue
		}
		if child.Lease != nil || child.State != "done" && child.State != "cancelled" {
			return core.Fail("rule_violation", "epic contains unfinished work")
		}
		if child.Evidence.Text("commit") != "" {
			targets[child.Key] = true
		}
		if child.Review != nil {
			targets[child.Review.TargetKey] = true
		}
		if child.State == "done" && !successful(b, child, "") && (child.Review == nil || verificationCurrent(b, child) && activeVerification(b, child)) {
			return core.Fail("rule_violation", "epic contains unsatisfied gates")
		}
	}
	for _, check := range b.Tasks {
		if check.Review != nil && targets[check.Review.TargetKey] && !hasCurrentVerification(b, check.Review.TargetKey, check.Role) {
			return core.Fail("rule_violation", "epic requires current positive verification for each required role")
		}
	}
	return nil
}

// Run inside the successful mutation's transaction, before persisting its
// idempotent response. Recursive gate evaluation covers cross-epic cascades
// without rewriting completed child attempts or their evidence.
func reconcileEpics(b *board, now time.Time, session string) {
	keys := make([]string, 0)
	for key, task := range b.Tasks {
		if task.Type == "epic" && task.State == "done" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if epicClosable(b, key) != nil {
			reopenEpic(b, key, now, session)
		}
	}
}
