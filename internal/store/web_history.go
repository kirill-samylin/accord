package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"time"
)

// ListDecisions reads immutable committed events, never today's mutable answer.
// This additive reader leaves the native ascending Events cursor contract intact.
func (s *Store) ListDecisions(ctx context.Context, a core.Actor, in core.DecisionsInput) (core.DecisionsPage, error) {
	if in.Limit == 0 {
		in.Limit = 50
	}
	if in.Limit < 1 || in.Limit > 100 || in.BeforeCursor < 0 {
		return core.DecisionsPage{}, core.Fail("VALIDATION", "invalid decision cursor or limit")
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.DecisionsPage, error) {
		page := core.DecisionsPage{Events: []core.Event{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		rows, err := b.tx.QueryContext(ctx, `SELECT data FROM events WHERE json_extract(data,'$.type')='human_answered' AND (?=0 OR cursor<?) AND (?='' OR json_extract(data,'$.data.epic_key')=?) ORDER BY cursor DESC LIMIT ?`, in.BeforeCursor, in.BeforeCursor, in.EpicKey, in.EpicKey, in.Limit+1)
		if err != nil {
			return page, err
		}
		defer rows.Close()
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return page, err
			}
			if len(page.Events) == in.Limit {
				page.HasMore = true
				break
			}
			var e core.Event
			if err = json.Unmarshal(raw, &e); err != nil {
				return page, err
			}
			page.Events = append(page.Events, e)
			page.NextCursor = e.Cursor
		}
		if !page.HasMore {
			page.NextCursor = 0
		}
		return page, rows.Err()
	})
}
