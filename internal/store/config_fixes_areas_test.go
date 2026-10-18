package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"reflect"
	"testing"
	"time"
)

func eventCount(t *testing.T, f *fixture) int {
	t.Helper()
	var count int
	if err := f.s.db.QueryRow(`SELECT count(*) FROM events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestLiveAreaUpdatesRequireEditorRole(t *testing.T) {
	for _, mode := range []string{"off", "warn", "block"} {
		for _, path := range []string{"secret", "restricted/child"} {
			t.Run(mode+"/"+path, func(t *testing.T) {
				f := newFixture(t)
				resetConfig(t, f, func(c *config.Config) {
					c.Rules.AreaOverlap.Mode = mode
					c.Areas = []config.Area{{Name: "public", Aliases: []string{}}, {Name: "restricted", Aliases: []string{"secret"}, SingleEditorRole: "analyst"}}
				})
				task, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "safe work", Description: "implement", Role: "developer", Areas: []string{"public"}})
				if err != nil {
					t.Fatal(err)
				}
				dev := f.actor(t, "developer")
				claim := f.claim(t, dev, task.Key)
				before := f.get(t, task.Key)
				events := eventCount(t, f)
				f.clock.advance(time.Second)
				areas := []string{path}
				input := core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Areas: &areas}
				_, err = f.s.Update(testContext, dev, input)
				requireCode(t, err, "FORBIDDEN_ROLE")
				if after := f.get(t, task.Key); !reflect.DeepEqual(before, after) || eventCount(t, f) != events {
					t.Fatal("denied update changed metadata/lease/audit")
				}
				input.Generation++
				input.RequestID = f.id()
				_, err = f.s.Update(testContext, dev, input)
				requireCode(t, err, "LEASE_LOST")
				other := f.actor(t, "developer")
				input.Generation = claim.Lease.Generation
				input.RequestID = f.id()
				_, err = f.s.Update(testContext, other, input)
				requireCode(t, err, "LEASE_LOST")
				// Planning an unclaimed task for another role remains legal.
				planned := f.task(t, "analyst")
				_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: planned.Key}, Areas: &areas})
				if err != nil {
					t.Fatal(err)
				}
				analyst := f.actor(t, "analyst")
				positive := f.claim(t, analyst, planned.Key)
				areas = []string{"restricted/other"}
				_, err = f.s.Update(testContext, analyst, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: planned.Key, Generation: positive.Lease.Generation}, Areas: &areas})
				if err != nil {
					t.Fatal("editor cannot update own live area", err)
				}
			})
		}
	}
}

func TestAutomaticQueueSkipsAreaBlockedCandidate(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		c.Rules.AreaOverlap.Mode = "block"
		c.Areas = []config.Area{{Name: "busy", Aliases: []string{}}, {Name: "free", Aliases: []string{}}}
	})
	create := func(title, area string, priority int) core.Task {
		t.Helper()
		task, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: title, Description: "implement", Role: "developer", Areas: []string{area}, Priority: priority})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	active := create("holding", "busy", 1)
	holder := f.actor(t, "developer")
	f.claim(t, holder, active.Key)
	originalEpic := f.epic
	skippedEpic := f.newEpic(t)
	f.epic = skippedEpic
	blockedTask := create("blocked candidate", "busy/child", 2)
	f.epic = originalEpic
	free := create("free candidate", "free", 3)
	before := f.get(t, blockedTask.Key)
	dev := f.actor(t, "developer")
	view, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "queue view", Machine: f.p.Machine})
	if err != nil || view.Queue.Ready != 1 || view.Queue.Waiting != 1 {
		t.Fatal("inaccurate describe queue", view, err)
	}
	events := eventCount(t, f)
	_, err = f.s.Next(testContext, dev, core.NextInput{RequestID: f.id(), TaskKey: blockedTask.Key})
	requireCode(t, err, "AREA_OVERLAP")
	if eventCount(t, f) != events {
		t.Fatal("explicit rejection emitted events")
	}
	claim, err := f.s.Next(testContext, dev, core.NextInput{RequestID: f.id()})
	if err != nil || claim.Task == nil || claim.Task.Key != free.Key {
		t.Fatal("queue did not progress", claim, err)
	}
	if len(claim.Task.Warnings) != 0 || !reflect.DeepEqual(before, f.get(t, blockedTask.Key)) || f.get(t, skippedEpic.Key).Owner != "" {
		t.Fatal("skipped candidate was mutated")
	}
	if claim.Queue.Ready != 0 || claim.Queue.Waiting != 1 || claim.Queue.InProgress != 2 {
		t.Fatal("inaccurate next queue", claim.Queue)
	}
	stats, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: f.clock.read().Add(-time.Hour), To: f.clock.read().Add(time.Hour), GroupBy: "role"})
	if err != nil || stats.Totals.CurrentQueue != claim.Queue {
		t.Fatal("inaccurate stats queue", stats.Totals.CurrentQueue, err)
	}
	for _, g := range stats.Groups {
		if g.Key == "developer" && g.Metrics.CurrentQueue != claim.Queue {
			t.Fatal("group queue diverged", g)
		}
	}
}
