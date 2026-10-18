package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func (s *Store) Transition(ctx context.Context, a core.Actor, in core.TransitionInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "transition", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		t, err := taskFor(b, in.TaskKey)
		if err != nil {
			return core.Task{}, err
		}
		if in.TokensSpent != nil && (in.State != "done" || t.Type != "task") {
			return core.Task{}, core.Fail("invalid_input", "final tokens are only accepted on task completion")
		}
		if in.Owner != nil && t.Type != "epic" {
			return core.Task{}, core.Fail("VALIDATION", "owner only applies to epic handoff")
		}
		if len(in.Branch) > 512 {
			return core.Task{}, core.Fail("invalid_input", "branch too long")
		}
		if t.Type == "epic" {
			return transitionEpic(b, t, sess, in, now)
		}
		t.Warnings = nil
		switch {
		case in.State == "done" && t.State == "in_progress":
			if _, err = owned(b, a, t.Key, in.Generation); err != nil {
				return core.Task{}, err
			}
			if blocked(b, t) {
				return core.Task{}, core.Fail("rule_violation", "dependencies are not currently satisfied")
			}
			if err = completion(b, t, sess, in); err != nil {
				return core.Task{}, err
			}
			if total, present, tokenErr := t.Evidence.Integer("tokens_spent"); tokenErr != nil {
				return core.Task{}, core.Fail("VALIDATION", tokenErr.Error())
			} else if present {
				if in.TokensSpent != nil && *in.TokensSpent != total {
					return core.Task{}, core.Fail("VALIDATION", "tokens_spent totals disagree")
				}
				in.TokensSpent = &total
			}
			if err = finalTokens(t, in.TokensSpent); err != nil {
				return core.Task{}, err
			}
			if t.CreatedAt != nil && t.FirstDoneAt == nil {
				t.FirstDoneAt = &now
			}
			t.AuthorSession = a.SessionID
			finishAttempt(t, now, "completed", in.Branch)
			t.Lease = nil
			t.State = "done"
		case in.State == "ready" && t.State == "done":
			if !reopenAllowed(b, sess.Role, t.Role) {
				return core.Task{}, core.Fail("forbidden", "role cannot reopen this task")
			}
			if !bounded(in.Reason, 4096) {
				return core.Task{}, core.Fail("invalid_input", "reopen requires reason")
			}
			reference := b.Tasks[in.EvidenceRef]
			if reference == nil || reference.State != "done" || !hasEvidence(reference.Evidence) {
				return core.Task{}, core.Fail("evidence_missing", "reopen requires existing defect evidence")
			}
			t.ReturnsCount++
			t.State = "ready"
			t.ReadyAt = now
		case in.State == "cancelled" && (t.State == "ready" || t.State == "created"):
			if !canCancel(b, sess, t) {
				return core.Task{}, core.Fail("forbidden", "only requester or coordinator may cancel")
			}
			if !bounded(in.Reason, 4096) {
				return core.Task{}, core.Fail("invalid_input", "cancellation reason required")
			}
			t.State = "cancelled"
		case in.State == "ready" && (t.State == "created" || t.State == "cancelled"):
			if sess.Role != "coordinator" && t.Requester != a.Principal.ID {
				return core.Task{}, core.Fail("forbidden", "only requester or coordinator may make ready")
			}
			if t.State == "cancelled" && !bounded(in.Reason, 4096) {
				return core.Task{}, core.Fail("invalid_input", "restoration reason required")
			}
			if !bounded(t.Description, 16384) {
				return core.Task{}, core.Fail("VALIDATION", "ready task requires description")
			}
			if err := validateMetadata(b, t); err != nil {
				return core.Task{}, err
			}
			t.State = "ready"
			t.ReadyAt = now
		default:
			if t.Lease != nil && t.Lease.SessionID != a.SessionID {
				return core.Task{}, core.Fail("lease_held", "another session owns this task")
			}
			return core.Task{}, core.Fail("invalid_transition", "state transition is not allowed")
		}
		t.Version++
		if t.State != "done" {
			reopenEpic(b, t.EpicKey, now, a.SessionID)
		}
		emitWarnings(b, t, now, a.SessionID)
		emit(b, now, "task_"+t.State, t.Key, a.SessionID)
		if in.TokensSpent != nil {
			total := *in.TokensSpent
			correction := total - t.Attempts[len(t.Attempts)-1].TokensDelta
			b.Events[len(b.Events)-1].Data = &core.EventData{TokensSpentTotal: &total, CorrectionDelta: &correction}
		}
		return viewTask(b, t), nil
	})
}
func reopenEpic(b *board, key string, now time.Time, session string) {
	if epic := b.Tasks[key]; epic != nil && epic.State == "done" {
		epic = b.editTask(epic.Key)
		epic.State = "open"
		epic.Version++
		emit(b, now, "epic_reopened", key, session)
	}
}
func transitionEpic(b *board, t *core.Task, sess *core.Session, in core.TransitionInput, now time.Time) (core.Task, error) {
	if in.State == "cancelled" {
		if !canCancel(b, sess, t) {
			return core.Task{}, core.Fail("forbidden", "cancel permission or requester required")
		}
	} else if sess.Role != "coordinator" && sess.Role != "human" {
		return core.Task{}, core.Fail("forbidden", "epic transition requires coordinator or human")
	}
	if in.Owner != nil {
		if in.State != "open" {
			return core.Task{}, core.Fail("VALIDATION", "handoff transition state must be open")
		}
		if sess.Role != "coordinator" {
			return core.Task{}, core.Fail("FORBIDDEN_ROLE", "coordinator required for handoff")
		}
		return handoffEpic(b, t, *in.Owner, in.ExpectedVersion, in.Reason, sess.SessionID, now)
	}
	switch in.State {
	case "done":
		if t.State != "open" {
			return core.Task{}, core.Fail("invalid_transition", "only open epic may finish")
		}
		if err := epicClosable(b, t.Key); err != nil {
			return core.Task{}, err
		}
	case "cancelled":
		if t.State != "open" || !bounded(in.Reason, 4096) {
			return core.Task{}, core.Fail("invalid_transition", "open epic and cancellation reason required")
		}
		for _, child := range b.Tasks {
			if child.EpicKey == t.Key && child.Lease != nil {
				return core.Task{}, core.Fail("lease_held", "epic contains active lease")
			}
		}
	case "open":
		if t.State == "open" || !bounded(in.Reason, 4096) {
			return core.Task{}, core.Fail("invalid_transition", "reopen reason required")
		}
	default:
		return core.Task{}, core.Fail("invalid_transition", "invalid epic state")
	}
	if in.State == "done" && t.CreatedAt != nil && t.FirstDoneAt == nil {
		t.FirstDoneAt = &now
	}
	t.State = in.State
	t.Version++
	emit(b, now, "epic_"+in.State, t.Key, sess.SessionID)
	return viewTask(b, t), nil
}

func hasEvidence(e core.Evidence) bool {
	return len(e) > 0
}
