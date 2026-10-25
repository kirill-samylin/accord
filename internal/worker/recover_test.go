package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestWorkerRecoverVerifiesNoOwnedProcessThenReportsTerminalStop(t *testing.T) {
	e := testEngine(t)
	if _, err := Open(e.config); err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatal("a second process could take the private state while a worker holds it", err)
	}
	e.journal.HostID = "host_test"
	run := &RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1, State: "needs_recovery", CommandID: "command_start", ProcessGroups: []int{4242}}
	e.journal.Runs[run.ID] = run
	e.journal.Receipts[run.CommandID] = &Receipt{CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
	e.journal.Pending = []*Receipt{{RequestID: "old", CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "delivery_unknown"}}
	lock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "worker_old run_test")
	if err != nil {
		t.Fatal(err)
	}
	if err = lock.release(false); err != nil { // A crashed worker leaves its intent.
		t.Fatal(err)
	}
	e.groupCensus = func(groups []int) ([]int, error) { return groups, nil }
	if err = e.Recover(context.Background(), run.ID); err == nil || run.State != "needs_recovery" {
		t.Fatal("recovery ignored a live owned process group", err)
	}
	var phases []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/session"):
			_ = json.NewEncoder(w).Encode(core.SessionView{SessionID: "human_test", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/managed/report"):
			var in core.ManagedReportInput
			_ = json.NewDecoder(r.Body).Decode(&in)
			phases = append(phases, in.Phase)
			if in.WorkerID != e.journal.WorkerID || in.CommandID != run.CommandID {
				t.Error("recovery report is not bound to this incarnation and command")
			}
			_ = json.NewEncoder(w).Encode(core.ManagedResult{})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	e.config.Server = server.URL
	e.groupCensus = func([]int) ([]int, error) { return nil, nil }
	if err = e.Recover(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	if run.State != "stopped" || run.LocalDiagnostic != "owner_recovered" || len(e.journal.Pending) != 0 || !run.RecoveryNotified || strings.Join(phases, ",") != "recovered_stopped" {
		t.Fatal("recover did not stop the run and report recovered_stopped once", run.State, phases)
	}
	next, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "worker_new run_next")
	if err != nil {
		t.Fatal("recovered checkout still blocked", err)
	}
	_ = next.release(true)
	if err = e.Recover(context.Background(), run.ID); err == nil {
		t.Fatal("recover accepted a run that is not awaiting recovery")
	}
}
