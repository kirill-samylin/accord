package store

import (
	"agent-board/internal/core"
	"time"
)

func duplicateLink(b *board, source, target *core.Task, sess *core.Session, in core.LinkInput, now time.Time) (core.Task, error) {
	if in.Unlink || in.Condition != "" || source.Key == target.Key || source.Type != target.Type || !bounded(in.Reason, 4096) {
		return core.Task{}, core.Fail("VALIDATION", "duplicate_of requires distinct same-kind objects and reason")
	}
	if !canCancel(b, sess, source) {
		return core.Task{}, core.Fail("FORBIDDEN_ROLE", "cancel authority required")
	}
	if source.Lease != nil {
		return core.Task{}, core.Fail("LEASE_HELD", "duplicate source has live lease")
	}
	for _, child := range b.Tasks {
		if source.Type == "epic" && child.EpicKey == source.Key && child.Lease != nil {
			return core.Task{}, core.Fail("LEASE_HELD", "duplicate epic has live child lease")
		}
	}
	for _, link := range source.Links {
		if link.Type == "duplicate_of" {
			return core.Task{}, core.Fail("CONFLICT", "source already has duplicate_of link")
		}
	}
	if terminalState(source.State) {
		return core.Task{}, core.Fail("INVALID_TRANSITION", "duplicate source must be nonterminal")
	}
	source.Links = append(source.Links, core.Link{From: source.Key, To: target.Key, Type: "duplicate_of"})
	source.State = "cancelled"
	source.Version++
	emit(b, now, "duplicate_linked", source.Key, sess.SessionID)
	b.Events[len(b.Events)-1].Reason = in.Reason
	return viewTask(b, source), nil
}
