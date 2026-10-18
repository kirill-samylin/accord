package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"time"

	"agent-board/internal/core"
)

func prefixAt(ctx context.Context, tx *sql.Tx, key statsGroupKey, at time.Time) (exactVector, error) {
	var value exactVector
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM stats_prefix WHERE kind=? AND group_key=? AND (sec,ns)<(?,?) ORDER BY sec DESC,ns DESC LIMIT 1`, key.Kind, key.Key, at.Unix(), at.Nanosecond()).Scan(&raw)
	if err == sql.ErrNoRows {
		return value, nil
	}
	if err == nil {
		err = json.Unmarshal(raw, &value)
	}
	return value, err
}
func prefixRange(ctx context.Context, tx *sql.Tx, key statsGroupKey, in core.StatsInput) (exactVector, error) {
	upper, err := prefixAt(ctx, tx, key, in.To)
	if err != nil {
		return upper, err
	}
	lower, err := prefixAt(ctx, tx, key, in.From)
	if err == nil {
		upper.subtract(&lower)
	}
	return upper, err
}
func blockedRange(ctx context.Context, tx *sql.Tx, key statsGroupKey, from, to time.Time) (*big.Int, error) {
	if !to.After(from) {
		return new(big.Int), nil
	}
	upper, err := blockedBoundary(ctx, tx, key, to, true)
	if err != nil {
		return nil, err
	}
	lower, err := blockedBoundary(ctx, tx, key, from, true)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Sub(&upper.Area, &lower.Area), nil
}
func projectedStats(ctx context.Context, b *board, now time.Time, in core.StatsInput, result core.Stats) (core.Stats, error) {
	control, err := ensureProjection(ctx, b.tx)
	if err != nil {
		return result, err
	}
	if control.Compatibility || control.Cursor != 0 && now.Before(time.Unix(control.Seconds, int64(control.Nanos))) {
		return statsOracle(ctx, b, now, in, result)
	}
	// An open legacy span longer than time.Duration must preserve Time.Sub's
	// saturating behavior through the compatibility reducer.
	var sec int64
	var ns int
	err = b.tx.QueryRowContext(ctx, `SELECT sec,ns FROM blocked_open ORDER BY sec,ns LIMIT 1`).Scan(&sec, &ns)
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	end := minTime(in.To, now)
	if err == nil && !elapsedNS(time.Unix(sec, int64(ns)), end).IsInt64() {
		return statsOracle(ctx, b, now, in, result)
	}
	total, err := prefixRange(ctx, b.tx, statsGroupKey{"total", ""}, in)
	if err != nil {
		return result, err
	}
	result.Totals, err = total.metrics()
	if err != nil {
		return result, err
	}
	if !total[xIncomplete].IsInt64() {
		return result, core.Fail("limit_exceeded", "incomplete event count exceeds int64")
	}
	result.IncompleteEvents = total[xIncomplete].Int64()
	result.IncompleteData = result.IncompleteEvents > 0
	area, err := blockedRange(ctx, b.tx, statsGroupKey{"total", ""}, in.From, end)
	if err != nil {
		return result, err
	}
	result.Totals.BlockedDurationSeconds = exactSeconds(area)
	rows, err := b.tx.QueryContext(ctx, `SELECT group_key FROM stats_groups WHERE kind=? ORDER BY group_key`, in.GroupBy)
	if err != nil {
		return result, err
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			break
		}
		keys = append(keys, key)
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if rowErr != nil {
		return result, rowErr
	}
	groups := map[string]*core.StatsMetrics{}
	for _, key := range keys {
		group := statsGroupKey{in.GroupBy, key}
		delta, err := prefixRange(ctx, b.tx, group, in)
		if err != nil {
			return result, err
		}
		area, err := blockedRange(ctx, b.tx, group, in.From, end)
		if err != nil {
			return result, err
		}
		if delta[xPresence].Sign() == 0 && area.Sign() == 0 {
			continue
		}
		m, err := delta.metrics()
		if err != nil {
			return result, err
		}
		m.BlockedDurationSeconds = exactSeconds(area)
		groups[key] = &m
	}
	return finishStats(b, now, in, result, groups), nil
}
