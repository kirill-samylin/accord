package store

import (
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestControlEmptyStatesBlockedAndExpiredSessions(t *testing.T) {
	f := newFixture(t)
	empty, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil || empty.TasksTotal != 0 || empty.OpenEpics != 1 || empty.ActiveAgentSessions != 1 || empty.LeasedTasks != 0 {
		t.Fatalf("empty %+v %v", empty, err)
	}
	states := []string{"created", "ready", "in_progress", "done", "cancelled"}
	keys := []string{}
	for range states {
		keys = append(keys, f.task(t, "analyst").Key)
	}
	a := f.actor(t, "analyst")
	err = f.s.adminTransaction(testContext, func(b *board, now time.Time) error {
		for i, key := range keys {
			b.Tasks[key].State = states[i]
		}
		// Two blocked nonterminal tasks; a blocked terminal is excluded from the badge.
		for _, i := range []int{0, 1, 3} {
			b.Tasks[keys[i]].Links = []core.Link{{From: keys[2], To: keys[i], Type: "blocks", Condition: "completed"}}
		}
		b.Sessions[a.SessionID].ExpiresAt = now.Add(-time.Second)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	counts, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil || counts.TasksTotal != 5 || counts.TasksCreated != 1 || counts.TasksReady != 1 || counts.TasksInProgress != 1 || counts.TasksDone != 1 || counts.TasksCancelled != 1 || counts.TasksBlocked != 2 || counts.ActiveAgentSessions != 1 {
		t.Fatalf("states %+v %v", counts, err)
	}
	expired, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: a.SessionID, State: "expired"})
	if err != nil || len(expired.Sessions) != 1 || expired.Sessions[0].Status != "expired" {
		t.Fatalf("expired %+v %v", expired, err)
	}
}
