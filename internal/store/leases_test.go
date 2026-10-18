package store

import (
	"agent-board/internal/core"
	"sync"
	"testing"
	"time"
)

func TestConcurrentClaimAcrossSQLiteConnections(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	a := f.actor(t, "developer")
	z := f.actor(t, "developer")
	other, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	start := make(chan struct{})
	results := make(chan core.Claim, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, actor := range []core.Actor{a, z} {
		wg.Add(1)
		go func(i int, a core.Actor) {
			defer wg.Done()
			<-start
			s := f.s
			if i == 1 {
				s = other
			}
			v, err := s.Next(testContext, a, core.NextInput{RequestID: "concurrent", TaskKey: task.Key})
			results <- v
			errs <- err
		}(i, actor)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	owners := 0
	for v := range results {
		if v.Task != nil {
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("owners=%d", owners)
	}
}
func TestLeaseExpiryFencingRestartAndReadDoesNotRenew(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	a := f.actor(t, "developer")
	z := f.actor(t, "developer")
	old := f.claim(t, a, task.Key)
	f.clock.advance(50 * time.Second)
	if _, err := f.s.Get(testContext, a, core.GetInput{TaskKey: task.Key}); err != nil {
		t.Fatal(err)
	}
	f.clock.advance(10 * time.Second)
	_, err := f.s.Heartbeat(testContext, a, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: old.Lease.Generation})
	requireCode(t, err, "LEASE_LOST")
	fresh := f.claim(t, z, task.Key)
	if fresh.Lease.Generation != old.Lease.Generation+1 {
		t.Fatal("fence did not advance")
	}
	_, err = f.s.Comment(testContext, a, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: old.Lease.Generation}, Text: "stale"})
	requireCode(t, err, "LEASE_LOST")
	if err = f.s.Close(); err != nil {
		t.Fatal(err)
	}
	f.s, err = Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.s.Close() })
	current, err := f.s.Get(testContext, z, core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation != fresh.Lease.Generation || len(current.Attempts) != 2 || current.Attempts[0].Reason != "lease_expired" {
		t.Fatalf("lost provenance: %+v", current)
	}
}
func TestSingletonAndHeartbeatAtomicity(t *testing.T) {
	f := newFixture(t)
	a := f.actor(t, "architect")
	_, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "architect", RunID: "other", Machine: f.p.Machine})
	requireCode(t, err, "ROLE_HELD")
	task := f.task(t, "architect")
	claim := f.claim(t, a, task.Key)
	before, err := f.s.Get(testContext, a, core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(30 * time.Second)
	_, err = f.s.Heartbeat(testContext, a, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, RoleGeneration: 999})
	requireCode(t, err, "LEASE_LOST")
	after, err := f.s.Get(testContext, a, core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	if !before.Lease.ExpiresAt.Equal(after.Lease.ExpiresAt) {
		t.Fatal("failed heartbeat renewed lease")
	}
	f.clock.advance(time.Hour)
	z := f.actor(t, "architect")
	_, err = f.s.Heartbeat(testContext, a, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
	if z.SessionID == a.SessionID {
		t.Fatal("expired session resurrected")
	}
	f.actor(t, "coordinator")
	f.actor(t, "coordinator")
}
