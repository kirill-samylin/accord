package acceptance

import (
	"bytes"
	"testing"

	"agent-board/internal/core"
)

func TestCLIRetryAndFilteredPagination(t *testing.T) {
	b := newBoard(t, "30s")
	c := newClient(t, b, "coordinator")
	b.start()
	c.open(t, "coordinator", "coordinator")
	epic := c.epic(t)
	input := c.prepare(t, payload{"request_id": "uncertain-create", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Synthetic retry", "description": "Lost response simulation"})
	// The first complete response is intentionally discarded by the harness.
	// This simulates response uncertainty, not a TCP fault or interrupted commit.
	if first := c.raw("create", input); first.err != nil {
		t.Fatal("initial create failed before response-discard simulation")
	}
	retry := c.raw("create", input)
	if retry.err != nil {
		t.Fatal("identical create retry failed")
	}
	created := decode[core.Task](t, retry.data)
	secondRetry := c.raw("create", input)
	if secondRetry.err != nil || !bytes.Equal(retry.data, secondRetry.data) {
		t.Fatal("repeated create did not return identical saved response")
	}
	c.fails(t, "create", "IDEMPOTENCY_CONFLICT", payload{"request_id": "uncertain-create", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Changed payload", "description": "Lost response simulation"})
	extra := c.task(t, epic.Key, "analyst")
	expected := map[string]bool{epic.Key: true, created.Key: true, extra.Key: true}
	seen := map[string]bool{}
	var cursor int64
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("filtered event pagination did not terminate")
		}
		page := decode[core.EventsPage](t, c.ok(t, "events", payload{"since_cursor": cursor, "limit": 1, "types": []string{"task_created"}}))
		if page.NextCursor < cursor || page.NextCursor > page.HighWater {
			t.Fatal("filtered cursor is outside its valid range")
		}
		for _, event := range page.Events {
			if event.Type != "task_created" || seen[event.TaskKey] || !expected[event.TaskKey] {
				t.Fatal("missing filter enforcement, duplicate create event or unexpected task")
			}
			seen[event.TaskKey] = true
		}
		if pages == 0 {
			appended := c.task(t, epic.Key, "human")
			expected[appended.Key] = true
		}
		cursor = page.NextCursor
		if !page.HasMore {
			break
		}
	}
	if len(seen) != len(expected) {
		t.Fatal("pagination skipped a matching task_created event")
	}
	filtered := decode[core.EventsPage](t, c.ok(t, "events", payload{"since_cursor": 0, "limit": 1, "types": []string{"task_created"}, "task_key": created.Key}))
	if len(filtered.Events) != 1 || filtered.Events[0].TaskKey != created.Key || filtered.HasMore {
		t.Fatal("retry produced duplicate persisted create events or task filtering failed")
	}
	empty := decode[core.EventsPage](t, c.ok(t, "events", payload{"since_cursor": 0, "limit": 1, "types": []string{"absent_synthetic_type"}}))
	if len(empty.Events) != 0 || empty.HasMore || empty.NextCursor != empty.HighWater {
		t.Fatal("empty filtered scan did not advance to high_water")
	}
	t.Log("discarded-first-response simulation; identical retry, conflict detection, one create event; filtered pages include concurrent append")
}

func TestCLIRestartAndRevocation(t *testing.T) {
	b := newBoard(t, "30s")
	c := newClient(t, b, "coordinator")
	developer := newClient(t, b, "developer")
	b.start()
	open := c.prepare(t, payload{"request_id": "persistent-session", "role": "coordinator", "run_id": "coordinator", "machine": "coordinator"})
	r := c.raw("describe", open)
	if r.err != nil {
		t.Fatal("describe before restart failed")
	}
	c.session = decode[core.SessionView](t, r.data).SessionID
	developer.open(t, "developer", "developer")
	epic := c.epic(t)
	create := c.prepare(t, payload{"request_id": "persistent-create", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Persisted slice", "description": "Restart evidence"})
	first := c.raw("create", create)
	if first.err != nil {
		t.Fatal("create before restart failed")
	}
	task := decode[core.Task](t, first.data)
	claim := developer.claim(t, task.Key)
	developer.ok(t, "release", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "reason": "Persist handoff", "branch": "synthetic/persistent"})
	before := decode[core.EventsPage](t, c.ok(t, "events", payload{"since_cursor": 0, "limit": 200}))
	b.stop()
	b.start()
	sessionReplay := c.raw("describe", open)
	if sessionReplay.err != nil || !bytes.Equal(sessionReplay.data, r.data) {
		t.Fatal("describe replay did not persist the board session across restart")
	}
	replay := c.raw("create", create)
	if replay.err != nil || !bytes.Equal(replay.data, first.data) {
		t.Fatal("mutation replay did not persist across restart")
	}
	after := decode[core.EventsPage](t, c.ok(t, "events", payload{"since_cursor": before.NextCursor, "limit": 200}))
	if len(after.Events) != 0 || after.HighWater != before.HighWater {
		t.Fatal("restart or saved replay added duplicate domain events")
	}
	read := decode[core.Task](t, c.ok(t, "get", payload{"task_key": task.Key}))
	if read.State != "ready" || read.Generation != claim.Lease.Generation || len(read.Attempts) != 1 || read.Attempts[0].Branch != "synthetic/persistent" {
		t.Fatal("task state, generation or historical attempt was lost after restart")
	}
	fresh := developer.claim(t, task.Key)
	if fresh.Lease.Generation != claim.Lease.Generation+1 {
		t.Fatal("lease generation reset after restart")
	}
	if revoke := command(nil, "token", "revoke", "--db", b.db, "--id", c.tokenID); revoke.err != nil {
		t.Fatal("local CLI token revocation failed")
	}
	c.fails(t, "create", "UNAUTHORIZED", payload{"request_id": "persistent-create", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Persisted slice", "description": "Restart evidence"})
	c.fails(t, "create", "UNAUTHORIZED", payload{"type": "epic", "title": "After revocation", "description": "Must fail"})
	readInput := []byte(`{"session_id":"` + c.session + `","task_key":"` + task.Key + `"}`)
	denied := c.raw("get", readInput)
	if denied.err == nil || domainCode(t, denied.data) != "UNAUTHORIZED" {
		t.Fatal("revoked token still permits reads")
	}
	t.Log("SIGTERM exit and same SQLite restart; state, session, replay, events and generation survive; revocation blocks replay, mutation and read")
}
