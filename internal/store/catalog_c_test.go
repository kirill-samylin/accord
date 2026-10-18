package store

import (
	"reflect"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestCatalogStableNumericBoundaryAndNoRenewal(t *testing.T) {
	f := newFixture(t)
	var expected []string
	for range 12 {
		expected = append(expected, f.task(t, "developer").Key)
	}
	before := f.s.cfg.SessionTTL
	page, err := f.s.ListTasks(testContext, f.coord, core.ListInput{Limit: 1, Role: "developer", Owner: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{page.Tasks[0].Key}
	high := page.HighWater
	f.task(t, "developer")
	for page.HasMore {
		page, err = f.s.ListTasks(testContext, f.coord, core.ListInput{Limit: 1, Role: "developer", Owner: "alice", Cursor: page.NextCursor})
		if err != nil {
			t.Fatal(err)
		}
		if page.HighWater != high {
			t.Fatal("creation boundary changed")
		}
		for _, v := range page.Tasks {
			got = append(got, v.Key)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("numeric pass got %v want %v", got, expected)
	}
	p, err := f.s.ListTasks(testContext, f.coord, core.ListInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.ListTasks(testContext, f.coord, core.ListInput{Limit: 1, Cursor: p.NextCursor, Role: "architect"})
	requireCode(t, err, "VALIDATION")
	f.clock.advance(before)
	_, err = f.s.ListTasks(testContext, f.coord, core.ListInput{})
	requireCode(t, err, "LEASE_LOST")
}
func TestSessionCatalogBoundaryAndHistory(t *testing.T) {
	f := newFixture(t)
	f.actor(t, "developer")
	f.actor(t, "analyst")
	p, err := f.s.Sessions(testContext, f.coord, core.ListInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	high := p.HighWater
	got := 1
	f.actor(t, "human")
	for p.HasMore {
		p, err = f.s.Sessions(testContext, f.coord, core.ListInput{Limit: 1, Cursor: p.NextCursor})
		if err != nil {
			t.Fatal(err)
		}
		got += len(p.Sessions)
		if p.HighWater != high {
			t.Fatal("boundary changed")
		}
	}
	if got != 3 {
		t.Fatalf("got %d sessions", got)
	}
	f.clock.advance(time.Hour)
	active := f.actor(t, "coordinator")
	p, err = f.s.Sessions(testContext, active, core.ListInput{State: "expired", Owner: "alice"})
	if err != nil || len(p.Sessions) != 4 {
		t.Fatalf("history %d %v", len(p.Sessions), err)
	}
}
func TestTaskMetadataUniqueAndReadHistory(t *testing.T) {
	f := newFixture(t)
	in := core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "a", Description: "a", Role: "developer", Category: "technical", Complexity: "normal", ExternalID: "source-1"}
	v, err := f.s.Create(testContext, f.coord, in)
	if err != nil {
		t.Fatal(err)
	}
	in.RequestID = f.id()
	_, err = f.s.Create(testContext, f.coord, in)
	requireCode(t, err, "CONFLICT")
	bad := "other"
	_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{TaskKey: v.Key, RequestID: f.id()}, Complexity: &bad})
	requireCode(t, err, "VALIDATION")
	d := f.actor(t, "developer")
	f.finish(t, d, f.claim(t, d, v.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "test"}), "")
	p, err := f.s.ListTasks(testContext, f.coord, core.ListInput{})
	if err != nil || len(p.Tasks) != 0 {
		t.Fatal("closed task in live list")
	}
	p, err = f.s.ListTasks(testContext, f.coord, core.ListInput{State: "done"})
	if err != nil || len(p.Tasks) != 1 || p.Tasks[0].CreatedAt == nil || p.Tasks[0].FirstClaimAt == nil || p.Tasks[0].FirstDoneAt == nil {
		t.Fatalf("missing history %v %v", p, err)
	}
}
