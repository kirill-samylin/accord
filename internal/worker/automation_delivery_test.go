package worker

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func managedEcho(phases *[]string, refuse map[string]int) postFunc {
	return func(_ context.Context, _ string, input, output any) error {
		in := input.(core.ManagedReportInput)
		*phases = append(*phases, in.Phase)
		if code := refuse[in.Phase]; code != 0 {
			return statusError{code: code, server: "CONFLICT", message: "Команда уже terminal"}
		}
		status, state := "applied", "stopped"
		switch in.Phase {
		case "recovered_stopped":
			status = "cancelled"
		case "failed":
			status, state = "failed", "failed"
		case "completed":
			state = "completed"
		}
		*output.(*core.ManagedResult) = core.ManagedResult{Command: core.ManagedCommand{ID: in.CommandID, DispatchID: in.DispatchID, RunID: in.RunID, Generation: in.Generation, Status: status}, Run: core.ManagedRun{ID: in.RunID, Generation: in.Generation, State: state}}
		return nil
	}
}

func TestAlreadyTerminalRefusalNeverEndsTheRunLoop(t *testing.T) {
	e := testEngine(t)
	run := &RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1, State: "stopped", CommandID: "command_start"}
	receipt := &Receipt{CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
	e.journal.Runs[run.ID], e.journal.Receipts[receipt.CommandID] = run, receipt
	e.mu.Lock()
	e.reportFor(run.CommandID, "failed", run, "stopped before native turn was observed", "")
	e.reportFor(run.CommandID, "delivery_unknown", run, "late", "")
	e.mu.Unlock()
	if len(e.journal.Pending) != 1 {
		t.Fatal("delivery_unknown was queued after a terminal report")
	}
	var phases []string
	if err := e.sendPendingReports(context.Background(), managedEcho(&phases, map[string]int{"failed": 409}), 8); err != nil {
		t.Fatal("409 for an already terminal command stopped the worker", err)
	}
	if strings.Join(phases, ",") != "failed,recovered_stopped" || len(e.journal.Pending) != 0 {
		t.Fatal("refused terminal report was not re-sent once as recovered_stopped", phases)
	}
	// An obsolete report after a terminal one is dropped on 409.
	e.mu.Lock()
	e.reportFor(run.CommandID, "started", run, "", "")
	e.mu.Unlock()
	phases = nil
	if err := e.sendPendingReports(context.Background(), managedEcho(&phases, map[string]int{"started": 409}), 8); err != nil || len(e.journal.Pending) != 0 {
		t.Fatal("obsolete report after terminal was not dropped", err)
	}
	// Without a local terminal report a 409 still stops the worker.
	other := &Receipt{RequestID: "report_other", CommandID: "command_other", RunID: run.ID, DispatchID: "dispatch_other", Generation: 1, Phase: "accepted"}
	e.journal.Receipts[other.CommandID] = other
	e.journal.Pending = []*Receipt{other}
	if err := e.sendPendingReports(context.Background(), managedEcho(&phases, map[string]int{"accepted": 409}), 8); err == nil {
		t.Fatal("unexplained 409 was silently ignored")
	}
}

func TestLocalDeadlineStopIsTheLiveWorkersTerminalStop(t *testing.T) {
	e, request := automationTestEngine(t)
	run := e.journal.Runs[request.RunID]
	active := e.active[run.ID]
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		*out.(*core.Task) = core.Task{Key: "T-1", Role: "tester", Generation: 3}
		return nil
	}
	e.wg.Add(1)
	go e.observe(run.ID, active.process)
	e.mu.Lock()
	active.automation.deadline = time.Now().Add(-time.Second)
	e.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	e.watchdog(ctx)
	cancel()
	e.wg.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	if run.State != "stopped" || run.LocalDiagnostic != "local_deadline_stop" || active.stopReason != "local_deadline" {
		t.Fatal("deadline stop is not an explicit local stop", run.State, run.LocalDiagnostic)
	}
	if len(e.journal.Pending) != 1 || e.journal.Pending[0].Phase != "recovered_stopped" || !strings.HasPrefix(e.journal.Pending[0].Error, "local_deadline:") {
		t.Fatal("deadline stop must queue exactly one recovered_stopped report with its reason", e.journal.Pending)
	}
	if len(run.Automation.Pending) != 1 || run.Automation.Pending[0].CommandID != run.CommandID {
		t.Fatal("deadline exit proof is not bound to the execution command")
	}
}

func TestShutdownStopsProcessesThenFlushesReportAndExitProof(t *testing.T) {
	e, request := automationTestEngine(t)
	run := e.journal.Runs[request.RunID]
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		*out.(*core.Task) = core.Task{Key: "T-1", Role: "tester", Generation: 3}
		return nil
	}
	e.automationTransport = func(_ context.Context, _ string, input, output any) error {
		in := input.(core.WorkflowAttestInput)
		*output.(*core.WorkflowAttestResult) = core.WorkflowAttestResult{ControlEpoch: in.ControlEpoch, Attestation: core.WorkflowAttestation{Kind: "run_exit", Status: "completed", WorkflowID: in.WorkflowID, StepID: in.StepID, ControlEpoch: in.ControlEpoch, RunID: in.RunID, Generation: in.Generation, TaskKey: in.TaskKey, TaskGeneration: in.TaskGeneration, NativeSessionID: in.NativeSessionID}}
		return nil
	}
	e.wg.Add(1)
	go e.observe(run.ID, e.active[run.ID].process)
	if err := e.StopAll("worker_shutdown: local worker is stopping"); err != nil {
		t.Fatal(err)
	}
	var phases []string
	e.drainOnShutdown(managedEcho(&phases, nil), 5*time.Second)
	e.mu.Lock()
	defer e.mu.Unlock()
	if strings.Join(phases, ",") != "recovered_stopped" || len(e.journal.Pending) != 0 {
		t.Fatal("shutdown did not flush exactly the terminal stop", phases)
	}
	if !run.Automation.ExitAck || len(run.Automation.Pending) != 0 || e.active[run.ID] != nil {
		t.Fatal("shutdown closed before the exit proof was acknowledged")
	}
}

func TestStatusErrorCarriesBoundedServerCodeAndMessage(t *testing.T) {
	got := readStatusError(409, strings.NewReader(`{"error":{"code":"CONFLICT","message":"Команда уже terminal\u0007"}}`))
	if got.code != 409 || got.server != "CONFLICT" || got.message != "Команда уже terminal" {
		t.Fatal("server code/message lost or unsanitized", got)
	}
	if text := got.Error(); !strings.Contains(text, "HTTP 409 CONFLICT: Команда уже terminal") || strings.Contains(text, "check local enrollment") {
		t.Fatal(text)
	}
	if text := readStatusError(401, io.NopCloser(strings.NewReader("<html>secret</html>"))).Error(); !strings.Contains(text, "HTTP 401") || strings.Contains(text, "secret") || !strings.Contains(text, "check local enrollment") {
		t.Fatal(text)
	}
	if long := readStatusError(400, strings.NewReader(`{"error":{"code":"VALIDATION","message":"`+strings.Repeat("x", 5000)+`"}}`)); len(long.message) > 200 {
		t.Fatal("unbounded server message")
	}
}
