package web

import (
	"agent-board/internal/core"
	"context"
	"net/url"
	"strconv"
	"testing"
)

func TestEpicsHaveIndependentPaginationAndPreserveFilters(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "one")
	actor := f.native(t, token)
	for i := 0; i < 101; i++ {
		if _, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "epic-" + strconv.Itoa(i), Type: "epic", Title: "Epic " + strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
	}
	cookie := f.login(t, token)
	first := f.page(t, cookie, "/web/?view=tasks&state=done&role=developer")
	if len(first.Epics) != 100 || first.EpicsNextCursor == "" {
		t.Fatalf("first epic page %d/%q", len(first.Epics), first.EpicsNextCursor)
	}
	cursor := first.EpicsNextCursor
	second := f.page(t, cookie, "/web/?view=tasks&state=done&role=developer&epic_cursor="+url.QueryEscape(cursor))
	if len(second.Epics) != 1 || second.EpicsNextCursor != "" {
		t.Fatalf("second epic page %d/%q", len(second.Epics), second.EpicsNextCursor)
	}
	if second.State != "done" || second.Role != "developer" || second.Draft["epic_cursor"] != cursor {
		t.Fatal("independent pagination lost filters")
	}
}
