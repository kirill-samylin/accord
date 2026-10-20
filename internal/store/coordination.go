package store

import (
	"agent-board/internal/core"
	"context"
	"sort"
	"time"
)

func reason(code, message, key string) core.CoordinationReason {
	return core.CoordinationReason{Code: code, Message: message, TaskKey: key}
}
func routingReasons(b *board, t *core.Task) []core.CoordinationReason {
	out := []core.CoordinationReason{}
	if t.Type != "task" {
		return out
	}
	if t.State != "ready" {
		out = append(out, reason("task_not_ready", "Задача не находится в готовой очереди", t.Key))
	}
	if t.Lease != nil {
		out = append(out, reason("lease_held", "Задача уже имеет действующую аренду", t.Key))
	}
	epic := b.Tasks[t.EpicKey]
	if epic == nil || epic.State != "open" {
		out = append(out, reason("epic_not_open", "Эпик не открыт", t.EpicKey))
	}
	if a := t.Assignment; a != nil {
		p := b.Profiles[a.ProfileID]
		switch {
		case p == nil:
			out = append(out, reason("assignment_missing", "Назначенный профиль отсутствует", t.Key))
		case p.State != "active":
			out = append(out, reason("assignment_archived", "Назначенный профиль архивирован", t.Key))
		case !assignmentAvailable(b, t):
			out = append(out, reason("assignment_changed", "Профиль изменён; требуется явное переназначение", t.Key))
		}
	}
	return out
}

// Explain only annotates failures already decided by the shared gate evaluator.
// It cannot independently declare a dependency satisfied or grant a claim.
func explainBlocked(b *board, t *core.Task, seen map[string]bool, out *[]core.CoordinationReason) {
	if len(*out) > 200 {
		return
	}
	if !blocked(b, t) {
		return
	}
	if seen[t.Key] {
		return
	}
	seen[t.Key] = true
	if r := t.Review; r != nil {
		target := b.Tasks[r.TargetKey]
		switch {
		case target == nil || target.State != "done":
			appendCoordReason(out, reason("target_not_done", "Целевой результат ещё не завершён", r.TargetKey))
		case target.Evidence.Commit != r.TargetCommit:
			appendCoordReason(out, reason("target_sha_changed", "SHA целевого результата изменился", r.TargetKey))
		case r.TargetGeneration != 0 && target.Generation != r.TargetGeneration:
			appendCoordReason(out, reason("target_generation_changed", "Поколение целевого результата изменилось", r.TargetKey))
		default:
			explainBlocked(b, target, seen, out)
		}
	}
	for _, l := range t.Links {
		if len(*out) > 200 {
			return
		}
		if l.To != t.Key || l.Type != "blocks" && l.Type != "contract_of" || successful(b, b.Tasks[l.From], l.Condition) {
			continue
		}
		code, message := "dependency_unsatisfied", "Зависимость не имеет актуального успешного результата"
		source := b.Tasks[l.From]
		if source != nil && source.Review != nil {
			switch {
			case !activeVerification(b, source):
				code, message = "verification_superseded", "Проверка заменена новым циклом или её target устарел"
			case source.State != "done":
				code, message = "verification_pending", "Проверка ещё не завершена"
			case !checkTargetCurrent(b, source):
				code, message = "verification_stale", "Целевая версия результата проверки устарела"
			case !verificationCurrent(b, source):
				code, message = "verification_not_positive", "Проверка завершена без положительного результата"
			}
		}
		appendCoordReason(out, reason(code, message, l.From))
		if source != nil {
			explainBlocked(b, source, seen, out)
		}
	}
}
func (s *Store) Coordination(ctx context.Context, a core.Actor, in core.CoordinationInput) (core.TaskCoordination, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.TaskCoordination, error) {
		out := core.TaskCoordination{TaskKey: in.TaskKey, ObservedAt: now, RoutingReasons: []core.CoordinationReason{}, Blockers: []core.CoordinationReason{}, Dependencies: []core.CoordinationDependency{}, Checks: []core.CoordinationCheck{}, ClosureReasons: []core.CoordinationReason{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return out, err
		}
		t, err := taskFor(b, in.TaskKey)
		if err != nil {
			return out, err
		}
		out.TaskVersion = t.Version
		out.Assignment = t.Assignment
		out.RoutingReasons = routingReasons(b, t)
		explainBlocked(b, t, map[string]bool{}, &out.Blockers)
		for _, l := range t.Links {
			satisfied := true
			if l.Type == "blocks" || l.Type == "contract_of" {
				satisfied = successful(b, b.Tasks[l.From], l.Condition)
			}
			out.Dependencies = append(out.Dependencies, core.CoordinationDependency{Link: l, Satisfied: satisfied})
			if len(out.Dependencies) > 200 {
				sort.Slice(out.Dependencies, func(i, j int) bool { return dependencyBefore(out.Dependencies[i], out.Dependencies[j]) })
				out.Dependencies = out.Dependencies[:200]
				out.Truncated = true
			}
		}
		targets := map[string]bool{t.Key: true}
		if t.Review != nil {
			targets[t.Review.TargetKey] = true
		}
		if t.Type == "epic" {
			closable := epicClosable(b, t.Key) == nil
			out.EpicClosable = &closable
			if !closable {
				appendCoordReason(&out.ClosureReasons, reason("epic_gates_unsatisfied", "Есть незавершённая работа или обязательные непройденные проверки", t.Key))
			}
			for _, child := range b.Tasks {
				if child.EpicKey == t.Key {
					targets[child.Key] = true
					if child.Review != nil {
						targets[child.Review.TargetKey] = true
					}
					if child.Lease != nil || child.State != "done" && child.State != "cancelled" {
						appendCoordReason(&out.ClosureReasons, reason("unfinished_work", "Работа ещё не завершена", child.Key))
					}
				}
			}
		}
		for _, check := range b.Tasks {
			if check.Review == nil || !targets[check.Review.TargetKey] {
				continue
			}
			r := check.Review
			latest := latestVerification(b, r.TargetKey, check.Role)
			out.Checks = append(out.Checks, core.CoordinationCheck{TaskKey: check.Key, Role: check.Role, State: check.State, Outcome: r.Outcome, TargetCommit: r.TargetCommit, TargetGeneration: r.TargetGeneration, Current: checkTargetCurrent(b, check), Successful: successful(b, check, ""), Latest: latest != nil && latest.Key == check.Key})
			if len(out.Checks) > 200 {
				sort.Slice(out.Checks, func(i, j int) bool { return checkBefore(out.Checks[i], out.Checks[j]) })
				out.Checks = out.Checks[:200]
				out.Truncated = true
			}
			if t.Type == "epic" && !hasCurrentVerification(b, r.TargetKey, check.Role) {
				appendCoordReason(&out.ClosureReasons, reason("verification_required", "Нужна актуальная положительная проверка роли "+check.Role, r.TargetKey))
			}
		}
		normalizeCoordination(&out)
		return out, nil
	})
}
func normalizeReasons(items []core.CoordinationReason) ([]core.CoordinationReason, bool) {
	sort.Slice(items, func(i, j int) bool { return reasonBefore(items[i], items[j]) })
	out := make([]core.CoordinationReason, 0, len(items))
	for _, v := range items {
		if len(out) == 0 || out[len(out)-1].TaskKey != v.TaskKey || out[len(out)-1].Code != v.Code || out[len(out)-1].Message != v.Message {
			out = append(out, v)
		}
	}
	truncated := len(out) > 200
	if truncated {
		out = out[:200]
	}
	return out, truncated
}
func normalizeCoordination(out *core.TaskCoordination) {
	for _, items := range []*[]core.CoordinationReason{&out.RoutingReasons, &out.Blockers, &out.ClosureReasons} {
		v, truncated := normalizeReasons(*items)
		*items = v
		out.Truncated = out.Truncated || truncated
	}
	sort.Slice(out.Dependencies, func(i, j int) bool { return dependencyBefore(out.Dependencies[i], out.Dependencies[j]) })
	sort.Slice(out.Checks, func(i, j int) bool { return checkBefore(out.Checks[i], out.Checks[j]) })
	if len(out.Dependencies) > 200 {
		out.Dependencies = out.Dependencies[:200]
		out.Truncated = true
	}
	if len(out.Checks) > 200 {
		out.Checks = out.Checks[:200]
		out.Truncated = true
	}
}

