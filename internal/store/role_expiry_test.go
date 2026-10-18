package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestRoleTTLExpiryFencesLiveSessionAndReclaimsTask(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		c.Leases.RoleTTL = "1s"
		c.Leases.TaskTTL = "1h"
		c.Leases.SessionTTL = "1h"
		c.Leases.MaxSilent = "1s"
		c.Leases.SweepInterval = "1s"
	})
	old := f.actor(t, "architect")
	task := f.task(t, "architect")
	claim := f.claim(t, old, task.Key)
	f.clock.advance(1500 * time.Millisecond)
	replacement := f.actor(t, "architect")
	_, err := f.s.Heartbeat(testContext, old, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation, RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
	_, err = f.s.Create(testContext, old, core.CreateInput{RequestID: f.id(), Type: "task", Title: "old writer", Role: "developer", Description: "x", EpicKey: f.epic.Key})
	requireCode(t, err, "LEASE_LOST")
	current := f.claim(t, replacement, task.Key)
	if current.Lease.Generation != claim.Lease.Generation+1 {
		t.Fatal("generation not advanced")
	}
	hb, err := f.s.Heartbeat(testContext, replacement, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: current.Lease.Generation, RoleGeneration: 2})
	if err != nil || hb.SessionTTLSeconds != 3600 || hb.RoleTTLSeconds != 1 || hb.TaskTTLSeconds != 3600 || hb.HeartbeatEverySeconds != 1.0/3 {
		t.Fatal(hb, err)
	}
}
