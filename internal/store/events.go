package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"time"
)

func (s *Store) Events(ctx context.Context, a core.Actor, in core.EventsInput) (core.EventsPage, error) {
	if in.Limit == 0 {
		in.Limit = 100
	}
	if in.Limit < 1 || in.Limit > 200 || in.SinceCursor < 0 {
		return core.EventsPage{}, core.Fail("invalid_input", "cursor>=0 and limit in [1,200] required")
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.EventsPage, error) {
		if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
			return core.EventsPage{}, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return core.EventsPage{}, err
		}
		page := core.EventsPage{Events: []core.Event{}, NextCursor: in.SinceCursor, HighWater: b.EventBase + int64(len(b.Events))}
		// Materialized expiry events are included in the same snapshot as reads.
		if err := persistNewEvents(ctx, b); err != nil {
			return page, err
		}
		rows, err := b.tx.QueryContext(ctx, `SELECT cursor,data FROM events WHERE cursor>? AND cursor<=? ORDER BY cursor`, in.SinceCursor, page.HighWater)
		if err != nil {
			return page, err
		}
		defer rows.Close()
		for rows.Next() {
			var cursor int64
			var raw []byte
			if err = rows.Scan(&cursor, &raw); err != nil {
				return page, err
			}
			var e core.Event
			if err = json.Unmarshal(raw, &e); err != nil {
				return page, err
			}
			matches := in.TaskKey == "" || in.TaskKey == e.TaskKey
			if len(in.Types) > 0 {
				found := false
				for _, kind := range in.Types {
					if e.Type == kind {
						found = true
					}
				}
				matches = matches && found
			}
			if matches && len(page.Events) == in.Limit {
				page.HasMore = true
				break
			}
			page.NextCursor = cursor
			if matches {
				page.Events = append(page.Events, e)
			}
		}
		return page, rows.Err()
	})
}
func persistNewEvents(ctx context.Context, b *board) error {
	for _, e := range b.Events {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err = b.tx.ExecContext(ctx, `INSERT INTO events(cursor,data) VALUES(?,?)`, e.Cursor, string(raw)); err != nil {
			return err
		}
	}
	b.EventBase += int64(len(b.Events))
	b.Events = nil
	return nil
}
