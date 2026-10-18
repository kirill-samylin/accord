package store

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"agent-board/internal/core"
)

func (s *Store) Stats(ctx context.Context, a core.Actor, in core.StatsInput) (core.Stats, error) {
	result := core.Stats{From: in.From.UTC(), To: in.To.UTC(), GroupBy: in.GroupBy, Groups: []core.StatsGroup{}}
	if in.From.IsZero() || in.To.IsZero() || !in.From.Before(in.To) || in.GroupBy != "role" && in.GroupBy != "owner" && in.GroupBy != "epic" {
		return result, core.Fail("invalid_input", "from < to and group_by role, owner or epic required")
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.Stats, error) {
		if err := readActor(ctx, b, a, now); err != nil {
			return result, err
		}
		if err := persistNewEvents(ctx, b); err != nil {
			return result, err
		}
		rows, err := b.tx.QueryContext(ctx, `SELECT data FROM events ORDER BY cursor`)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		groups := map[string]*core.StatsMetrics{}
		blockedStarts := map[string]core.Event{}
		for rows.Next() {
			var raw []byte
			var e core.Event
			if err = rows.Scan(&raw); err != nil {
				return result, err
			}
			if err = json.Unmarshal(raw, &e); err != nil {
				return result, err
			}
			if !e.At.Before(in.To) {
				continue
			}
			terminal := e.Data != nil && e.Data.Kind == "task" && terminalState(e.Data.NewState)
			if e.Type == "task_created" && e.Data != nil && e.Data.Blocked && !terminal {
				blockedStarts[e.TaskKey] = e
			}
			if terminal || e.Type == "blocked_changed" && e.Data != nil {
				if !terminal && e.Data.Blocked {
					blockedStarts[e.TaskKey] = e
				} else if start, ok := blockedStarts[e.TaskKey]; ok {
					end := e.At
					begin := start.At
					if begin.Before(in.From) {
						begin = in.From
					}
					if end.After(begin) {
						seconds := end.Sub(begin).Seconds()
						result.Totals.BlockedDurationSeconds += seconds
						metricGroup(groups, statsKey(start.Data, in.GroupBy)).BlockedDurationSeconds += seconds
					}
					delete(blockedStarts, e.TaskKey)
				}
			}
			if e.At.Before(in.From) {
				continue
			}
			if e.Data == nil {
				if e.TaskKey != "" {
					result.IncompleteData = true
					result.IncompleteEvents++
				}
				continue
			}
			if e.Data.Incomplete {
				result.IncompleteData = true
				result.IncompleteEvents++
			}
			key := statsKey(e.Data, in.GroupBy)
			group := metricGroup(groups, key)
			for _, m := range []*core.StatsMetrics{&result.Totals, group} {
				if err = aggregateEvent(m, e); err != nil {
					return result, err
				}
			}
		}
		if err = rows.Err(); err != nil {
			return result, err
		}
		end := minTime(in.To, now)
		for _, start := range blockedStarts {
			begin := start.At
			if begin.Before(in.From) {
				begin = in.From
			}
			if end.After(begin) {
				seconds := end.Sub(begin).Seconds()
				result.Totals.BlockedDurationSeconds += seconds
				metricGroup(groups, statsKey(start.Data, in.GroupBy)).BlockedDurationSeconds += seconds
			}
		}
		result.Totals.CurrentQueue = queueSummary(b, "")
		for _, t := range b.Tasks {
			if t.Type != "task" {
				continue
			}
			d := &core.EventData{Role: t.Role, Owner: leaseOwner(b, t), EpicKey: t.EpicKey}
			q := &metricGroup(groups, statsKey(d, in.GroupBy)).CurrentQueue
			if t.State == "in_progress" {
				q.InProgress++
			} else if t.State == "created" {
				q.Waiting++
			} else if t.State == "ready" {
				epic := b.Tasks[t.EpicKey]
				if blocked(b, t) || epic == nil || epic.State != "open" {
					q.Waiting++
				} else {
					q.Ready++
				}
			}
		}
		missingShare(&result.Totals)
		if result.Totals.MissingTokens > 0 {
			result.IncompleteData = true
		}
		keys := make([]string, 0, len(groups))
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			missingShare(groups[k])
			result.Groups = append(result.Groups, core.StatsGroup{Key: k, Metrics: *groups[k]})
		}
		return result, nil
	})
}
func statsKey(d *core.EventData, group string) string {
	switch group {
	case "role":
		return d.Role
	case "owner":
		return d.Owner
	default:
		return d.EpicKey
	}
}
func metricGroup(groups map[string]*core.StatsMetrics, key string) *core.StatsMetrics {
	if groups[key] == nil {
		groups[key] = &core.StatsMetrics{}
	}
	return groups[key]
}
func missingShare(m *core.StatsMetrics) {
	if m.Attempts > 0 {
		m.MissingTokensShare = float64(m.MissingTokens) / float64(m.Attempts)
	}
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func aggregateEvent(m *core.StatsMetrics, e core.Event) error {
	d := e.Data
	if e.Type == "task_done" {
		if d.FirstDone {
			m.FirstCompletions++
			if d.FirstClaimAt != nil && d.FirstDoneAt != nil {
				m.WallDurationSeconds += d.FirstDoneAt.Sub(*d.FirstClaimAt).Seconds()
			}
		} else if d.Rework {
			m.ReworkCompletions++
		}
	}
	if e.Type == "epic_done" {
		m.EpicCompletions++
	}
	if e.Type == "task_ready" {
		m.Returns += d.Returns
	}
	if e.Type == "lease_expired" {
		m.ExpiredLeases++
	}
	if strings.HasSuffix(e.Type, "_warning") {
		m.RuleWarnings++
	}
	if d.AttemptEnded {
		m.Attempts++
		if d.TokensSpent == nil {
			m.MissingTokens++
		} else {
			if *d.TokensSpent > math.MaxInt64-m.TokensSpent {
				return core.Fail("limit_exceeded", "statistics token total exceeds int64")
			}
			m.TokensSpent += *d.TokensSpent
		}
	}
	return nil
}
