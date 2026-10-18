package store

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestCancelledEpicCannotSupplyNewClaims(t *testing.T) {
	for _, selection := range []string{"queue", "epic", "task"} {
		t.Run(selection, func(t *testing.T) {
			f := newFixture(t)
			task := f.task(t, "developer")
			developer := f.actor(t, "developer")
			_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key, Reason: "pause epic"}, State: "cancelled"})
			if err != nil {
				t.Fatal(err)
			}
			input := core.NextInput{RequestID: f.id()}
			if selection == "epic" {
				input.EpicKey = f.epic.Key
			} else if selection == "task" {
				input.TaskKey = task.Key
			}
			claim, err := f.s.Next(testContext, developer, input)
			if err != nil || claim.Task != nil || claim.Lease != nil {
				t.Fatalf("cancelled epic supplied claim: %+v %v", claim, err)
			}
			current, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: task.Key})
			if err != nil || current.State != "ready" || current.Generation != 0 || len(current.Attempts) != 0 {
				t.Fatal("claim mutated child of cancelled epic")
			}
			_, err = f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key, Reason: "resume epic"}, State: "open"})
			if err != nil {
				t.Fatal(err)
			}
			if resumed := f.claim(t, developer, task.Key); resumed.Lease.Generation != 1 {
				t.Fatal("reopening did not restore normal first claim")
			}
		})
	}
}

func TestExpiredGenerationCannotFallBackToAdministrativeRights(t *testing.T) {
	for _, role := range []string{"developer", "coordinator"} {
		for _, operation := range []string{"update", "link", "unlink"} {
			t.Run(role+"/"+operation, func(t *testing.T) {
				f := newFixture(t)
				holder := f.actor(t, role)
				// The holder is also the requester, so the unclaimed admin path
				// would otherwise admit this stale operation after expiration.
				target, err := f.s.Create(testContext, holder, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "Consumer", Description: "Synthetic", Role: role})
				if err != nil {
					t.Fatal(err)
				}
				source := f.task(t, "analyst")
				if operation == "unlink" {
					_, err = f.s.Link(testContext, holder, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "related"})
					if err != nil {
						t.Fatal(err)
					}
				}
				claim := f.claim(t, holder, target.Key)
				f.clock.advance(time.Minute)
				before, err := f.s.Get(testContext, holder, core.GetInput{TaskKey: target.Key})
				if err != nil || before.Lease != nil {
					t.Fatal("expiration was not materialized")
				}
				beforeSession := storedSession(t, f, holder)
				var beforeEvents, beforeReplays int
				if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&beforeEvents); err != nil {
					t.Fatal(err)
				}
				if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM idempotency`).Scan(&beforeReplays); err != nil {
					t.Fatal(err)
				}
				f.clock.advance(time.Second)
				requestID := f.id()
				mutate := func(generation int64) error {
					if operation == "update" {
						title := "Changed"
						_, err := f.s.Update(testContext, holder, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: requestID, TaskKey: target.Key, Generation: generation}, Title: &title})
						return err
					}
					_, err := f.s.Link(testContext, holder, core.LinkInput{RequestID: requestID, From: source.Key, To: target.Key, Type: "related", Unlink: operation == "unlink", Generation: generation})
					return err
				}
				requireCode(t, mutate(claim.Lease.Generation), "LEASE_LOST")
				after, err := f.s.Get(testContext, holder, core.GetInput{TaskKey: target.Key})
				if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeSession, storedSession(t, f, holder)) {
					t.Fatal("rejected stale mutation changed task or renewed session")
				}
				var afterEvents, afterReplays int
				if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&afterEvents); err != nil {
					t.Fatal(err)
				}
				if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM idempotency`).Scan(&afterReplays); err != nil {
					t.Fatal(err)
				}
				if beforeEvents != afterEvents || beforeReplays != afterReplays {
					t.Fatal("rejected stale mutation persisted audit or replay")
				}
				// An explicitly administrative operation with no generation is
				// still permitted and the rejected request ID remains unused.
				if err := mutate(0); err != nil {
					t.Fatalf("explicit administrative operation rejected: %v", err)
				}
			})
		}
	}
}

func storedSession(t *testing.T, f *fixture, actor core.Actor) core.Session {
	t.Helper()
	var raw string
	if err := f.s.db.QueryRow(`SELECT data FROM sessions WHERE key=?`, actor.SessionID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var session core.Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		t.Fatal(err)
	}
	return session
}
