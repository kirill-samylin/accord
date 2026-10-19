package store

import (
	"context"
	"time"

	"agent-board/internal/core"
)

// ControlOverview reads counts and effective credentials in the same transaction.
func (s *Store) ControlOverview(ctx context.Context, a core.Actor) (core.ControlOverview, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ControlOverview, error) {
		p := core.ControlOverview{ObservedAt: now, Project: b.Config.Project}
		if err := readActor(ctx, b, a, now); err != nil {
			return p, err
		}
		credentials, err := controlCredentials(ctx, b, now)
		if err != nil {
			return p, err
		}
		for _, sess := range b.Sessions {
			if controlStatus(sess, credentials, now) != "active" {
				continue
			}
			if sess.Role == "human" {
				p.ActiveHumanSessions++
			} else {
				p.ActiveAgentSessions++
			}
		}
		for _, t := range b.Tasks {
			if t.Type == "epic" {
				if t.State == "open" {
					p.OpenEpics++
				}
				continue
			}
			p.TasksTotal++
			switch t.State {
			case "created":
				p.TasksCreated++
			case "ready":
				p.TasksReady++
			case "in_progress":
				p.TasksInProgress++
			case "done":
				p.TasksDone++
			case "cancelled":
				p.TasksCancelled++
			}
			if t.State != "done" && t.State != "cancelled" && blocked(b, t) {
				p.TasksBlocked++
			}
			if controlLease(b, t, credentials, now) {
				p.LeasedTasks++
			}
		}
		p.LatestEventCursor = b.EventBase + int64(len(b.Events))
		return p, nil
	})
}
