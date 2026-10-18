package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestDeveloperAsksArchitectAndResumesAfterContract(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, dev, task.Key)
	request, err := f.s.Create(testContext, dev, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "Architecture question", Description: "Need decision", Role: "architect"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Link(testContext, dev, core.LinkInput{RequestID: f.id(), From: request.Key, To: task.Key, Type: "contract_of", Generation: claim.Lease.Generation})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Release(testContext, dev, core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "Waiting for architecture", Branch: "feature/work"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := f.s.Next(testContext, dev, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Task != nil {
		t.Fatal("dependency bypass")
	}
	architect := f.actor(t, "architect")
	f.finish(t, architect, f.claim(t, architect, request.Key), core.Evidence{Artifact: []string{"docs/decision.md"}, Summary: "Decision"}, "")
	resumed := f.claim(t, dev, task.Key)
	if resumed.Lease.Generation != 2 {
		t.Fatal("generation did not persist release")
	}
	if resumed.Task.Attempts[0].Branch != "feature/work" {
		t.Fatal("handoff branch lost")
	}
}
func TestLinkRequiresLiveConsumerGeneration(t *testing.T) {
	f := newFixture(t)
	dev := f.actor(t, "developer")
	target := f.task(t, "developer")
	source := f.task(t, "architect")
	claim := f.claim(t, dev, target.Key)
	_, err := f.s.Link(testContext, dev, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "blocks", Generation: claim.Lease.Generation - 1})
	requireCode(t, err, "LEASE_LOST")
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "blocks", Generation: claim.Lease.Generation})
	requireCode(t, err, "LEASE_LOST")
	_, err = f.s.Link(testContext, dev, core.LinkInput{RequestID: f.id(), From: source.Key, To: target.Key, Type: "blocks", Generation: claim.Lease.Generation})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSourceRequesterCannotBlockForeignReadyConsumer(t *testing.T) {
	f := newFixture(t)
	consumer := f.task(t, "developer")
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	session, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: "open", Role: "developer", RunID: "bob", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	bob := core.Actor{Principal: p, SessionID: session.SessionID}
	source, err := f.s.Create(testContext, bob, core.CreateInput{RequestID: "source", Type: "task", EpicKey: f.epic.Key, Title: "Decision", Description: "Question", Role: "architect"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Link(testContext, bob, core.LinkInput{RequestID: "malicious-link", From: source.Key, To: consumer.Key, Type: "blocks"})
	requireCode(t, err, "FORBIDDEN_ROLE")
	target, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: consumer.Key})
	if err != nil {
		t.Fatal(err)
	}
	if target.Blocked {
		t.Fatal("foreign dependency persisted")
	}
}
