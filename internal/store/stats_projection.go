package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agent-board/internal/core"
)

const reducerVersion = 1

type projectionControl struct {
	Cursor               int64
	Dirty, Compatibility bool
	Seconds              int64
	Nanos                int
}

func readProjection(ctx context.Context, tx *sql.Tx) (projectionControl, error) {
	var c projectionControl
	var version int
	err := tx.QueryRowContext(ctx, `SELECT reducer_version,built_cursor,dirty,compatibility,last_sec,last_ns FROM projection_control WHERE id=1`).Scan(&version, &c.Cursor, &c.Dirty, &c.Compatibility, &c.Seconds, &c.Nanos)
	if err == nil && version != reducerVersion {
		return c, fmt.Errorf("unsupported statistics reducer version")
	}
	return c, err
}
func ensureProjection(ctx context.Context, tx *sql.Tx) (projectionControl, error) {
	c, err := readProjection(ctx, tx)
	if err != nil {
		return c, err
	}
	var high int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cursor),0) FROM events`).Scan(&high); err != nil {
		return c, err
	}
	if !c.Dirty && c.Cursor == high {
		return c, nil
	}
	if err = rebuildProjection(ctx, tx); err != nil {
		return c, err
	}
	return readProjection(ctx, tx)
}

// rebuildProjection streams at most 256 raw events at once. The caller owns the
// same write transaction as DDL/trigger invalidation and atomic publication.
func rebuildProjection(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"stats_prefix", "blocked_prefix", "blocked_open", "stats_groups"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	c := projectionControl{}
	for {
		rows, err := tx.QueryContext(ctx, `SELECT cursor,data FROM events WHERE cursor>? ORDER BY cursor LIMIT 256`, c.Cursor)
		if err != nil {
			return err
		}
		batch := make([]core.Event, 0, 256)
		for rows.Next() {
			var e core.Event
			var raw []byte
			if err = rows.Scan(&e.Cursor, &raw); err == nil {
				cursor := e.Cursor
				err = json.Unmarshal(raw, &e)
				e.Cursor = cursor
			}
			if err != nil {
				break
			}
			batch = append(batch, e)
		}
		rowErr := rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if rowErr != nil {
			return rowErr
		}
		if len(batch) == 0 {
			break
		}
		for _, e := range batch {
			if err = projectEvent(ctx, tx, &c, e); err != nil {
				return err
			}
		}
	}
	return publishProjection(ctx, tx, c)
}
func publishProjection(ctx context.Context, tx *sql.Tx, c projectionControl) error {
	_, err := tx.ExecContext(ctx, `UPDATE projection_control SET reducer_version=?,built_cursor=?,dirty=0,compatibility=?,last_sec=?,last_ns=? WHERE id=1`, reducerVersion, c.Cursor, c.Compatibility, c.Seconds, c.Nanos)
	return err
}
func projectEvent(ctx context.Context, tx *sql.Tx, c *projectionControl, e core.Event) error {
	if c.Cursor != 0 && e.At.Before(time.Unix(c.Seconds, int64(c.Nanos))) {
		c.Compatibility = true
	}
	// Negative raw tokens are not supported by the nonnegative writer contract.
	if e.Data != nil && e.Data.TokensSpent != nil && *e.Data.TokensSpent < 0 {
		c.Compatibility = true
	}
	if c.Compatibility {
		c.Cursor, c.Seconds, c.Nanos = e.Cursor, e.At.Unix(), e.At.Nanosecond()
		return nil
	}
	delta := eventVector(e)
	for _, key := range eventGroups(e.Data) {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO stats_groups VALUES(?,?)`, key.Kind, key.Key); err != nil {
			return err
		}
		var cumulative exactVector
		var raw []byte
		err := tx.QueryRowContext(ctx, `SELECT data FROM stats_prefix WHERE kind=? AND group_key=? ORDER BY sec DESC,ns DESC LIMIT 1`, key.Kind, key.Key).Scan(&raw)
		if err == nil {
			err = json.Unmarshal(raw, &cumulative)
		}
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		for i := range cumulative {
			cumulative[i].Add(&cumulative[i], &delta[i])
		}
		encoded, err := json.Marshal(cumulative)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO stats_prefix VALUES(?,?,?,?,?) ON CONFLICT(kind,group_key,sec,ns) DO UPDATE SET data=excluded.data`, key.Kind, key.Key, e.At.Unix(), e.At.Nanosecond(), string(encoded))
		if err != nil {
			return err
		}
	}
	if err := projectBlocked(ctx, tx, c, e); err != nil {
		return err
	}
	c.Cursor, c.Seconds, c.Nanos = e.Cursor, e.At.Unix(), e.At.Nanosecond()
	return nil
}
