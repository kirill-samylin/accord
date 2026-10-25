package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func cycle(b *board, from, to string) bool {
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(key string) bool {
		if key == from {
			return true
		}
		if seen[key] {
			return false
		}
		seen[key] = true
		for _, t := range b.Tasks {
			for _, l := range t.Links {
				if l.From == key && (l.Type == "blocks" || l.Type == "contract_of") && visit(l.To) {
					return true
				}
			}
		}
		return false
	}
	return visit(to)
}
func (s *Store) Link(ctx context.Context, a core.Actor, in core.LinkInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "link", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		// No relation type is meaningful from an item to itself; unlink of such a
		// legacy row stays possible so it can be cleaned up.
		if in.From == in.To && !in.Unlink {
			return core.Task{}, core.Fail("invalid_input", "a link must connect two different items")
		}
		source, err := taskFor(b, in.From)
		if err != nil {
			return core.Task{}, err
		}
		target, err := taskFor(b, in.To)
		if err != nil {
			return core.Task{}, err
		}
		// duplicate_of: From is the duplicate (cancelled), To the canonical item.
		if in.Type == "duplicate_of" {
			if in.Unlink || in.Generation != 0 || in.Condition != "" {
				return core.Task{}, core.Fail("invalid_input", "duplicate_of accepts only from/to; restore a duplicate with transition to ready")
			}
			return s.markDuplicate(b, sess, a, in.From, in.To, "", 0, now)
		}
		// Epics can dismiss a suspected duplicate pair with a descriptive relation;
		// dependency and lease-generation semantics remain task-only.
		epicRelated := source.Type == "epic" && target.Type == "epic" && in.Type == "related" && in.Condition == "" && in.Generation == 0
		if !epicRelated && (source.Type != "task" || target.Type != "task") {
			return core.Task{}, core.Fail("invalid_input", "pilot links require tasks")
		}

		if target.Lease != nil || in.Generation != 0 {
			if _, err = owned(b, a, target.Key, in.Generation); err != nil {
				return core.Task{}, err
			}
		} else if !administrative(sess, target, a) {
			return core.Task{}, core.Fail("forbidden", "only coordinator or requester may link ready work")
		}
		if in.Unlink {
			if !administrative(sess, target, a) && (target.Lease == nil || target.Lease.SessionID != a.SessionID) {
				return core.Task{}, core.Fail("forbidden", "only coordinator, consumer requester or holder may unlink")
			}
			for i, l := range target.Links {
				if l.From == in.From && l.To == in.To && l.Type == in.Type {
					target.Links = append(target.Links[:i], target.Links[i+1:]...)
					target.Version++
					s.renew(target, now)
					emit(b, now, "unlinked", target.Key, a.SessionID)
					return viewTask(b, target), nil
				}
			}
			return core.Task{}, core.Fail("not_found", "link does not exist")
		}
		switch in.Type {
		case "blocks", "contract_of", "spawned_by", "related":
		default:
			return core.Task{}, core.Fail("invalid_input", "unsupported link type")
		}
		switch in.Condition {
		case "", "completed", "approved", "passed":
		default:
			return core.Task{}, core.Fail("invalid_input", "unsupported link condition")
		}
		if in.Condition == "approved" && source.Role != "reviewer" || in.Condition == "passed" && source.Role != "tester" {
			return core.Task{}, core.Fail("invalid_input", "condition incompatible with source role")
		}
		if (in.Type == "blocks" || in.Type == "contract_of") && cycle(b, in.From, in.To) {
			return core.Task{}, core.Fail("cycle", "dependency creates a cycle")
		}
		for _, l := range target.Links {
			if l.From == in.From && l.To == in.To && l.Type == in.Type {
				return core.Task{}, core.Fail("rule_violation", "link already exists")
			}
		}
		target.Links = append(target.Links, core.Link{From: in.From, To: in.To, Type: in.Type, Condition: in.Condition})
		target.Version++
		s.renew(target, now)
		emit(b, now, "linked", target.Key, a.SessionID)
		return viewTask(b, target), nil
	})
}
