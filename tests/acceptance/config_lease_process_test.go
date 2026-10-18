package acceptance

import (
	"testing"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestConfigLostSingletonSIDNeverRevivesProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		c.Leases.RoleTTL = "1s"
		c.Leases.SessionTTL = "1h"
		c.Leases.TaskTTL = "1h"
		c.Leases.MaxSilent = "1s"
		c.Leases.SweepInterval = "1s"
		r := c.Roles["architect"]
		r.CanCreate = []string{"task", "epic"}
		c.Roles["architect"] = r
		r = c.Roles["coordinator"]
		r.Singleton = true
		c.Roles["coordinator"] = r
	})
	old := newClient(t, b, "singleton-old")
	admin := newClient(t, b, "singleton-admin")
	fresh := newClient(t, b, "singleton-fresh")
	startExistingImplicit(t, b)
	view := old.open(t, "architect", "singleton-old")
	if view.RoleExpiresAt == nil || view.RoleTTLSeconds != 1 || view.SessionTTLSeconds != 3600 || view.TaskTTLSeconds != 3600 {
		t.Fatal("independent role/session/task TTL not returned")
	}
	epic := old.epic(t)
	task := old.task(t, epic.Key, "architect")
	claim := old.claim(t, task.Key)
	// Last successful claim renews the role; fetch its persisted server deadline.
	db := inspectDatabase(t, b.db)
	var deadline time.Time
	raw := sqlString(t, db, "SELECT data FROM sessions WHERE key='"+old.session+"'")
	session := decode[core.Session](t, []byte(raw))
	deadline = *session.RoleExpiresAt
	b.waitExpired(deadline)
	old.fails(t, "heartbeat", "LEASE_LOST", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "role_generation": view.RoleGeneration})
	applyConfig(t, b, func(c *config.Config) { r := c.Roles["architect"]; r.Singleton = false; c.Roles["architect"] = r })
	old.fails(t, "heartbeat", "LEASE_LOST", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "role_generation": view.RoleGeneration})
	old.fails(t, "comment", "LEASE_LOST", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "text": "stale singleton"})
	// Cancel the expired task via a new short singleton coordinator. Removal of
	// roles referenced by nonterminal work remains protected, so finish it first.
	adminView := admin.open(t, "coordinator", "singleton-admin")
	admin.ok(t, "transition", payload{"task_key": task.Key, "state": "cancelled", "reason": "synthetic completed fixture"})
	raw = sqlString(t, db, "SELECT data FROM sessions WHERE key='"+admin.session+"'")
	session = decode[core.Session](t, []byte(raw))
	b.waitExpired(*session.RoleExpiresAt)
	var saved config.Role
	applyConfig(t, b, func(c *config.Config) {
		saved = c.Roles["architect"]
		delete(c.Roles, "architect")
		delete(c.MCP.ToolsByRole, "architect")
		c.Reopen["developer"] = []string{"analyst"}
	})
	old.fails(t, "heartbeat", "LEASE_LOST", payload{"role_generation": view.RoleGeneration})
	applyConfig(t, b, func(c *config.Config) {
		saved.Singleton = true
		c.Roles["architect"] = saved
		c.MCP.ToolsByRole = map[string][]string{}
		c.Reopen["developer"] = []string{"analyst", "architect"}
	})
	successor := fresh.open(t, "architect", "singleton-fresh")
	if successor.RoleGeneration != view.RoleGeneration+1 {
		t.Fatal("delete/readd reset role generation")
	}
	old.fails(t, "heartbeat", "LEASE_LOST", payload{"role_generation": view.RoleGeneration})
	old.fails(t, "create", "LEASE_LOST", payload{"type": "task", "epic_key": epic.Key, "role": "developer", "title": "stale write", "description": "deny"})
	if adminView.RoleGeneration < 1 {
		t.Fatal("test coordinator did not have bounded singleton lease")
	}
	t.Log("D056-09 roleTTL1s with session/task1h; expired task fenced; toggle/delete/readd keep old SID lost and advance new singleton generation")
}

