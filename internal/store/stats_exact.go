package store

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"agent-board/internal/core"
)

const (
	xFirst = iota
	xRework
	xEpic
	xReturns
	xTokens
	xAttempts
	xMissing
	xExpired
	xWarnings
	xWall
	xPresence
	xIncomplete
	exactSize
)

// exactVector stores decimal integers in JSON strings. SQLite never coerces
// historical totals to INTEGER/REAL; even a prefix exceeding int64 is valid.
type exactVector [exactSize]big.Int

func (v exactVector) MarshalJSON() ([]byte, error) {
	var out [exactSize]string
	for i := range v {
		out[i] = v[i].String()
	}
	return json.Marshal(out)
}
func (v *exactVector) UnmarshalJSON(raw []byte) error {
	var values [exactSize]string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	for i, value := range values {
		if _, ok := v[i].SetString(value, 10); !ok {
			return fmt.Errorf("invalid exact statistics prefix")
		}
	}
	return nil
}
func (v *exactVector) add(i int, n int64) { v[i].Add(&v[i], big.NewInt(n)) }
func (v *exactVector) subtract(other *exactVector) {
	for i := range v {
		v[i].Sub(&v[i], &other[i])
	}
}
func eventVector(e core.Event) exactVector {
	var v exactVector
	d := e.Data
	if d == nil {
		if e.TaskKey != "" {
			v.add(xIncomplete, 1)
		}
		return v
	}
	v.add(xPresence, 1)
	if d.Incomplete {
		v.add(xIncomplete, 1)
	}
	if e.Type == "task_done" {
		if d.FirstDone {
			v.add(xFirst, 1)
			if d.FirstClaimAt != nil && d.FirstDoneAt != nil {
				v.add(xWall, int64(d.FirstDoneAt.Sub(*d.FirstClaimAt)))
			}
		} else if d.Rework {
			v.add(xRework, 1)
		}
	}
	if e.Type == "epic_done" {
		v.add(xEpic, 1)
	}
	if e.Type == "task_ready" {
		v.add(xReturns, d.Returns)
	}
	if e.Type == "lease_expired" {
		v.add(xExpired, 1)
	}
	if strings.HasSuffix(e.Type, "_warning") {
		v.add(xWarnings, 1)
	}
	if d.AttemptEnded {
		v.add(xAttempts, 1)
		if d.TokensSpent == nil {
			v.add(xMissing, 1)
		} else {
			v.add(xTokens, *d.TokensSpent)
		}
	}
	return v
}
func (v *exactVector) metrics() (core.StatsMetrics, error) {
	var m core.StatsMetrics
	fields := []*int64{&m.FirstCompletions, &m.ReworkCompletions, &m.EpicCompletions, &m.Returns, &m.TokensSpent, &m.Attempts, &m.MissingTokens, &m.ExpiredLeases, &m.RuleWarnings}
	for i, field := range fields {
		if !v[i].IsInt64() {
			return m, core.Fail("limit_exceeded", "statistics total exceeds int64")
		}
		*field = v[i].Int64()
	}
	m.WallDurationSeconds = exactSeconds(&v[xWall])
	return m, nil
}
func exactSeconds(value *big.Int) float64 {
	seconds := new(big.Rat).SetFrac(value, big.NewInt(int64(time.Second)))
	out, _ := seconds.Float64()
	return out
}
func elapsedNS(from, to time.Time) *big.Int {
	seconds := new(big.Int).Sub(big.NewInt(to.Unix()), big.NewInt(from.Unix()))
	seconds.Mul(seconds, big.NewInt(int64(time.Second)))
	return seconds.Add(seconds, big.NewInt(int64(to.Nanosecond()-from.Nanosecond())))
}

type statsGroupKey struct{ Kind, Key string }

func eventGroups(d *core.EventData) []statsGroupKey {
	out := []statsGroupKey{{"total", ""}}
	if d != nil {
		out = append(out, statsGroupKey{"role", d.Role}, statsGroupKey{"owner", d.Owner}, statsGroupKey{"epic", d.EpicKey})
	}
	return out
}
