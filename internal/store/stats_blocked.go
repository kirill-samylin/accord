package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"time"

	"agent-board/internal/core"
)

type blockedIntegral struct {
	Area   big.Int
	Active big.Int
	At     time.Time
}

func blockedBoundary(ctx context.Context, tx *sql.Tx, key statsGroupKey, at time.Time, strict bool) (blockedIntegral, error) {
	var out blockedIntegral
	var sec int64
	var ns int
	var area, active string
	comparator := "<="
	if strict {
		comparator = "<"
	}
	err := tx.QueryRowContext(ctx, `SELECT sec,ns,area,active FROM blocked_prefix WHERE kind=? AND group_key=? AND (sec,ns)`+comparator+`(?,?) ORDER BY sec DESC,ns DESC LIMIT 1`, key.Kind, key.Key, at.Unix(), at.Nanosecond()).Scan(&sec, &ns, &area, &active)
	if err == sql.ErrNoRows {
		out.At = at
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if _, ok := out.Area.SetString(area, 10); !ok {
		return out, core.Fail("CONFIG_INTERNAL", "invalid blocked area")
	}
	if _, ok := out.Active.SetString(active, 10); !ok {
		return out, core.Fail("CONFIG_INTERNAL", "invalid blocked count")
	}
	out.At = time.Unix(sec, int64(ns)).UTC()
	out.Area.Add(&out.Area, new(big.Int).Mul(&out.Active, elapsedNS(out.At, at)))
	out.At = at
	return out, nil
}
func changeBlocked(ctx context.Context, tx *sql.Tx, d *core.EventData, at time.Time, delta int64) error {
	for _, key := range eventGroups(d) {
		state, err := blockedBoundary(ctx, tx, key, at, false)
		if err != nil {
			return err
		}
		state.Active.Add(&state.Active, big.NewInt(delta))
		if state.Active.Sign() < 0 {
			return core.Fail("CONFIG_INTERNAL", "negative blocked count")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO blocked_prefix VALUES(?,?,?,?,?,?) ON CONFLICT(kind,group_key,sec,ns) DO UPDATE SET area=excluded.area,active=excluded.active`, key.Kind, key.Key, at.Unix(), at.Nanosecond(), state.Area.String(), state.Active.String()); err != nil {
			return err
		}
	}
	return nil
}
func projectBlocked(ctx context.Context, tx *sql.Tx, c *projectionControl, e core.Event) error {
	d := e.Data
	terminal := d != nil && d.Kind == "task" && terminalState(d.NewState)
	start := d != nil && !terminal && d.Blocked && (e.Type == "task_created" || e.Type == "blocked_changed")
	stop := terminal || e.Type == "blocked_changed" && d != nil && !d.Blocked
	if !start && !stop {
		return nil
	}
	var raw []byte
	var previous core.Event
	err := tx.QueryRowContext(ctx, `SELECT data FROM blocked_open WHERE task_key=?`, e.TaskKey).Scan(&raw)
	exists := err == nil
	if exists {
		if err = json.Unmarshal(raw, &previous); err != nil {
			return err
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if start && exists {
		// The original reducer replaces a repeated start without crediting its old
		// interval. It is deliberately served through the compatibility reducer.
		c.Compatibility = true
		return nil
	}
	if start {
		raw, err = json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO blocked_open VALUES(?,?,?,?)`, e.TaskKey, string(raw), e.At.Unix(), e.At.Nanosecond()); err != nil {
			return err
		}
		return changeBlocked(ctx, tx, d, e.At, 1)
	}
	if exists {
		elapsed := elapsedNS(previous.At, e.At)
		if !elapsed.IsInt64() || elapsed.Sign() < 0 {
			c.Compatibility = true
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM blocked_open WHERE task_key=?`, e.TaskKey); err != nil {
			return err
		}
		if e.At.Before(previous.At) {
			return nil
		}
		return changeBlocked(ctx, tx, previous.Data, e.At, -1)
	}
	return nil
}
