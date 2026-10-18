package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"reflect"
	"testing"
	"time"
)

func TestEpicCancellationUsesConfiguredGrantOrRequester(t *testing.T) {
	for _, role := range []string{"human", "coordinator"} {
		for _, granted := range []bool{false, true} {
			for _, own := range []bool{false, true} {
				name := role + "/foreign/denied"
				if granted {
					name = role + "/grant"
				}
				if own {
					name += "/requester"
				}
				t.Run(name, func(t *testing.T) {
					f := newFixture(t)
					resetConfig(t, f, func(c *config.Config) {
						r := c.Roles[role]
						r.CanCancel = granted
						r.CanCreate = []string{"task", "epic"}
						c.Roles[role] = r
					})
					token, err := f.s.CreateToken(testContext, core.TokenInput{Roles: []string{role}, Owner: "alice", Machine: "other", TTL: time.Hour})
					if err != nil {
						t.Fatal(err)
					}
					p, err := f.s.Authenticate(testContext, token.Secret)
					if err != nil {
						t.Fatal(err)
					}
					session, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: f.id(), Role: role, RunID: "cancel", Machine: p.Machine})
					if err != nil {
						t.Fatal(err)
					}
					actor := core.Actor{Principal: p, SessionID: session.SessionID}
					epic := f.epic
					if own {
						epic, err = f.s.Create(testContext, actor, core.CreateInput{RequestID: f.id(), Type: "epic", Title: "own epic"})
						if err != nil {
							t.Fatal(err)
						}
					}
					before := f.get(t, epic.Key)
					events := eventCount(t, f)
					result, err := f.s.Transition(testContext, actor, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: epic.Key, Reason: "cancelled by request"}, State: "cancelled"})
					if granted || own {
						if err != nil || result.State != "cancelled" {
							t.Fatal("legal cancellation rejected", result, err)
						}
					} else {
						requireCode(t, err, "FORBIDDEN_ROLE")
						if !reflect.DeepEqual(before, f.get(t, epic.Key)) || eventCount(t, f) != events {
							t.Fatal("rejected cancellation changed epic/audit")
						}
					}
				})
			}
		}
	}
}

func TestIncompleteChildEvidenceBlocksEpicClosureAndReconciliation(t *testing.T) {
	for _, mode := range []string{"warn", "off"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			resetConfig(t, f, func(c *config.Config) {
				r := c.Roles["developer"]
				r.EvidenceOnDone = append(r.EvidenceOnDone, "pr")
				c.Roles["developer"] = r
				c.Rules.EvidenceRequired.Mode = mode
			})
			dev := f.actor(t, "developer")
			task := f.task(t, "developer")
			done := f.finish(t, dev, f.claim(t, dev, task.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "tested"}), "")
			if done.EvidenceComplete == nil || *done.EvidenceComplete {
				t.Fatal("expected incomplete evidence", done)
			}
			before := f.get(t, f.epic.Key)
			events := eventCount(t, f)
			_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, State: "done"})
			requireCode(t, err, "RULE_VIOLATION")
			if !reflect.DeepEqual(before, f.get(t, f.epic.Key)) || eventCount(t, f) != events {
				t.Fatal("rejected closure changed epic/audit")
			}
			// Model a persisted pre-fix done epic, without changing child history.
			if _, err = f.s.db.Exec(`UPDATE tasks SET data=json_set(data,'$.state','done') WHERE key=?`, f.epic.Key); err != nil {
				t.Fatal(err)
			}
			_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}})
			if err != nil {
				t.Fatal(err)
			}
			after := f.get(t, f.epic.Key)
			if after.State != "open" || !after.Blocked {
				t.Fatal("reconciliation retained incomplete done epic", after)
			}
			if !reflect.DeepEqual(done, f.get(t, task.Key)) {
				t.Fatal("reconciliation rewrote historical completion")
			}
			// Correct evidence still satisfies the same positive child gate.
			validEpic := f.newEpic(t)
			f.epic = validEpic
			valid := f.task(t, "developer")
			f.finish(t, dev, f.claim(t, dev, valid.Key), core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "tested", "pr": "https://example.test/pr/1"}), "")
			f.closeEpic(t, validEpic.Key)
			if result := f.get(t, validEpic.Key); result.State != "done" || result.Blocked {
				t.Fatal("complete evidence failed gate", result)
			}
		})
	}
}
