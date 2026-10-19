package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHumanDecisionIsImmutableAndReplayDoesNotDuplicate(t *testing.T) {
	f := newFixture(t)
	human := f.actor(t, "human")
	q := f.task(t, "human")
	claim := f.claim(t, human, q.Key)
	in := core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: "answer-once", TaskKey: q.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &core.Evidence{Answer: "First decision"}}
	if _, err := f.s.Transition(testContext, human, in); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Transition(testContext, human, in); err != nil {
		t.Fatal(err)
	}
	page, err := f.s.ListDecisions(testContext, human, core.DecisionsInput{})
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("replay history: %+v %v", page, err)
	}
	e := page.Events[0]
	if e.Data == nil || e.Data.Answer == nil || *e.Data.Answer != "First decision" || e.Data.AuthorPrincipal != f.p.ID || e.Data.Machine != f.p.Machine || e.Data.QuestionTitle != q.Title || e.Data.Generation != 1 {
		t.Fatalf("missing frozen provenance: %+v", e)
	}
	// A later answer replaces current evidence but cannot rewrite the first event.
	if _, err = f.s.Transition(testContext, human, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: q.Key, Reason: "Clarify decision"}, State: "ready", EvidenceRef: q.Key}); err != nil {
		t.Fatal(err)
	}
	f.clock.advance(time.Second)
	f.finish(t, human, f.claim(t, human, q.Key), core.Evidence{Answer: "Revised decision"}, "")
	page, err = f.s.ListDecisions(testContext, human, core.DecisionsInput{Limit: 1})
	if err != nil || len(page.Events) != 1 || !page.HasMore || *page.Events[0].Data.Answer != "Revised decision" {
		t.Fatalf("newest page: %+v %v", page, err)
	}
	older, err := f.s.ListDecisions(testContext, human, core.DecisionsInput{BeforeCursor: page.NextCursor, Limit: 1})
	if err != nil || len(older.Events) != 1 || older.HasMore || *older.Events[0].Data.Answer != "First decision" {
		t.Fatalf("older page: %+v %v", older, err)
	}
}

func TestDecisionReadRequiresLiveSessionAndCurrentToken(t *testing.T) {
	f := newFixture(t)
	human := f.actor(t, "human")
	q := f.task(t, "human")
	f.finish(t, human, f.claim(t, human, q.Key), core.Evidence{Answer: "Answer"}, "")
	if _, err := f.s.ListDecisions(testContext, core.Actor{Principal: f.p, SessionID: "other"}, core.DecisionsInput{}); err == nil {
		t.Fatal("foreign session accepted")
	}
	f.clock.advance(2 * time.Hour)
	_, err := f.s.ListDecisions(testContext, human, core.DecisionsInput{})
	requireCode(t, err, "LEASE_LOST")
	fresh := f.actor(t, "human")
	if err := f.s.RevokeToken(testContext, f.p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.ListDecisions(testContext, fresh, core.DecisionsInput{}); err == nil {
		t.Fatal("revoked identity accepted")
	}
}

func TestDecisionRowsRemainAppendOnlyDuringOldShapeRead(t *testing.T) {
	f := newFixture(t)
	human := f.actor(t, "human")
	q := f.task(t, "human")
	f.finish(t, human, f.claim(t, human, q.Key), core.Evidence{Answer: "Durable answer"}, "")
	var before string
	if err := f.s.db.QueryRow(`SELECT data FROM events WHERE json_extract(data,'$.type')='human_answered'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	// Old readers ignore new fields. Board reads must not reserialize raw events.
	var old struct {
		Type string `json:"type"`
		Data *struct {
			Kind string `json:"kind"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(before), &old); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Get(context.Background(), human, core.GetInput{TaskKey: q.Key}); err != nil {
		t.Fatal(err)
	}
	var after string
	if err := f.s.db.QueryRow(`SELECT data FROM events WHERE json_extract(data,'$.type')='human_answered'`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("read rewrote immutable history")
	}
}
