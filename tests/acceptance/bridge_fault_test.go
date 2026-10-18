package acceptance

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"agent-board/internal/core"
)

// Faults are injected at the HTTP boundary; both the server and stdio bridge
// remain real CLI processes. This proxy is a test double, not a remote VPS.
func TestMCPProcessFaultQuarantine(t *testing.T) {
	for _, fault := range []string{"network", "revocation"} {
		t.Run(fault, func(t *testing.T) {
			b := newBoard(t, "1s")
			coordinator := newClient(t, b, "coordinator")
			worker := newClient(t, b, "bridge")
			b.start()
			coordinator.open(t, "coordinator", "coordinator")
			task := coordinator.task(t, coordinator.epic(t).Key, "developer")
			target, err := url.Parse(b.url)
			if err != nil {
				t.Fatal(err)
			}
			reverse := httputil.NewSingleHostReverseProxy(target)
			reverse.ErrorLog = log.New(io.Discard, "", 0)
			var failed atomic.Bool
			var calls atomic.Int64
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
				if err != nil {
					http.Error(w, "synthetic read failure", 503)
					return
				}
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewReader(data))
				var frame struct {
					Method string `json:"method"`
				}
				if json.Unmarshal(data, &frame) == nil && frame.Method == "tools/call" {
					calls.Add(1)
				}
				if failed.Load() {
					http.Error(w, "synthetic unavailable service", 503)
					return
				}
				reverse.ServeHTTP(w, r)
			}))
			t.Cleanup(proxy.Close)
			p := startMCP(t, worker, proxy.URL)
			bridgeSession(t, p, worker)
			claim := mcpValue[core.Claim](t, p.call("next", payload{"request_id": "claim", "task_key": task.Key}))
			if fault == "network" {
				failed.Store(true)
				p.uncertain(p.call("comment", payload{"request_id": "uncertain-write", "task_key": task.Key, "generation": claim.Lease.Generation, "text": "must not apply"}))
				failed.Store(false)
				mcpValue[core.Task](t, p.call("get", payload{"task_key": task.Key}))
			} else {
				if revoke := command(nil, "token", "revoke", "--db", b.db, "--id", worker.tokenID); revoke.err != nil {
					t.Fatal("synthetic token revocation failed")
				}
				deadline := time.NewTimer(4 * time.Second)
				defer deadline.Stop()
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for !bytes.Contains(p.stderr.data(), []byte("LEASE_UNCERTAIN")) {
					select {
					case <-deadline.C:
						t.Fatal("background pulse did not detect token revocation")
					case <-ticker.C:
					}
				}
			}
			before := calls.Load()
			for _, operation := range []string{"heartbeat", "describe", "next", "comment", "release"} {
				p.uncertain(p.call(operation, payload{"request_id": "blocked-" + operation}))
			}
			if calls.Load() != before {
				t.Fatal("quarantine forwarded a mutation or attempted to reacquire")
			}
			p.stop()
			p.assertProtocol(worker.token)
			current := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
			if current.Lease != nil {
				b.waitExpired(current.Lease.ExpiresAt)
			}
			current = decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
			if current.Lease != nil || current.Generation != claim.Lease.Generation || len(current.Attempts) != 1 || len(current.Comments) != 0 {
				t.Fatal("fault/quarantine wrote task data or renewed/reacquired stale work")
			}
			t.Log("real stdio child quarantined permanently; blocked mutations never reached HTTP, owned lease expired")
		})
	}
}
