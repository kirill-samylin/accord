package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestQuestionPagesUseGlobalQueueOrderAndDoNotRenew(t *testing.T) {
	f := newFixture(t)
	human := f.actor(t, "human")
	old := f.task(t, "human")
	f.clock.advance(time.Second)
	urgent, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Role: "human", Title: "Urgent", Description: "Synthetic urgent question", Priority: 1})
	if err != nil {
		t.Fatal(err)
	}
	_ = f.task(t, "developer")
	first, err := f.s.ListQuestions(testContext, human, core.ListInput{Limit: 1})
	if err != nil || len(first.Tasks) != 1 || first.Tasks[0].Key != urgent.Key || !first.HasMore {
		t.Fatalf("first: %+v %v", first, err)
	}
	second, err := f.s.ListQuestions(testContext, human, core.ListInput{Limit: 1, Cursor: first.NextCursor})
	if err != nil || len(second.Tasks) != 1 || second.Tasks[0].Key != old.Key || second.HasMore {
		t.Fatalf("second: %+v %v", second, err)
	}
	if _, err = f.s.ListQuestions(testContext, human, core.ListInput{Limit: 1, State: "done", Cursor: first.NextCursor}); err == nil {
		t.Fatal("cursor crossed filters")
	}
	claim := f.claim(t, human, urgent.Key)
	expiry := claim.Lease.ExpiresAt
	f.clock.advance(20 * time.Second)
	if _, err = f.s.ListQuestions(testContext, human, core.ListInput{}); err != nil {
		t.Fatal(err)
	}
	got, err := f.s.Get(testContext, human, core.GetInput{TaskKey: urgent.Key})
	if err != nil || got.Lease == nil || !got.Lease.ExpiresAt.Equal(expiry) {
		t.Fatal("question read renewed lease", err)
	}
}
