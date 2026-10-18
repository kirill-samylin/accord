package bridge

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestLongOperationMaintainsLeaseAndReplayDoesNotReacquire(t *testing.T) {
	f := setup(t, time.Hour)
	task := f.task()
	session := f.describe()
	if session.SessionTTLSeconds != 1 || session.TaskTTLSeconds != 1 || session.HeartbeatEverySeconds <= 0 {
		t.Fatal("fractional server heartbeat contract missing")
	}
	next := map[string]any{"request_id": "claim-once", "task_key": task.Key}
	claimed := value[core.Claim](t, f.invoke("next", next))
	time.Sleep(2200 * time.Millisecond)
	got := f.get(task.Key)
	if got.Lease == nil || got.Generation != claimed.Lease.Generation || !got.Lease.ExpiresAt.After(time.Now()) {
		t.Fatal("active bridge did not maintain lease beyond 2 TTL")
	}
	release := map[string]any{"request_id": "release-once", "task_key": task.Key, "generation": got.Generation, "reason": "test handoff"}
	value[core.Task](t, f.invoke("release", release))
	historical := value[core.Claim](t, f.invoke("next", next))
	if historical.Lease.Generation != got.Generation {
		t.Fatal("server replay changed")
	}
	time.Sleep(1100 * time.Millisecond)
	if current := f.get(task.Key); current.Lease != nil || current.State != "ready" {
		t.Fatal("historical next replay revived lease")
	}
	value[core.Claim](t, f.invoke("next", map[string]any{"request_id": "new-claim", "task_key": task.Key}))
	// The old successful release must not clear the newly claimed generation.
	value[core.Task](t, f.invoke("release", release))
	time.Sleep(1100 * time.Millisecond)
	current := f.get(task.Key)
	if current.Lease == nil || current.Generation != got.Generation+1 {
		t.Fatal("historical release cleared current generation tracking")
	}
	if result := f.invoke("describe", core.OpenSession{RequestID: "new-open", Role: "analyst", RunID: "other", Machine: "test-machine"}); !result.IsError {
		t.Fatal("bridge opened a second hidden session")
	}
	value[core.SessionView](t, f.invoke("describe", core.OpenSession{RequestID: "bridge-open", Role: "developer", RunID: "synthetic-run", Machine: "test-machine"}))
	if result := f.invoke("get", map[string]any{"session_id": "foreign-session", "task_key": task.Key}); !result.IsError {
		t.Fatal("foreign board session accepted")
	}
}

func TestEOFStopsHeartbeatWithoutEarlyRelease(t *testing.T) {
	f := setup(t, 40*time.Millisecond)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	if err := f.client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-f.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("EOF did not stop bridge")
	}
	immediate := f.get(task.Key)
	if immediate.Lease == nil {
		t.Fatal("EOF implicitly released work before expiry")
	}
	time.Sleep(1100 * time.Millisecond)
	expired := f.get(task.Key)
	if expired.Lease != nil || expired.State != "ready" || len(expired.Attempts) != 1 || expired.Attempts[0].Reason != "lease_expired" {
		t.Fatal("lease did not naturally expire after EOF")
	}
	actor := f.admin()
	session, err := f.store.Describe(context.Background(), actor.Principal, core.OpenSession{RequestID: "replacement", Role: "developer", RunID: "replacement", Machine: "test-machine"})
	if err != nil {
		t.Fatal(err)
	}
	next, err := f.store.Next(context.Background(), core.Actor{Principal: actor.Principal, SessionID: session.SessionID}, core.NextInput{RequestID: "replacement-next", TaskKey: task.Key})
	if err != nil || next.Lease == nil || next.Lease.Generation != claim.Lease.Generation+1 {
		t.Fatalf("replacement generation: %v", err)
	}
}

func TestRevocationQuarantinesAndNeverLeaksToken(t *testing.T) {
	f := setup(t, 40*time.Millisecond)
	task := f.task()
	f.describe()
	value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	if err := f.store.RevokeToken(context.Background(), f.token.ID); err != nil {
		t.Fatal(err)
	}
	f.waitQuarantine()
	before := f.calls.Load()
	result := f.invoke("create", map[string]string{"request_id": "forbidden-after-revoke", "type": "epic", "title": "must never be sent"})
	raw, err := resultJSON(result)
	if err != nil || !result.IsError || errorCode(raw) != "LEASE_UNCERTAIN" {
		t.Fatal("missing fail-closed error")
	}
	if f.calls.Load() != before {
		t.Fatal("quarantined mutation reached network")
	}
	if !strings.Contains(f.stderr.String(), "stop local edits") || strings.Contains(f.stderr.String(), f.token.Secret) {
		t.Fatal("missing stop guidance or secret leaked")
	}
}

func TestToolCatalogueMirrorsSessionOptionalSchema(t *testing.T) {
	f := setup(t, 0)
	tools, err := f.client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 11 {
		t.Fatal("upstream tools missing")
	}
	for _, tool := range tools.Tools {
		data, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Required []string `json:"required"`
		}
		if json.Unmarshal(data, &schema) != nil {
			t.Fatal("bad schema")
		}
		for _, name := range schema.Required {
			if name == "session_id" {
				t.Fatal("injected session is still required")
			}
		}
	}
	// Requests without describe do not open a session implicitly.
	result := f.invoke("events", map[string]int{"since_cursor": 0, "limit": 1})
	if !result.IsError {
		t.Fatal("read opened a session implicitly")
	}
}

func TestStdoutContainsOnlyProtocolAndNoCredential(t *testing.T) {
	f := setup(t, 0)
	f.describe()
	output := f.stdout.String()
	if strings.Contains(output, f.token.Secret) {
		t.Fatal("credential leaked to stdout")
	}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var frame struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if json.Unmarshal([]byte(line), &frame) != nil || frame.JSONRPC != "2.0" {
			t.Fatal("non-protocol stdout output")
		}
	}
	if f.stderr.String() != "" {
		t.Fatal("healthy bridge emitted diagnostics")
	}
}