func checkTargetCurrent(b *board, check *core.Task) bool {
	r := check.Review
	if r == nil {
		return false
	}
	target := b.Tasks[r.TargetKey]
	return target != nil && target.State == "done" && target.Evidence.Commit == r.TargetCommit && target.Generation == r.TargetGeneration
}

func appendCoordReason(out *[]core.CoordinationReason, v core.CoordinationReason) {
	for _, existing := range *out {
		if existing.Code == v.Code && existing.TaskKey == v.TaskKey && existing.Message == v.Message {
			return
		}
	}
	*out = append(*out, v)
	if len(*out) > 201 {
		sort.Slice(*out, func(i, j int) bool { return reasonBefore((*out)[i], (*out)[j]) })
		*out = (*out)[:201]
	}
}

func dependencyBefore(a, b core.CoordinationDependency) bool {
	if a.Link.From != b.Link.From {
		return numericKey(a.Link.From) < numericKey(b.Link.From)
	}
	return a.Link.Type < b.Link.Type
}
func checkBefore(a, b core.CoordinationCheck) bool {
	if a.Latest != b.Latest {
		return a.Latest
	}
	if a.Current != b.Current {
		return a.Current
	}
	return numericKey(a.TaskKey) < numericKey(b.TaskKey)
}

func reasonBefore(a, b core.CoordinationReason) bool {
	if a.TaskKey != b.TaskKey {
		an, bn := numericKey(a.TaskKey), numericKey(b.TaskKey)
		if an != bn {
			return an < bn
		}
		return a.TaskKey < b.TaskKey
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}
