package acceptance

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"agent-board/internal/core"
)

func bridgeSession(t *testing.T, p *mcpProcess, c *client) core.SessionView {
	t.Helper()
	session := mcpValue[core.SessionView](t, p.call("describe", payload{"request_id": "bridge-open", "role": "developer", "run_id": "synthetic-bridge", "machine": "bridge"}))
	c.session = session.SessionID
	return session
}

func TestMCPProcessHeartbeatAndEOF(t *testing.T) {
	b := newBoard(t, "1s")
	coordinator := newClient(t, b, "coordinator")
	worker := newClient(t, b, "bridge")
	successor := newClient(t, b, "successor")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	successor.open(t, "developer", "successor")
	task := coordinator.task(t, coordinator.epic(t).Key, "developer")
	p := startMCP(t, worker, b.url)
	bridgeSession(t, p, worker)
	claim := mcpValue[core.Claim](t, p.call("next", payload{"request_id": "bridge-claim", "task_key": task.Key}))
	if claim.Lease == nil {
		t.Fatal("stdio worker did not claim task")
	}
	// Poll the authoritative HTTP clock until >2 initial task TTL have passed.
	// The model is idle: only the child bridge can maintain this lease.
	b.waitExpired(claim.Lease.ExpiresAt.Add(1500 * time.Millisecond))
	live := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if live.Lease == nil || live.Lease.Generation != claim.Lease.Generation || !live.Lease.ExpiresAt.After(claim.Lease.ExpiresAt) || len(live.Attempts) != 1 {
		t.Fatal("stdio child failed to maintain the same task while model was idle")
	}
	if len(p.stderr.data()) != 0 {
		t.Fatal("healthy stdio child unexpectedly reported a diagnostic")
	}
	p.stop()
	p.assertProtocol(worker.token)
	afterEOF := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if afterEOF.Lease != nil {
		b.waitExpired(afterEOF.Lease.ExpiresAt)
	}
	fresh := successor.claim(t, task.Key)
	if fresh.Lease.Generation != claim.Lease.Generation+1 || len(fresh.Task.Attempts) != 2 || fresh.Task.Attempts[0].Reason != "lease_expired" {
		t.Fatal("EOF did not stop renewal or successor reused stale fencing")
	}
	worker.fails(t, "comment", "LEASE_LOST", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "text": "stale child"})
	t.Log("real stdio child held generation >2TTL; EOF exited cleanly, natural expiry fenced the next CLI process")
}

func TestMCPProcessReplayAndServerRestart(t *testing.T) {
	b := newBoard(t, "1s")
	coordinator := newClient(t, b, "coordinator")
	worker := newClient(t, b, "bridge")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	task := coordinator.task(t, coordinator.epic(t).Key, "developer")
	p := startMCP(t, worker, b.url)
	bridgeSession(t, p, worker)
	input := payload{"request_id": "historical-next", "task_key": task.Key}
	first := p.call("next", input)
	claim := mcpValue[core.Claim](t, first)
	mcpValue[core.Task](t, p.call("release", payload{"request_id": "released", "task_key": task.Key, "generation": claim.Lease.Generation, "reason": "synthetic handoff", "branch": "synthetic/bridge"}))
	replay := p.call("next", input)
	if replay.IsError || !bytes.Equal(mcpJSON(t, first), mcpJSON(t, replay)) {
		t.Fatal("exact MCP retry did not return saved response")
	}
	b.waitExpired(claim.Lease.ExpiresAt.Add(time.Second))
	current := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if current.Lease != nil || current.State != "ready" || len(current.Attempts) != 1 || current.Attempts[0].Reason != "synthetic handoff" {
		t.Fatal("historical next replay secretly reacquired or renewed released ownership")
	}
	p.stop()
	p.assertProtocol(worker.token)
	before := decode[core.EventsPage](t, coordinator.ok(t, "events", payload{"since_cursor": 0, "limit": 200}))
	b.stop()
	b.start()
	input["session_id"] = worker.session
	saved := worker.raw("next", worker.prepare(t, input))
	if saved.err != nil || !reflect.DeepEqual(decode[core.Claim](t, mcpJSON(t, first)), decode[core.Claim](t, saved.data)) {
		t.Fatal("saved MCP operation response did not survive CLI server restart")
	}
	after := decode[core.EventsPage](t, coordinator.ok(t, "events", payload{"since_cursor": before.NextCursor, "limit": 200}))
	if len(after.Events) != 0 || after.HighWater != before.HighWater {
		t.Fatal("restart/retry duplicated events or renewed historical ownership")
	}
	current = decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	if current.Lease != nil || current.Generation != claim.Lease.Generation {
		t.Fatal("restart/retry restored a released lease")
	}
	t.Log("MCP next replay after release is historical; real server restart preserves response without events or reacquire")
}
