package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	"agent-board/internal/core"
)

// validateMigrationProjection compares the retained reducer and projection in
// the uncommitted migration. At most one event is decoded for each bound; the
// reducer streams its input. Compatibility histories retain the oracle path.
func validateMigrationProjection(ctx context.Context, tx *sql.Tx) error {
	c, err := readProjection(ctx, tx)
	if err != nil {
		return err
	}
	var high int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cursor),0) FROM events`).Scan(&high); err != nil {
		return err
	}
	if c.Dirty || c.Cursor != high {
		return fmt.Errorf("statistics projection coverage mismatch")
	}
	if high == 0 || c.Compatibility {
		return nil
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM events ORDER BY cursor LIMIT 1`).Scan(&raw); err != nil {
		return err
	}
	var first core.Event
	if err = json.Unmarshal(raw, &first); err != nil {
		return err
	}
	from := first.At
	to := time.Unix(c.Seconds, int64(c.Nanos)).UTC().Add(time.Nanosecond)
	middle := from.Add(to.Sub(from) / 2)
	b := &board{Tasks: map[string]*core.Task{}, tx: tx}
	for _, kind := range []string{"role", "owner", "epic"} {
		for _, begin := range []time.Time{from, middle} {
			in := core.StatsInput{From: begin, To: to, GroupBy: kind}
			base := core.Stats{From: begin, To: to, GroupBy: kind, Groups: []core.StatsGroup{}}
			oracle, oracleErr := statsOracle(ctx, b, to, in, base)
			projected, projectedErr := projectedStats(ctx, b, to, in, base)
			if oracleErr != nil || projectedErr != nil {
				// An overflowing entire range is a valid history. Narrow-range prefixes
				// still remain exact. The old reducer only checked token overflow;
				// schema4 also checks count/return overflow required by053.
				var left, right *core.Error
				if value, ok := oracleErr.(*core.Error); ok {
					left = value
				}
				if value, ok := projectedErr.(*core.Error); ok {
					right = value
				}
				if right != nil && right.Code == "LIMIT_EXCEEDED" && (left == nil && oracleErr == nil || left != nil && left.Code == right.Code) {
					continue
				}
				return fmt.Errorf("statistics migration differential error: oracle=%v projection=%v", oracleErr, projectedErr)
			}
			if !equivalentStats(oracle, projected) {
				return fmt.Errorf("statistics migration differential mismatch for %s", kind)
			}
		}
	}
	return nil
}

// Integer fields compare exactly. Existing seconds DTOs sum binary floats;
// exact nanosecond prefixes can differ only in accumulation rounding. Allow
// 10ns absolute plus 1e-12 relative error, never timestamp rounding.
func equivalentStats(a, b core.Stats) bool {
	close := func(x, y float64) bool { return math.Abs(x-y) <= 1e-8+math.Abs(y)*1e-12 }
	if !close(a.Totals.WallDurationSeconds, b.Totals.WallDurationSeconds) || !close(a.Totals.BlockedDurationSeconds, b.Totals.BlockedDurationSeconds) {
		return false
	}
	a.Totals.WallDurationSeconds = b.Totals.WallDurationSeconds
	a.Totals.BlockedDurationSeconds = b.Totals.BlockedDurationSeconds
	if len(a.Groups) != len(b.Groups) {
		return false
	}
	for i := range a.Groups {
		if !close(a.Groups[i].Metrics.WallDurationSeconds, b.Groups[i].Metrics.WallDurationSeconds) || !close(a.Groups[i].Metrics.BlockedDurationSeconds, b.Groups[i].Metrics.BlockedDurationSeconds) {
			return false
		}
		a.Groups[i].Metrics.WallDurationSeconds = b.Groups[i].Metrics.WallDurationSeconds
		a.Groups[i].Metrics.BlockedDurationSeconds = b.Groups[i].Metrics.BlockedDurationSeconds
	}
	return reflect.DeepEqual(a, b)
}
