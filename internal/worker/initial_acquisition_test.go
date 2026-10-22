package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestInitialAcquisitionWaitsOnlyWithoutPriorAuthority(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []int
		polls    int
		acquired bool
		local    string
	}{
		{"previous_lease_expires", []int{409, 409, 200, 401}, 4, true, ""},
		{"live_previous_worker", []int{409}, -1, false, ""},
		{"conflict_after_acquisition", []int{200, 409}, 2, true, ""},
		{"invalid_credential", []int{401}, 1, false, ""},
		{"revoked_host", []int{403}, 1, false, ""},
		{"unconfirmed_accepted", []int{409}, 1, false, "accepted"},
		{"ambiguous_process", []int{409}, 0, false, "needs_recovery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testEngine(t)
			e.deadline, e.journal.Epoch, e.journal.HostID = time.Time{}, 0, "host_test"
			if tc.local != "" {
				run := &RunState{ID: "run_test", Generation: 1, State: tc.local, CommandID: "command_start"}
				receipt := &Receipt{CommandID: run.CommandID, RunID: run.ID, Generation: 1, DispatchID: "dispatch_start", Phase: "accepted"}
				e.journal.Runs[run.ID], e.journal.Receipts[receipt.CommandID] = run, receipt
				e.journal.Pending = []*Receipt{receipt}
			}
			var mu sync.Mutex
			var ids []string
			launches := 0
			e.start = func(launchSpec) (process, error) { launches++; return successfulFake(), nil }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/session"):
					json.NewEncoder(w).Encode(core.SessionView{SessionID: "human_test", ExpiresAt: time.Now().Add(time.Minute)})
				case strings.HasSuffix(r.URL.Path, "/managed/report"):
					var report core.ManagedReportInput
					json.NewDecoder(r.Body).Decode(&report)
					if report.Phase != "delivery_unknown" {
						t.Error("unconfirmed local execution was falsely reported stopped")
					}
					json.NewEncoder(w).Encode(core.ManagedResult{})
				case strings.HasSuffix(r.URL.Path, "/managed/poll"):
					var poll core.ManagedPollInput
					json.NewDecoder(r.Body).Decode(&poll)
					mu.Lock()
					ids = append(ids, poll.RequestID)
					index := min(len(ids)-1, len(tc.statuses)-1)
					status := tc.statuses[index]
					mu.Unlock()
					if status != 200 {
						w.WriteHeader(status)
						return
					}
					json.NewEncoder(w).Encode(core.ManagedPollResult{WorkerEpoch: 9, LeaseExpiresAt: time.Now().Add(30 * time.Second)})
				default:
					t.Error("unexpected request while waiting for initial ownership")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			e.config.Server = server.URL
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			started := time.Now()
			err := e.run(ctx, 10*time.Millisecond, 80*time.Millisecond)
			elapsed := time.Since(started)
			if err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("acquisition did not terminate under its own bounded authority rules")
			}
			mu.Lock()
			defer mu.Unlock()
			if tc.polls >= 0 && len(ids) != tc.polls {
				t.Fatalf("wrong ownership polling count: got %d want %d", len(ids), tc.polls)
			}
			if tc.polls < 0 && (len(ids) < 2 || elapsed < 80*time.Millisecond || elapsed > time.Second) {
				t.Fatal("live previous owner was not respected for the bounded grace window")
			}
			seen := map[string]bool{}
			for _, id := range ids {
				if id == "" || seen[id] {
					t.Fatal("initial poll retry reused a cached ownership request")
				}
				seen[id] = true
			}
			if launches != 0 || e.deadline.IsZero() == tc.acquired || (e.journal.Epoch == 9) != tc.acquired {
				t.Fatal("waiting or recovery fabricated execution or ownership")
			}
		})
	}
}
