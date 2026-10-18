package acceptance

import (
	"sync"
	"testing"

	"agent-board/internal/core"
)

func TestCLIConcurrentClaim(t *testing.T) {
	b := newBoard(t, "30s")
	coordinator := newClient(t, b, "coordinator")
	a := newClient(t, b, "developer-a")
	z := newClient(t, b, "developer-z")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	a.open(t, "developer", "developer-a")
	z.open(t, "developer", "developer-z")
	if a.session == z.session {
		t.Fatal("independent runs share a board session")
	}
	epic := coordinator.epic(t)
	task := coordinator.task(t, epic.Key, "developer")
	read := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if epic.State != "open" || read.Key != task.Key || read.State != "ready" {
		t.Fatal("CLI create/get lost initial state")
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, c := range []*client{a, z} {
		input := c.prepare(t, payload{"task_key": task.Key})
		wg.Add(1)
		go func(c *client, input []byte) {
			defer wg.Done()
			<-start
			results <- c.raw("next", input)
		}(c, input)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	var winner core.Claim
	for r := range results {
		if r.err != nil {
			t.Fatal("concurrent CLI next failed")
		}
		claim := decode[core.Claim](t, r.data)
		if claim.Task != nil {
			winners++
			winner = claim
		}
	}
	if winners != 1 || winner.Lease == nil {
		t.Fatalf("concurrent independent CLI processes produced %d winners", winners)
	}
	read = decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if read.State != "in_progress" || read.Lease == nil || read.Lease.SessionID != winner.Lease.SessionID || len(read.Attempts) != 1 {
		t.Fatal("persisted lease/attempt disagrees with the single winner")
	}
	t.Log("two independent CLI processes and board sessions; one owner, one persisted attempt")
}

func TestCLIArchitectureHandoff(t *testing.T) {
	b := newBoard(t, "30s")
	coordinator := newClient(t, b, "coordinator")
	developer := newClient(t, b, "developer")
	architect := newClient(t, b, "architect")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	developer.open(t, "developer", "developer")
	architect.open(t, "architect", "architect")
	epic := coordinator.epic(t)
	task := coordinator.task(t, epic.Key, "developer")
	first := developer.claim(t, task.Key)
	request := developer.task(t, epic.Key, "architect")
	developer.fails(t, "link", "LEASE_LOST", payload{"from": request.Key, "to": task.Key, "type": "contract_of", "generation": first.Lease.Generation + 1})
	developer.ok(t, "link", payload{"from": request.Key, "to": task.Key, "type": "contract_of", "generation": first.Lease.Generation})
	developer.ok(t, "release", payload{"task_key": task.Key, "generation": first.Lease.Generation, "reason": "Waiting for architecture", "branch": "synthetic/architecture"})
	blocked := decode[core.Claim](t, developer.ok(t, "next", payload{"task_key": task.Key}))
	if blocked.Task != nil {
		t.Fatal("developer bypassed an unfinished architecture dependency")
	}
	decision := architect.claim(t, request.Key)
	architect.ok(t, "transition", payload{"task_key": request.Key, "generation": decision.Lease.Generation, "state": "done", "evidence": core.EvidenceOf(map[string]any{"artifact": []string{"docs/synthetic-decision.md"}, "summary": "Synthetic agreed contract"})})
	resumed := developer.claim(t, task.Key)
	if resumed.Lease.Generation != first.Lease.Generation+1 || resumed.Task.Key != first.Task.Key || len(resumed.Task.Attempts) != 2 || resumed.Task.Attempts[0].Branch != "synthetic/architecture" {
		t.Fatal("resumed developer task lost its key, generation or handoff branch")
	}
	read := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": request.Key}))
	summary, summaryErr := read.Evidence.String("summary")
	artifacts, artifactErr := read.Evidence.Strings("artifact")
	if read.State != "done" || summaryErr != nil || artifactErr != nil || summary == "" || len(artifacts) != 1 {
		t.Fatal("architect completion evidence was not stored")
	}
	t.Log("live generation checked; dependency blocked until architect evidence; original developer task reacquired")
}

func TestCLILeaseExpiry(t *testing.T) {
	b := newBoard(t, "1s")
	coordinator := newClient(t, b, "coordinator")
	a := newClient(t, b, "developer-a")
	z := newClient(t, b, "developer-z")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	a.open(t, "developer", "developer-a")
	z.open(t, "developer", "developer-z")
	task := coordinator.task(t, coordinator.epic(t).Key, "developer")
	old := a.claim(t, task.Key)
	b.waitExpired(old.Lease.ExpiresAt)
	a.fails(t, "heartbeat", "LEASE_LOST", payload{"task_key": task.Key, "generation": old.Lease.Generation})
	fresh := z.claim(t, task.Key)
	if fresh.Lease.Generation != old.Lease.Generation+1 {
		t.Fatal("expired lease generation was reused")
	}
	a.fails(t, "comment", "LEASE_LOST", payload{"task_key": task.Key, "generation": old.Lease.Generation, "text": "Synthetic stale write"})
	a.fails(t, "heartbeat", "LEASE_LOST", payload{"task_key": task.Key, "generation": old.Lease.Generation})
	read := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if len(read.Comments) != 0 || len(read.Attempts) != 2 || read.Attempts[0].Reason != "lease_expired" || read.Lease == nil || read.Lease.SessionID != z.session {
		t.Fatal("stale request changed persisted ownership or audit")
	}
	t.Log("real 1s task TTL, server Date reached expiry; heartbeat cannot revive and stale writes fail")
}
