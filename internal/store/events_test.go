package store

import (
	"agent-board/internal/core"
	"testing"
)

func TestEventsCursorPaginationWithFilteringAndConcurrentAppend(t *testing.T) {
	f := newFixture(t)
	f.task(t, "developer")
	f.task(t, "analyst")
	f.task(t, "human")
	var cursor int64
	seen := map[int64]bool{}
	pages := 0
	for {
		page, err := f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: cursor, Limit: 1, Types: []string{"task_created"}})
		if err != nil {
			t.Fatal(err)
		}
		if page.NextCursor < cursor || page.NextCursor > page.HighWater {
			t.Fatalf("invalid cursor: %+v", page)
		}
		for _, e := range page.Events {
			if seen[e.Cursor] {
				t.Fatal("duplicate event")
			}
			seen[e.Cursor] = true
		}
		cursor = page.NextCursor
		pages++
		if pages == 1 {
			f.task(t, "developer")
		}
		if !page.HasMore {
			break
		}
		if pages > 20 {
			t.Fatal("infinite pagination")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("missed events: %d", len(seen))
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{Limit: 1, Types: []string{"absent"}})
	if err != nil {
		t.Fatal(err)
	}
	if page.HasMore || page.NextCursor != page.HighWater || len(page.Events) != 0 {
		t.Fatalf("empty scan: %+v", page)
	}
}
func TestDependencyCycleAndUnlinkAudit(t *testing.T) {
	f := newFixture(t)
	a := f.task(t, "developer")
	b := f.task(t, "analyst")
	c := f.task(t, "human")
	for _, edge := range [][2]string{{a.Key, b.Key}, {b.Key, c.Key}} {
		_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: edge[0], To: edge[1], Type: "blocks"})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: c.Key, To: a.Key, Type: "contract_of"})
	requireCode(t, err, "CYCLE")
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: b.Key, To: c.Key, Type: "blocks", Unlink: true})
	if err != nil {
		t.Fatal(err)
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{Types: []string{"unlinked"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 1 {
		t.Fatal("missing unlink audit")
	}
}