func TestConfigRulesLiveReloadAfterSessionExpiryProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		c.Leases.RoleTTL = "1s"
		c.Leases.SessionTTL = "1s"
		c.Leases.MaxSilent = "1s"
		c.Leases.SweepInterval = "1s"
	})
	coord := newClient(t, b, "rule-coord")
	b.startOptions()
	v := coord.open(t, "coordinator", "rule-coord")
	pid := b.cmd.Process.Pid
	b.waitExpired(v.ExpiresAt)
	applied := applyConfig(t, b, func(c *config.Config) { c.Rules.DuplicateCheck.Mode = "block" })
	coord.session = ""
	view := coord.open(t, "coordinator", "rule-coord")
	if view.ConfigVersion != applied.ConfigVersion || view.Rules["duplicate_check"] != "block" || b.cmd.Process.Pid != pid {
		t.Fatal("running server did not load new rules after safe apply")
	}
	epic := coord.epic(t)
	coord.task(t, epic.Key, "developer")
	coord.fails(t, "create", "DUPLICATE", payload{"type": "task", "epic_key": epic.Key, "role": "developer", "title": "Synthetic slice", "description": "Process acceptance"})
	t.Log("D056-10 after natural session expiry other CLI process applies rules; same server advertises and enforces new duplicate block without restart")
}

func TestConfigStdioRenewsMinimumRoleTTLProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		c.Leases.RoleTTL = "1s"
		c.Leases.SessionTTL = "1h"
		c.Leases.TaskTTL = "1h"
		c.Leases.MaxSilent = "1s"
		c.Leases.SweepInterval = "1s"
	})
	coord := newClient(t, b, "role-pulse-coord")
	worker := newClient(t, b, "role-pulse-worker")
	b.startOptions()
	coord.open(t, "coordinator", "role-pulse-coord")
	task := coord.task(t, coord.epic(t).Key, "architect")
	p := startMCP(t, worker, b.url)
	view := exactMCP[core.SessionView](t, p.call("describe", payload{"request_id": "role-pulse-open", "role": "architect", "run_id": "pulse", "machine": "role-pulse-worker"}))
	worker.session = view.SessionID
	claim := exactMCP[core.Claim](t, p.call("next", payload{"request_id": "role-pulse-next", "task_key": task.Key}))
	if view.RoleExpiresAt == nil || view.HeartbeatEverySeconds > 0.334 {
		t.Fatal("bridge did not receive minimum role TTL")
	}
	b.waitExpired(view.RoleExpiresAt.Add(1500 * time.Millisecond))
	live := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": task.Key}))
	if live.Lease == nil || live.Lease.Generation != claim.Lease.Generation || len(live.Attempts) != 1 {
		t.Fatal("idle bridge failed to keep short singleton while task/session last1h")
	}
	p.stop()
	p.assertProtocol(worker.token)
	db := inspectDatabase(t, b.db)
	stored := decode[core.Session](t, []byte(sqlString(t, db, "SELECT data FROM sessions WHERE key='"+worker.session+"'")))
	if stored.RoleExpiresAt == nil || !stored.RoleExpiresAt.After(*view.RoleExpiresAt) {
		t.Fatal("role deadline did not advance through child heartbeat")
	}
	b.waitExpired(*stored.RoleExpiresAt)
	worker.fails(t, "heartbeat", "LEASE_LOST", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "role_generation": view.RoleGeneration})
	after := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": task.Key}))
	if after.Lease != nil || after.Attempts[0].Reason != "lease_expired" {
		t.Fatal("EOF role expiry did not fence hour-long task")
	}
	t.Log("D056-15 real idle stdio bridge renews roleTTL1s with task/session1h; EOF stops pulse and lost role fences task before task TTL")
}
