package store

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"testing"
	"time"

	"agent-board/internal/core"
)

func seedPerformanceEvents(t *testing.T, f *fixture, events []core.Event) {
	t.Helper()
	tx, err := f.s.db.BeginTx(testContext, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM events`); err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO events(cursor,data) VALUES(?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	for i, e := range events {
		e.Cursor = int64(i + 1)
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = stmt.Exec(e.Cursor, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
func comparePerformanceStats(t *testing.T, f *fixture, inputs []core.StatsInput, compatibility bool) {
	t.Helper()
	_, err := transact(f.s, testContext, func(b *board, now time.Time) (struct{}, error) {
		c, err := ensureProjection(testContext, b.tx)
		if err != nil {
			return struct{}{}, err
		}
		if c.Compatibility != compatibility {
			t.Fatalf("compatibility=%v expected=%v", c.Compatibility, compatibility)
		}
		for _, in := range inputs {
			base := core.Stats{From: in.From, To: in.To, GroupBy: in.GroupBy, Groups: []core.StatsGroup{}}
			fast, fastErr := projectedStats(testContext, b, now, in, base)
			oracle, oracleErr := statsOracle(testContext, b, now, in, base)
			if (fastErr == nil) != (oracleErr == nil) {
				t.Fatalf("errors fast=%v oracle=%v", fastErr, oracleErr)
			}
			if fastErr != nil {
				continue
			}
			if !equivalentStats(fast, oracle) {
				t.Fatalf("%+v\nfast=%+v\noracle=%+v", in, fast, oracle)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestPerformanceProjectionDifferentialRandomized(t *testing.T) {
	f := newFixture(t)
	start := f.clock.read().Add(-time.Hour)
	rng := rand.New(rand.NewSource(57))
	events := []core.Event{}
	blocked := map[string]bool{}
	for i := 0; i < 600; i++ {
		key := string(rune('a' + rng.Intn(20)))
		at := start.Add(time.Duration(i/3)*time.Second + time.Duration(i/3%7)) // exact ties and nanoseconds
		d := &core.EventData{Kind: "task", Role: []string{"developer", "reviewer", ""}[i%3], Owner: []string{"alice", "bob", ""}[i%3], EpicKey: "E-synthetic", NewState: "ready"}
		kind := []string{"commented", "task_done", "task_ready", "lease_expired", "area_warning"}[rng.Intn(5)]
		switch i % 9 {
		case 0, 1:
			kind = "blocked_changed"
			blocked[key] = !blocked[key]
			d.Blocked = blocked[key]
		case 2:
			kind = "task_done"
			d.NewState = "done"
			d.FirstDone = true
			claim, done := at.Add(time.Duration(rng.Intn(20)-10)*time.Second), at
			d.FirstClaimAt, d.FirstDoneAt = &claim, &done // negative wall durations included
			d.AttemptEnded = true
			tokens := int64(rng.Intn(100))
			if i%4 != 0 {
				d.TokensSpent = &tokens
			}
			blocked[key] = false
		default:
			d.Incomplete = i%17 == 0
		}
		events = append(events, core.Event{Type: kind, TaskKey: key, At: at, Data: d})
	}
	seedPerformanceEvents(t, f, events)
	inputs := []core.StatsInput{}
	for _, group := range []string{"role", "owner", "epic"} {
		for i := 0; i < 50; i++ {
			from := start.Add(time.Duration(rng.Intn(220)) * time.Second)
			inputs = append(inputs, core.StatsInput{From: from, To: from.Add(time.Duration(1+rng.Intn(60)) * time.Second), GroupBy: group})
		}
	}
	comparePerformanceStats(t, f, inputs, false)
}
func TestPerformanceProjectionExactPrefixAndCompatibility(t *testing.T) {
	f := newFixture(t)
	start := f.clock.read().Add(-time.Hour)
	huge := int64(math.MaxInt64)
	zero := int64(0)
	events := []core.Event{
		{Type: "released", TaskKey: "x", At: start, Data: &core.EventData{Role: "developer", AttemptEnded: true, TokensSpent: &huge}},
		{Type: "released", TaskKey: "x", At: start.Add(time.Second), Data: &core.EventData{Role: "developer", AttemptEnded: true, TokensSpent: &huge}},
		{Type: "released", TaskKey: "x", At: start.Add(2 * time.Second), Data: &core.EventData{Role: "developer", AttemptEnded: true, TokensSpent: &zero}},
		{Type: "commented", At: start.Add(3 * time.Second), Data: &core.EventData{Owner: "zero"}},
		{Type: "legacy", TaskKey: "old", At: start.Add(4 * time.Second)},
	}
	seedPerformanceEvents(t, f, events)
	for _, group := range []string{"role", "owner", "epic"} {
		comparePerformanceStats(t, f, []core.StatsInput{{From: start.Add(time.Second), To: start.Add(5 * time.Second), GroupBy: group}, {From: start.Add(2 * time.Second), To: start.Add(5 * time.Second), GroupBy: group}, {From: start, To: start.Add(5 * time.Second), GroupBy: group}}, false)
	}
	events = append(events, core.Event{Type: "commented", At: start.Add(-time.Second)})
	seedPerformanceEvents(t, f, events)
	comparePerformanceStats(t, f, []core.StatsInput{{From: start.Add(-time.Minute), To: start.Add(time.Minute), GroupBy: "owner"}}, true)
}
func TestPerformanceProjectionLegacyZeroAndExternalRewrite(t *testing.T) {
	f := newFixture(t)
	at := f.clock.read().Add(-time.Minute)
	events := []core.Event{{Type: "legacy", TaskKey: "x", At: time.Time{}}, {Type: "commented", At: at, Data: &core.EventData{Role: "developer"}}}
	seedPerformanceEvents(t, f, events)
	in := core.StatsInput{From: time.Date(1, 1, 1, 0, 0, 0, 1, time.UTC), To: at.Add(time.Second), GroupBy: "role"}
	comparePerformanceStats(t, f, []core.StatsInput{in}, false)
	if _, err := f.s.db.Exec(`UPDATE events SET data=json_set(data,'$.data.incomplete',json('true')) WHERE cursor=2`); err != nil {
		t.Fatal(err)
	}
	comparePerformanceStats(t, f, []core.StatsInput{in}, false)
	ctx, cancel := context.WithCancel(testContext)
	cancel()
	if _, err := f.s.Stats(ctx, f.coord, in); err == nil {
		t.Fatal("canceled stats succeeded")
	}
}

func TestPerformanceProjectionCountOverflowNarrowWindow(t *testing.T) {
	f := newFixture(t)
	start := f.clock.read().Add(-time.Hour)
	events := []core.Event{
		{Type: "task_ready", At: start, Data: &core.EventData{Returns: math.MaxInt64, Role: "developer"}},
		{Type: "task_ready", At: start.Add(time.Second), Data: &core.EventData{Returns: math.MaxInt64, Role: "developer"}},
	}
	seedPerformanceEvents(t, f, events)
	in := core.StatsInput{From: start.Add(time.Second), To: start.Add(2 * time.Second), GroupBy: "role"}
	result, err := f.s.Stats(testContext, f.coord, in)
	if err != nil || result.Totals.Returns != math.MaxInt64 {
		t.Fatal(result, err)
	}
	in.From = start
	_, err = f.s.Stats(testContext, f.coord, in)
	requireCode(t, err, "LIMIT_EXCEEDED")
}
