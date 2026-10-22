package store

import (
	"agent-board/internal/core"
	"fmt"
	"sort"
	"time"
)

// A positive local execution-exit report ends only the board sessions that this
// exact managed attempt opened. Ordinary bridge EOF and unknown process state
// grant no release authority. Task completion still requires native evidence.
func managedCloseSessions(b *board, r *core.ManagedRun, h *managedHostRecord, phase string, now time.Time) error {
	sessions := []string{}
	for id, sess := range b.Sessions {
		if sess.ManagedRunID == r.ID && sess.ManagedGeneration == r.Generation && sess.RunID == r.ID && sess.PrincipalID == h.Host.OwnerPrincipalID {
			sessions = append(sessions, id)
		}
	}
	sort.Strings(sessions)
	keys := []string{}
	for key := range b.Tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	reason := fmt.Sprintf("managed_%s:%s@generation=%d", phase, r.ID, r.Generation)
	for _, id := range sessions {
		sess := b.Sessions[id]
		for _, key := range keys {
			task := b.Tasks[key]
			lease := task.Lease
			if lease == nil || lease.SessionID != id {
				continue
			}
			if lease.Generation != task.Generation || task.State != "in_progress" || len(task.Attempts) == 0 {
				return core.Fail("conflict", "Managed lease не совпадает с текущей попыткой задачи")
			}
			attempt := task.Attempts[len(task.Attempts)-1]
			if attempt.SessionID != id || attempt.Generation != lease.Generation || attempt.EndedAt != nil {
				return core.Fail("conflict", "Managed attempt provenance не совпадает с lease")
			}
			finishAttempt(task, now, reason, "")
			task.Lease = nil
			task.State = "ready"
			task.ReadyAt = now
			task.Version++
			emit(b, now, "released", task.Key, id)
			b.Events[len(b.Events)-1].Reason = reason
		}
		if sess.ExpiresAt.After(now) {
			sess.ExpiresAt = now
			emit(b, now, "managed_session_closed", "", id)
			b.Events[len(b.Events)-1].Reason = reason
		}
	}
	return nil
}
