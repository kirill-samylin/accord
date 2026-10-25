package store

import (
	"context"
	"encoding/json"
	"time"

	"agent-board/internal/core"
)

// RecentEvents applies immutable DESC keyset and filters in SQL before LIMIT.
// Subsequent pages use cursor<before, so new events cannot shift older pages.
func (s *Store) RecentEvents(ctx context.Context, a core.Actor, in core.RecentEventsInput) (core.RecentEventsPage, error) {
	if in.Limit == 0 {
		in.Limit = 20
	}
	if in.BeforeCursor < 0 || in.Limit < 1 || in.Limit > 100 || in.TaskKey != "" && !core.ValidControlTaskKey(in.TaskKey) || in.SessionID != "" && !core.ValidControlSessionID(in.SessionID) || in.Type != "" && !core.ValidControlEventType(in.Type) {
		return core.RecentEventsPage{}, core.Fail("validation", "Неверный фильтр истории")
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.RecentEventsPage, error) {
		p := core.RecentEventsPage{Events: []core.Event{}, ObservedAt: now}
		if err := readActor(ctx, b, a, now); err != nil {
			return p, err
		}
		if err := persistNewEvents(ctx, b); err != nil {
			return p, err
		}
		rows, err := b.tx.QueryContext(ctx, `SELECT data FROM events WHERE (?=0 OR cursor<?) AND (?='' OR json_extract(data,'$.task_key')=?) AND (?='' OR json_extract(data,'$.session_id')=?) AND (?='' OR json_extract(data,'$.type')=?) ORDER BY cursor DESC LIMIT ?`, in.BeforeCursor, in.BeforeCursor, in.TaskKey, in.TaskKey, in.SessionID, in.SessionID, in.Type, in.Type, in.Limit+1)
		if err != nil {
			return p, err
		}
		defer rows.Close()
		for rows.Next() {
			if len(p.Events) == in.Limit {
				p.HasMore = true
				break
			}
			var raw []byte
			var e core.Event
			if err := rows.Scan(&raw); err != nil {
				return p, err
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return p, err
			}
			// Token administration IDs are not needed in the browser projection.
			e.TokenID = ""
			// Web sessions are opaque hashes; name the person who opened it
			// unless the event already records its own owner.
			if sess := b.Sessions[e.SessionID]; sess != nil && sess.Owner != "" && (e.Data == nil || e.Data.Owner == "") {
				e.Actor = sess.Owner
			}
			p.Events = append(p.Events, e)
			p.NextCursor = e.Cursor
		}
		if !p.HasMore {
			p.NextCursor = 0
		}
		return p, rows.Err()
	})
}
