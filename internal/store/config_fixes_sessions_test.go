package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"testing"
	"time"
)

func TestSingletonExpiryRemainsFencedAcrossConfigChanges(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		c.Leases.RoleTTL = "1s"
		c.Leases.SessionTTL = "1h"
		c.Leases.TaskTTL = "1h"
		c.Leases.SweepInterval = "1s"
		c.Leases.MaxSilent = "1s"
	})
	// Expire the setup coordinator; only the soon-expiring singleton is active.
	f.clock.advance(2 * time.Hour)
	old := f.actor(t, "architect")
	f.clock.advance(time.Second)
	f.configure(t, func(c *config.Config) {
		role := c.Roles["architect"]
		role.Singleton = false
		c.Roles["architect"] = role
	})
	// A second connection must enforce the stored lease kind as well.
	other, err := OpenExisting(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	assertStale := func() {
		t.Helper()
		_, err := other.Heartbeat(testContext, old, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
		requireCode(t, err, "LEASE_LOST")
		_, err = other.Create(testContext, old, core.CreateInput{RequestID: f.id(), Type: "task", Title: "stale write", Description: "implement", Role: "developer", EpicKey: f.epic.Key})
		requireCode(t, err, "LEASE_LOST")
	}
	assertStale()
	var saved config.Role
	f.configure(t, func(c *config.Config) {
		saved = c.Roles["architect"]
		delete(c.Roles, "architect")
		delete(c.MCP.ToolsByRole, "architect")
		c.Reopen["developer"] = []string{"analyst"}
	})
	assertStale()
	f.configure(t, func(c *config.Config) {
		saved.Singleton = true
		c.Roles["architect"] = saved
		c.MCP.ToolsByRole = map[string][]string{}
		c.Reopen["developer"] = []string{"analyst", "architect"}
	})
	replacement := f.actor(t, "architect")
	current, err := f.s.Heartbeat(testContext, replacement, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 2})
	if err != nil || current.RoleGeneration != 2 {
		t.Fatal("generation reset on role re-add", current, err)
	}
	assertStale()
	_, err = f.s.Heartbeat(testContext, replacement, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
	f.clock.advance(time.Second)
	f.configure(t, func(c *config.Config) {
		role := c.Roles["architect"]
		role.Singleton = false
		c.Roles["architect"] = role
	})
	assertStale()
	fresh := f.actor(t, "architect")
	hb, err := f.s.Heartbeat(testContext, fresh, core.HeartbeatInput{RequestID: f.id()})
	if err != nil || hb.RoleGeneration != 0 {
		t.Fatal("new ordinary session unavailable", hb, err)
	}
}

func TestLegacySingletonWithoutRoleDeadlineUsesGenerationAndSessionExpiry(t *testing.T) {
	f := newFixture(t)
	old := f.actor(t, "architect")
	// Historical sessions predate the additive role_expires_at field.
	if _, err := f.s.db.Exec(`UPDATE sessions SET data=json_remove(data,'$.role_expires_at') WHERE key=?`, old.SessionID); err != nil {
		t.Fatal(err)
	}
	hb, err := f.s.Heartbeat(testContext, old, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	if err != nil || hb.RoleGeneration != 1 {
		t.Fatal("legacy session lost compatibility", hb, err)
	}
	var raw []byte
	if err = f.s.db.QueryRow(`SELECT data FROM sessions WHERE key=?`, old.SessionID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var stored core.Session
	if err = json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	// A stale generation stays fenced even without the additive deadline.
	if _, err = f.s.db.Exec(`UPDATE sessions SET data=json_remove(data,'$.role_expires_at') WHERE key=?`, old.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.db.Exec(`UPDATE role_generations SET generation=generation+1 WHERE role='architect'`); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Heartbeat(testContext, old, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
	// Session expiry is still authoritative for a matching historical generation.
	if _, err = f.s.db.Exec(`UPDATE role_generations SET generation=? WHERE role='architect'`, stored.RoleGeneration); err != nil {
		t.Fatal(err)
	}
	f.clock.advance(time.Hour)
	f.configure(t, func(c *config.Config) { r := c.Roles["architect"]; r.Singleton = false; c.Roles["architect"] = r })
	_, err = f.s.Heartbeat(testContext, old, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1})
	requireCode(t, err, "LEASE_LOST")
}
