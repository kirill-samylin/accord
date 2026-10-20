package web

import (
	"context"
	"testing"

	"agent-board/internal/core"
)

func TestTaskDetailActivityIsScopedAndReadDoesNotRenew(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	task := f.question(t, token)
	actor := f.native(t, token)
	other, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "other", Type: "task", Role: "human", EpicKey: task.EpicKey, Title: "Other", Description: "Other context"})
	if err != nil {
		t.Fatal(err)
	}
	cookie := f.login(t, token)
	p := f.page(t, cookie, "/web/tasks/"+task.Key)
	if len(p.Activity) != 1 || p.Activity[0].TaskKey != task.Key || p.Activity[0].Type != "task_created" {
		t.Fatalf("task activity %+v", p.Activity)
	}
	if w := f.post(cookie, "/web/tasks/"+task.Key+"/claim", p, "claim", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	before := f.page(t, cookie, "/web/tasks/"+task.Key)
	after := f.page(t, cookie, "/web/tasks/"+task.Key)
	if before.Task.Lease == nil || !before.Task.Lease.ExpiresAt.Equal(after.Task.Lease.ExpiresAt) || before.Task.Generation != after.Task.Generation || !before.Session.ExpiresAt.Equal(after.Session.ExpiresAt) || len(before.Activity) != len(after.Activity) {
		t.Fatal("task timeline renewed lease/session or emitted events")
	}
	for i, event := range after.Activity {
		if event.TaskKey != task.Key || i > 0 && event.Cursor >= after.Activity[i-1].Cursor {
			t.Fatalf("cross-task or unordered activity %+v", after.Activity)
		}
	}
	otherPage := f.page(t, cookie, "/web/tasks/"+other.Key)
	if len(otherPage.Activity) != 1 || otherPage.Activity[0].TaskKey != other.Key {
		t.Fatal("other task timeline leaked events")
	}
	ep := f.page(t, cookie, "/web/tasks/"+task.EpicKey)
	if ep.Task.Type != "epic" || len(ep.Activity) != 0 {
		t.Fatal("epic detail should remain available without unsupported task filter")
	}
}

type taskActivityBoard struct {
	Board
	input core.RecentEventsInput
	actor core.Actor
}

func (b *taskActivityBoard) RecentEvents(_ context.Context, actor core.Actor, input core.RecentEventsInput) (core.RecentEventsPage, error) {
	b.input, b.actor = input, actor
	return core.RecentEventsPage{}, core.Fail("forbidden", "timeline unavailable")
}

func TestTaskDetailActivityBoundAndFailure(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	task := f.question(t, token)
	cookie := f.login(t, token)
	board := &taskActivityBoard{Board: f.board}
	f.s.board = board
	w := request(f.s, "GET", "/web/tasks/"+task.Key, nil, cookie)
	if w.Code != 403 || board.input != (core.RecentEventsInput{TaskKey: task.Key, Limit: 10}) || board.actor.SessionID == "" {
		t.Fatalf("task timeline error/bounds %d %+v", w.Code, board.input)
	}
}
