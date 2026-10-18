package store

import (
	"agent-board/internal/core"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLostResponsesReplayWithoutRenewalOrDuplicateEvents(t *testing.T) {
	f := newFixture(t)
	developer := f.actor(t, "developer")
	create := core.CreateInput{RequestID: "lost-create", Type: "task", EpicKey: f.epic.Key, Title: "Job", Description: "Work", Role: "developer"}
	task, err := f.s.Create(testContext, f.coord, create)
	if err != nil {
		t.Fatal(err)
	}
	input := core.NextInput{RequestID: "lost-next", TaskKey: task.Key}
	claim, err := f.s.Next(testContext, developer, input)
	if err != nil {
		t.Fatal(err)
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	before := page.HighWater
	f.clock.advance(30 * time.Second)
	duplicate, err := f.s.Create(testContext, f.coord, create)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Key != task.Key {
		t.Fatal("duplicate task")
	}
	replayed, err := f.s.Next(testContext, developer, input)
	if err != nil {
		t.Fatal(err)
	}
	originalJSON, _ := json.Marshal(claim)
	replayJSON, _ := json.Marshal(replayed)
	if string(originalJSON) != string(replayJSON) {
		t.Fatal("response changed on replay")
	}
	page, err = f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if page.HighWater != before {
		t.Fatal("replay emitted event")
	}
	create.Title = "Changed"
	_, err = f.s.Create(testContext, f.coord, create)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	f.clock.advance(30 * time.Second)
	_, err = f.s.Comment(testContext, developer, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "late"})
	requireCode(t, err, "LEASE_LOST")
}
func TestRejectedMutationLeavesNoPartialTaskOrEvents(t *testing.T) {
	f := newFixture(t)
	a := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, a, task.Key)
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Transition(testContext, a, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &core.Evidence{Commit: commitA}})
	requireCode(t, err, "EVIDENCE_MISSING")
	after, err := f.s.Get(testContext, a, core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	if after.State != "in_progress" || after.Evidence.Commit != "" {
		t.Fatal("rejected operation persisted task changes")
	}
	next, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if page.HighWater != next.HighWater {
		t.Fatal("rejected operation persisted event")
	}
}
func TestTokenHashExpiryRevocationAndSessionBinding(t *testing.T) {
	f := newFixture(t)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	secretBytes, decodeErr := hex.DecodeString(strings.TrimPrefix(token.Secret, "ab_"))
	if decodeErr != nil || len(secretBytes) < 32 {
		t.Fatal("token must contain at least 256 random bits")
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Describe(testContext, p, core.OpenSession{RequestID: "wrong-machine", Role: "developer", RunID: "run", Machine: "pc-a"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.Get(testContext, core.Actor{Principal: p, SessionID: f.coord.SessionID}, core.GetInput{TaskKey: f.epic.Key})
	requireCode(t, err, "UNAUTHORIZED")
	var hash string
	if err = f.s.db.QueryRow(`SELECT hash FROM tokens WHERE id=?`, token.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == token.Secret || len(hash) != 64 {
		t.Fatal("raw secret stored")
	}
	if err = f.s.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Authenticate(testContext, token.Secret)
	requireCode(t, err, "UNAUTHORIZED")
	_, err = f.s.Describe(testContext, p, core.OpenSession{RequestID: "revoked-replay", Role: "developer", RunID: "run", Machine: p.Machine})
	requireCode(t, err, "UNAUTHORIZED")
	expiring, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(time.Second)
	_, err = f.s.Authenticate(testContext, expiring.Secret)
	requireCode(t, err, "UNAUTHORIZED")
}
