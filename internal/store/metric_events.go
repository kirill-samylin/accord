package store

import (
	"time"

	"agent-board/internal/core"
)

type taskFact struct {
	State     string
	Blocked   bool
	FirstDone bool
	Returns   int64
}

func metricFacts(b *board) map[string]taskFact {
	result := map[string]taskFact{}
	for key, t := range b.Tasks {
		result[key] = taskFact{State: t.State, Blocked: metricBlocked(b, t), FirstDone: t.FirstDoneAt != nil, Returns: t.ReturnsCount}
	}
	return result
}

// enrichEvents captures event-time facts before commit. Stats never joins old
// events to mutable current task metadata.
func enrichEvents(b *board, before map[string]taskFact, now time.Time) {
	for i := range b.Events {
		e := &b.Events[i]
		t := b.Tasks[e.TaskKey]
		if t == nil || e.Data != nil && e.Data.Kind != "" {
			continue
		}
		old := before[t.Key]
		owner := sessionOwner(b, e.SessionID)
		epicKey := t.EpicKey
		if t.Type == "epic" {
			epicKey = t.Key
		}
		d := &core.EventData{Role: t.Role, Owner: owner, EpicKey: epicKey, Kind: t.Type, OldState: old.State, NewState: t.State, FirstDone: !old.FirstDone && t.FirstDoneAt != nil && t.CreatedAt != nil, Rework: old.FirstDone, Returns: t.ReturnsCount - old.Returns, FirstClaimAt: t.FirstClaimAt, FirstDoneAt: t.FirstDoneAt, Blocked: metricBlocked(b, t), Incomplete: t.CreatedAt == nil}
		if e.Type == "task_done" || e.Type == "released" || e.Type == "lease_expired" || e.Type == "lease_recovered" || e.Type == "attempt_recovered" {
			if len(t.Attempts) > 0 {
				last := t.Attempts[len(t.Attempts)-1]
				if last.EndedAt != nil {
					d.Owner = sessionOwner(b, last.SessionID)
					d.TokensSpent = last.TokensSpent
					d.AttemptEnded = true
				}
			}
		}
		if previous := e.Data; previous != nil {
			d.AssignmentChange = previous.AssignmentChange
			d.CheckCreated = previous.CheckCreated
			d.NoteKey = previous.NoteKey
			d.ScopeType = previous.ScopeType
			d.ScopeID = previous.ScopeID
			d.TokensSpentDelta = previous.TokensSpentDelta
			d.TokensSpentTotal = previous.TokensSpentTotal
			d.CorrectionDelta = previous.CorrectionDelta
		}
		e.Data = d
	}
	// Dependency operations may change another task's queue state without a
	// state transition. Store this boundary explicitly for blocked wall duration.
	for key, t := range b.Tasks {
		old, exists := before[key]
		current := metricBlocked(b, t)
		if exists && old.Blocked != current {
			emit(b, now, "blocked_changed", key, "")
			e := &b.Events[len(b.Events)-1]
			e.Data = &core.EventData{Role: t.Role, Owner: leaseOwner(b, t), EpicKey: t.EpicKey, Kind: t.Type, OldState: old.State, NewState: t.State, Blocked: current, Incomplete: t.CreatedAt == nil}
		}
	}
}

// Terminal work has no operational waiting interval. Gate validity remains
// independent: a completed result may still fail dependency/review predicates.
func metricBlocked(b *board, t *core.Task) bool {
	return t.Type == "task" && !terminalState(t.State) && blocked(b, t)
}

func terminalState(state string) bool { return state == "done" || state == "cancelled" }

func sessionOwner(b *board, id string) string {
	if session := b.Sessions[id]; session != nil {
		return session.Owner
	}
	return ""
}

func leaseOwner(b *board, t *core.Task) string {
	if t.Lease != nil {
		return sessionOwner(b, t.Lease.SessionID)
	}
	return ""
}
