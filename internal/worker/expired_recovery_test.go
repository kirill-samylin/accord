package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestExpiredRecoveryPrecedesPollAndRetriesExactCurrentGeneration(t *testing.T) {
	e := testEngine(t)
	e.journal.HostID = "host_test"
	e.journal.Runs["run_test"] = &RunState{ID: "run_test", Workspace: "test", Generation: 2, State: "stopped", SessionID: syntheticSession, CommandID: "command_resume"}
	current := Receipt{RequestID: "accepted_original", CommandID: "command_resume", RunID: "run_test", DispatchID: "dispatch_resume", Generation: 2, Phase: "accepted"}
	e.journal.Pending = []*Receipt{{RequestID: "older_generation", CommandID: "command_start", RunID: "run_test", Generation: 1, Phase: "completed"}, &current}
	latest := current
	latest.Phase, latest.Forced = "recovered_stopped", true
	e.journal.Receipts[current.CommandID] = &latest
	e.plans[current.CommandID] = plan{command: core.ManagedCommand{Kind: "resume"}}
	launches := 0
	e.start = func(launchSpec) (process, error) { launches++; return successfulFake(), nil }
	expired := time.Now().Add(-time.Second)
	if err := e.SetLease(7, expired); err != nil {
		t.Fatal(err)
	}
	var requests []core.ManagedReportInput
	sessions := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/session"):
			sessions++
			json.NewEncoder(w).Encode(core.SessionView{SessionID: "fresh_human", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/managed/report"):
			var input core.ManagedReportInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			requests = append(requests, input)
			if len(requests) == 1 {
				http.Error(w, "synthetic lost recovery response", 500)
				return
			}
			json.NewEncoder(w).Encode(core.ManagedResult{})
		default:
			t.Error("ownership poll or execution was attempted before recovery")
			w.WriteHeader(409)
		}
	}))
	defer server.Close()
	e.config.Server = server.URL
	r, err := newRemote(e.config)
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := e.recoverExpiredReports(context.Background(), r)
	if err != nil || !blocked || len(e.journal.Pending) != 1 {
		t.Fatal("transient recovery must remain pending without polling or exiting")
	}
	blocked, err = e.recoverExpiredReports(context.Background(), r)
	if err != nil || blocked || len(e.journal.Pending) != 0 {
		t.Fatal("confirmed recovery did not settle the pending report")
	}
	if len(requests) != 2 || !reflect.DeepEqual(requests[0], requests[1]) {
		t.Fatal("lost recovery response changed the retry payload")
	}
	input := requests[0]
	if input.Phase != "recovered_stopped" || input.Generation != 2 || input.CommandID != current.CommandID || input.ProviderSessionID != syntheticSession || !input.Forced {
		t.Fatal("recovery did not preserve current execution identity and observed exit")
	}
	if launches != 0 || len(e.plans) != 0 || !e.deadline.Equal(expired) || sessions != 1 {
		t.Fatal("recovery revived authority, replayed a launch, or reopened a live human session")
	}
}

func TestExpiredRecoveryKeepsUncertainProcessesQuarantined(t *testing.T) {
	e := testEngine(t)
	e.journal.Runs["run_test"] = &RunState{ID: "run_test", Generation: 1, State: "needs_recovery", CommandID: "command_start"}
	receipt := &Receipt{CommandID: "command_start", RunID: "run_test", DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
	e.journal.Receipts[receipt.CommandID] = receipt
	e.journal.Pending = []*Receipt{receipt}
	if err := e.SetLease(1, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	phase := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input core.ManagedReportInput
		json.NewDecoder(r.Body).Decode(&input)
		phase = input.Phase
		json.NewEncoder(w).Encode(core.ManagedResult{})
	}))
	defer server.Close()
	e.config.Server = server.URL
	r, err := newRemote(e.config)
	if err != nil {
		t.Fatal(err)
	}
	r.session.ExpiresAt = time.Now().Add(time.Minute)
	blocked, err := e.recoverExpiredReports(context.Background(), r)
	if blocked || err != nil || phase != "delivery_unknown" || e.journal.Runs["run_test"].State != "needs_recovery" || !e.journal.Runs["run_test"].RecoveryNotified {
		t.Fatal("uncertain ownership was reported as a confirmed stop or stopped serving other work", blocked, err)
	}
	// The run stays quarantined locally: its workspace refuses a new launch
	// until the owner runs `worker recover`.
	e.journal.Runs["run_test"].Workspace = "test"
	e.journal.Runs["run_test"].Provider = "codex"
	command, managed := testDispatch("start")
	managed.ID, command.RunID, command.ID, command.DispatchID = "run_other", "run_other", "command_other", "dispatch_other"
	if err = e.Accept(command, managed); err != nil {
		t.Fatal("a quarantined workspace stopped the worker instead of refusing one start", err)
	}
	refused := e.journal.Runs["run_other"]
	if refused.State != "failed" || refused.LocalDiagnostic != "workspace_quarantined" || len(e.journal.Pending) != 2 || e.journal.Pending[0].Phase != "accepted" || e.journal.Pending[1].Phase != "failed" || len(e.plans) != 0 {
		t.Fatal("quarantined workspace launch was not refused as accepted+failed", refused.State, e.journal.Pending)
	}
	command, _ = testDispatch("stop")
	if err = e.Accept(command, core.ManagedRun{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1}); !errors.Is(err, errRunQuarantined) {
		t.Fatal("a dispatch for the needs_recovery run is not marked quarantined", err)
	}
}

func TestRecoveryRequestChangesOnlyForNewWorkerIncarnation(t *testing.T) {
	e := testEngine(t)
	e.journal.Runs["run_test"] = &RunState{ID: "run_test", Generation: 1, State: "stopped"}
	e.journal.Pending = []*Receipt{{CommandID: "command_start", RunID: "run_test", Generation: 1}}
	if err := e.prepareReconciliation(); err != nil {
		t.Fatal(err)
	}
	first := *e.Pending()
	e.journal.WorkerID = "worker_new_incarnation"
	if err := e.prepareReconciliation(); err != nil {
		t.Fatal(err)
	}
	next := e.Pending()
	if next.RequestID == first.RequestID || next.RecoveryWorkerID != e.journal.WorkerID || next.Phase != first.Phase {
		t.Fatal("new process reused a request bound to another worker incarnation")
	}
}
