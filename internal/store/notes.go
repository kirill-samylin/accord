package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"agent-board/internal/core"
)

func defaultNoteLimits(l *core.NoteLimits) {
	if l.MaxPerRole == 0 {
		l.MaxPerRole = 30
	}
	if l.MaxBytesPerRole == 0 {
		l.MaxBytesPerRole = 24000
	}
	if l.MaxPerEpic == 0 {
		l.MaxPerEpic = 20
	}
	if l.MaxBytesPerEpic == 0 {
		l.MaxBytesPerEpic = 16000
	}
}

// Note stores provenance and soft deletion in the same idempotent transaction.
func (s *Store) Note(ctx context.Context, a core.Actor, in core.NoteInput) (core.Note, error) {
	return mutate(s, ctx, a, in.RequestID, "note", in, func(b *board, sess *core.Session, now time.Time) (core.Note, error) {
		if in.Action == "delete" {
			if in.Text != "" || in.ScopeType != "" || in.ScopeID != "" || in.SourceTaskKey != "" {
				return core.Note{}, core.Fail("invalid_input", "delete accepts only key and reason")
			}
			n := b.Notes[in.Key]
			if n == nil {
				return core.Note{}, core.Fail("not_found", "note does not exist")
			}
			if n.AuthorPrincipal != a.Principal.ID && !b.Config.Workflow.Roles[sess.Role].CanCurateNotes {
				return core.Note{}, core.Fail("forbidden", "only author or note curator may delete")
			}
			if !bounded(in.Reason, 4096) {
				return core.Note{}, core.Fail("invalid_input", "deletion reason required")
			}
			if n.DeletedAt != nil {
				return core.Note{}, core.Fail("invalid_transition", "note already deleted")
			}
			n.DeletedAt = &now
			n.DeleteReason = in.Reason
			emit(b, now, "note_deleted", n.SourceTaskKey, a.SessionID)
			b.Events[len(b.Events)-1].Reason = in.Reason
			b.Events[len(b.Events)-1].Data = &core.EventData{NoteKey: n.Key, ScopeType: n.ScopeType, ScopeID: n.ScopeID}
			return *n, nil
		}
		if in.Key != "" || in.Reason != "" {
			return core.Note{}, core.Fail("invalid_input", "create does not accept key or reason")
		}
		if in.Action != "create" {
			return core.Note{}, core.Fail("invalid_input", "action must be create or delete")
		}
		if !bounded(in.Text, 2048) || !utf8.ValidString(in.Text) {
			return core.Note{}, core.Fail("invalid_input", "note text must be UTF8 and contain 1..2048 bytes")
		}
		source, err := taskFor(b, in.SourceTaskKey)
		if err != nil {
			return core.Note{}, err
		}
		if source.Type != "task" {
			return core.Note{}, core.Fail("invalid_input", "note source must be a task")
		}
		accessible := source.Role == sess.Role || source.Requester == a.Principal.ID || sess.Role == "coordinator"
		if !accessible {
			return core.Note{}, core.Fail("forbidden", "source task is not accessible to this role")
		}
		countLimit, byteLimit := b.Config.Notes.MaxPerRole, b.Config.Notes.MaxBytesPerRole
		switch in.ScopeType {
		case "role":
			if !validRole(b, in.ScopeID) {
				return core.Note{}, core.Fail("invalid_input", "unknown note role")
			}
			if in.ScopeID != sess.Role {
				return core.Note{}, core.Fail("forbidden", "may create notes only for own role")
			}
		case "epic":
			epic := b.Tasks[in.ScopeID]
			if epic == nil || epic.Type != "epic" || source.EpicKey != epic.Key {
				return core.Note{}, core.Fail("invalid_input", "source task must belong to note epic")
			}
			countLimit, byteLimit = b.Config.Notes.MaxPerEpic, b.Config.Notes.MaxBytesPerEpic
		default:
			return core.Note{}, core.Fail("invalid_input", "scope_type must be role or epic")
		}
		existing := scopeNotes(b, in.ScopeType, in.ScopeID)
		bytes := len(in.Text)
		keys := make([]string, 0, len(existing))
		for _, n := range existing {
			bytes += len(n.Text)
			keys = append(keys, n.Key)
		}
		if len(existing) >= countLimit || bytes > byteLimit {
			return core.Note{}, core.Fail("limit_exceeded", "note quota exceeded; oldest keys: "+strings.Join(keys, ","))
		}
		b.NoteSequence++
		n := &core.Note{Key: fmt.Sprintf("N-%d", b.NoteSequence), ScopeType: in.ScopeType, ScopeID: in.ScopeID, Text: in.Text, SourceTaskKey: source.Key, AuthorPrincipal: a.Principal.ID, AuthorSession: a.SessionID, CreatedAt: now}
		b.Notes[n.Key] = n
		emit(b, now, "note_created", source.Key, a.SessionID)
		b.Events[len(b.Events)-1].Data = &core.EventData{NoteKey: n.Key, ScopeType: n.ScopeType, ScopeID: n.ScopeID}
		return *n, nil
	})
}
func scopeNotes(b *board, kind, id string) []core.Note {
	result := []core.Note{}
	for _, n := range b.Notes {
		if n.ScopeType == kind && n.ScopeID == id && n.DeletedAt == nil {
			result = append(result, *n)
		}
	}
	sort.Slice(result, func(i, j int) bool { return numericKey(result[i].Key) < numericKey(result[j].Key) })
	return result
}
func sessionQueueSummary(b *board, sess *core.Session, now time.Time) core.QueueSummary {
	return queueSummary(b, sess.Role, sess, now)
}
func queueSummary(b *board, role string, sess *core.Session, now time.Time) core.QueueSummary {
	var q core.QueueSummary
	for _, t := range b.Tasks {
		if t.Type != "task" || role != "" && role != t.Role {
			continue
		}
		addToQueue(&q, b, t, sess, now)
	}
	return q
}

// Identity-dependent eligibility belongs to session views; global statistics
// still enforce the area's role authorization and blocking overlap rules.
func addToQueue(q *core.QueueSummary, b *board, t *core.Task, sess *core.Session, now time.Time) {
	switch t.State {
	case "in_progress":
		q.InProgress++
	case "created":
		q.Waiting++
	case "ready":
		epic := b.Tasks[t.EpicKey]
		eligible := areaRoleAllowed(b, t.Areas, t.Role) == nil && overlapAllowed(b, t, now) == nil
		if sess != nil {
			eligible = eligibleClaim(b, t, sess, now) == nil
		}
		if !eligible || blocked(b, t) || epic == nil || epic.State != "open" {
			q.Waiting++
		} else {
			q.Ready++
		}
	}
}
