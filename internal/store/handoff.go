package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func handoffEpic(b *board, t *core.Task, owner string, version int64, reason, session string, now time.Time) (core.Task, error) {
	if t.Type != "epic" || t.State != "open" || !bounded(owner, 128) || !bounded(reason, 4096) {
		return core.Task{}, core.Fail("VALIDATION", "handoff requires open epic, owner and reason")
	}
	if t.Version != version {
		return core.Task{}, core.Fail("CONFLICT", "epic version differs")
	}
	var count int
	if err := b.tx.QueryRow(`SELECT COUNT(*) FROM tokens WHERE owner=?`, owner).Scan(&count); err != nil {
		return core.Task{}, err
	}
	if count == 0 {
		return core.Task{}, core.Fail("VALIDATION", "handoff owner does not exist")
	}
	for _, child := range b.Tasks {
		if child.EpicKey == t.Key && child.Lease != nil && child.Lease.ExpiresAt.After(now) && sessionLive(b, b.Sessions[child.Lease.SessionID], now) {
			return core.Task{}, core.Fail("LEASE_HELD", "epic contains active task "+child.Key)
		}
	}
	t.Owner = owner
	t.Version++
	emit(b, now, "epic_handoff", t.Key, session)
	b.Events[len(b.Events)-1].Reason = reason
	return viewTask(b, t), nil
}

// HandoffEpic is the local administrator entrypoint, unavailable to bearer API.
func (s *Store) HandoffEpic(ctx context.Context, key, owner string, version int64, reason, actor string) (core.Task, error) {
	if !bounded(actor, 128) {
		return core.Task{}, core.Fail("VALIDATION", "local actor required")
	}
	// Return the entity through transaction so its complete outward graph is
	// detached while an error can still roll back this administrative mutation.
	return transaction(s, ctx, false, func(b *board, now time.Time) (core.Task, error) {
		t, err := taskFor(b, key)
		if err != nil {
			return core.Task{}, err
		}
		out, err := handoffEpic(b, t, owner, version, reason, "", now)
		if err == nil {
			b.Events[len(b.Events)-1].Reason = actor + ": " + reason
		}
		return out, err
	})
}
