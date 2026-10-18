package bridge

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestNetworkTimeoutQuarantinesWithoutMutationRecovery(t *testing.T) {
	f := setup(t, 100*time.Millisecond)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	// The authoritative service never sees this request; the bridge observes a
	// timed-out HTTP exchange and must not assume it is safe to keep working.
	f.fault.Store(2)
	result := f.invoke("comment", map[string]any{"request_id": "uncertain-comment", "task_key": task.Key, "generation": claim.Lease.Generation, "text": "not applied"})
	raw, err := resultJSON(result)
	if err != nil || !result.IsError || errorCode(raw) != "LEASE_UNCERTAIN" {
		t.Fatal("network timeout did not quarantine")
	}
	f.waitQuarantine()
	f.fault.Store(0)
	value[core.Task](t, f.invoke("get", map[string]string{"task_key": task.Key}))
	before := f.calls.Load()
	for _, name := range []string{"heartbeat", "describe", "next", "comment", "release"} {
		if result = f.invoke(name, map[string]string{"request_id": "no-auto-revival"}); !result.IsError {
			t.Fatalf("quarantine allowed %s", name)
		}
	}
	if f.calls.Load() != before {
		t.Fatal("quarantined mutation made an upstream request")
	}
	time.Sleep(1100 * time.Millisecond)
	current := f.get(task.Key)
	if current.Lease != nil || current.State != "ready" || len(current.Comments) != 0 {
		t.Fatal("quarantine renewed work or applied uncertain mutation")
	}
}

func TestConcurrentReleaseAndHeartbeatNeverRenewReleasedGeneration(t *testing.T) {
	f := setup(t, 10*time.Millisecond)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Read requests and timer contend with release while the owned generation
			// is stable. Authoritative transactions and bridge state are both serial.
			result := f.invoke("get", map[string]string{"task_key": task.Key})
			if result.IsError {
				t.Error("concurrent read failed")
			}
		}(i)
	}
	value[core.Task](t, f.invoke("release", map[string]any{"request_id": "release", "task_key": task.Key, "generation": claim.Lease.Generation, "reason": "handoff"}))
	wg.Wait()
	time.Sleep(1100 * time.Millisecond)
	current := f.get(task.Key)
	if current.Lease != nil || current.State != "ready" {
		t.Fatal("timer renewed released task")
	}
	if f.stderr.String() != "" {
		t.Fatalf("concurrent release caused false quarantine: %s", f.stderr.String())
	}
}

func TestStaleHeartbeatInputCannotChangeTrackedGeneration(t *testing.T) {
	f := setup(t, 30*time.Millisecond)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	result := f.invoke("heartbeat", map[string]any{"request_id": "stale", "task_key": task.Key, "generation": claim.Lease.Generation + 1})
	if !result.IsError {
		t.Fatal("stale generation heartbeat accepted")
	}
	time.Sleep(1100 * time.Millisecond)
	if current := f.get(task.Key); current.Lease == nil || current.Generation != claim.Lease.Generation {
		t.Fatal("bad client input changed canonical generation")
	}
}

func TestHistoricalDescribeCannotRestartExpiredSession(t *testing.T) {
	f := setup(t, 30*time.Millisecond)
	// Persist an already expired describe response with the same request ID the
	// client will supply to this fresh bridge process.
	actor := f.admin()
	opened, err := f.store.Describe(context.Background(), actor.Principal, core.OpenSession{RequestID: "bridge-open", Role: "developer", RunID: "synthetic-run", Machine: "test-machine"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	result := f.invoke("describe", core.OpenSession{RequestID: "bridge-open", Role: "developer", RunID: "synthetic-run", Machine: "test-machine"})
	raw, _ := resultJSON(result)
	if !result.IsError || errorCode(raw) != "LEASE_UNCERTAIN" {
		t.Fatal("historical describe revived session")
	}
	// The caller still receives a protocol result, never an expired identity as
	// a successful new lease. Stored replay is unchanged on the server.
	replay, err := f.store.Describe(context.Background(), actor.Principal, core.OpenSession{RequestID: "bridge-open", Role: "developer", RunID: "synthetic-run", Machine: "test-machine"})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(opened)
	z, _ := json.Marshal(replay)
	if string(a) != string(z) {
		t.Fatal("server historical replay changed")
	}
}

func TestAuthoritativeLeaseLossPermanentlyQuarantines(t *testing.T) {
	f := setup(t, 40*time.Millisecond)
	task := f.task()
	session := f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	principal, err := f.store.Authenticate(context.Background(), f.token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate another client using this same session to release its task. The
	// bridge must discover the loss; it cannot silently obtain a new generation.
	_, err = f.store.Release(context.Background(), core.Actor{Principal: principal, SessionID: session.SessionID}, core.LeaseInput{RequestID: "external-release", TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "test ownership loss"})
	if err != nil {
		t.Fatal(err)
	}
	f.waitQuarantine()
	result := f.invoke("next", map[string]string{"request_id": "must-not-reacquire", "task_key": task.Key})
	if !result.IsError {
		t.Fatal("bridge automatically reacquired lost task")
	}
	value[core.Task](t, f.invoke("get", map[string]string{"task_key": task.Key}))
	if current := f.get(task.Key); current.Lease != nil || current.Generation != claim.Lease.Generation {
		t.Fatal("diagnostic read revived ownership")
	}
}
