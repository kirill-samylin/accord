package store

import (
	"fmt"
	"strings"
	"testing"

	"agent-board/internal/core"
)

func TestPerformanceEventsIndexedGapsAndPlans(t *testing.T) {
	f := newFixture(t)
	at := f.clock.read()
	events := []core.Event{}
	for i := 0; i < 100; i++ {
		key := "T-other"
		kind := "commented"
		if i == 3 || i == 70 || i == 91 {
			key = "T-match"
			kind = "linked"
		}
		if i == 80 {
			key = "T-match"
			kind = "released"
		}
		events = append(events, core.Event{TaskKey: key, Type: kind, At: at})
	}
	seedPerformanceEvents(t, f, events)
	// An actual primary-key gap before a next match must not become its cursor-1.
	if _, err := f.s.db.Exec(`DELETE FROM events WHERE cursor>=20 AND cursor<=70`); err != nil {
		t.Fatal(err)
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{TaskKey: "T-match", Types: []string{"linked", "released", "linked"}, Limit: 1})
	if err != nil || !page.HasMore || len(page.Events) != 1 || page.Events[0].Cursor != 4 || page.NextCursor != 19 || page.HighWater != 100 {
		t.Fatalf("%+v %v", page, err)
	}
	page, err = f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: page.NextCursor, TaskKey: "T-match", Types: []string{"linked", "released"}, Limit: 2})
	if err != nil || !page.HasMore || len(page.Events) != 2 || page.Events[0].Cursor != 71 || page.Events[1].Cursor != 81 || page.NextCursor != 91 {
		t.Fatalf("%+v %v", page, err)
	}
	page, err = f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: page.NextCursor, TaskKey: "T-match", Types: []string{"linked"}, Limit: 2})
	if err != nil || page.HasMore || len(page.Events) != 1 || page.NextCursor != 100 {
		t.Fatalf("%+v %v", page, err)
	}
	for _, filter := range []string{"", ` AND json_extract(data,'$.task_key')='T-match'`, ` AND json_extract(data,'$.type')='linked'`, ` AND json_extract(data,'$.task_key')='T-match' AND json_extract(data,'$.type')='linked'`} {
		task := strings.Contains(filter, "$.task_key")
		kind := strings.Contains(filter, "$.type")
		rows, err := f.s.db.Query(`EXPLAIN QUERY PLAN `+eventStreamQuery(task, kind)+filter+` ORDER BY cursor LIMIT 21`, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		plan := ""
		for rows.Next() {
			var a, b, c int
			var detail string
			if err = rows.Scan(&a, &b, &c, &detail); err != nil {
				t.Fatal(err)
			}
			plan += detail
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		expected := "INTEGER PRIMARY KEY"
		if task && kind {
			expected = "events_task_type_cursor"
		} else if task {
			expected = "events_task_cursor"
		} else if kind {
			expected = "events_type_cursor"
		}
		if !strings.Contains(plan, expected) {
			t.Fatalf("wrong filter stream index %s: %s", filter, plan)
		}
		if !strings.Contains(plan, "SEARCH") || strings.Contains(plan, "TEMP B-TREE") {
			t.Fatal(fmt.Sprintf("unbounded plan %s: %s", filter, plan))
		}
	}
}
