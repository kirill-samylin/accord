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
		if err := readActor(ctx, b, a, now); err != nil {
			return core.EventsPage{}, err
		}
		sess, _ := sessionFor(b, a, now)
		if err := toolAllowed(b, sess.Role, "events"); err != nil {
			return core.EventsPage{}, err
		}
		page := core.EventsPage{ConfigVersion: b.Config.ConfigVersion, Events: []core.Event{}, NextCursor: in.SinceCursor, HighWater: b.EventBase + int64(len(b.Events))}
		// Materialized expiry events are included in the same snapshot as reads.
		if err := persistNewEvents(ctx, b); err != nil {
			return page, err
		}
		return indexedEvents(ctx, b, in, page)

	})
}
func persistNewEvents(ctx context.Context, b *board) error {
	if len(b.Events) == 0 {
		return nil
	}
	var schema int
	if err := b.tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&schema); err != nil {
		return err
	}
	var control projectionControl
	var err error
	if schema >= 4 {
		control, err = ensureProjection(ctx, b.tx)
		if err != nil {
			return err
		}
	}
	stmt, err := b.tx.PrepareContext(ctx, `INSERT INTO events(cursor,data) VALUES(?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range b.Events {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err = stmt.ExecContext(ctx, e.Cursor, string(raw)); err != nil {
			return err
		}
		if schema >= 4 {
			if err = projectEvent(ctx, b.tx, &control, e); err != nil {
				return err
			}
		}
	}
	if schema >= 4 {
		if err = publishProjection(ctx, b.tx, control); err != nil {
			return err
		}
	}
	b.EventBase += int64(len(b.Events))
	b.Events = nil
	return nil
}
