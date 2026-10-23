package worker

import (
	"agent-board/internal/core"
	"context"
	"io"
	"strings"
	"testing"
)

func TestAutomationStopCoversNativeAndEveryOwnedPreset(t *testing.T) {
	e, request := automationTestEngine(t)
	root := e.active[request.RunID].process.(*fakeProcess)
	check := &fakeProcess{output: io.NopCloser(strings.NewReader("")), done: make(chan struct{})}
	e.active[request.RunID].automation.checks["verification"] = check
	if err := e.StopAll("explicit local stop"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*fakeProcess{root, check} {
		select {
		case <-p.done:
		default:
			t.Fatal("an owned scope survived StopAll")
		}
	}
}
func TestRunExitWaitsExactTerminalCommandAndRebindsPendingStop(t *testing.T) {
	e, request := automationTestEngine(t)
	r := e.journal.Runs[request.RunID]
	a := r.Automation
	receipt := Receipt{RequestID: "terminal", CommandID: r.CommandID, DispatchID: "dispatch_start", RunID: r.ID, Generation: r.Generation, Phase: "completed"}
	a.Pending = []*core.WorkflowAttestInput{{ManagedPollInput: core.ManagedPollInput{RequestID: "exit"}, CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: r.ID, Generation: r.Generation, Kind: "run_exit"}}
	requests := 0
	e.automationTransport = func(context.Context, string, any, any) error { requests++; return nil }
	if err := e.flushAutomation(context.Background()); err != nil || requests != 0 {
		t.Fatal("exit proof escaped before owned terminal ACK", err)
	}
	stopping := receipt
	stopping.State = "stopping"
	if automationExitMatches(a, stopping) {
		t.Fatal("completed report while stopping was treated as terminal")
	}
	matching := receipt
	matching.State = "completed"
	if !automationExitMatches(a, matching) {
		t.Fatal("exact completed receipt rejected")
	}
	foreign := matching
	foreign.CommandID = "foreign"
	if automationExitMatches(a, foreign) {
		t.Fatal("unrelated terminal command authenticated this exit")
	}
	stop := &Receipt{CommandID: "command_stop", DispatchID: "dispatch_stop", RunID: r.ID, Generation: r.Generation}
	oldID := a.Pending[0].RequestID
	e.bindAutomationStopProofLocked(r, stop)
	if a.Pending[0].CommandID != stop.CommandID || a.Pending[0].DispatchID != stop.DispatchID || a.Pending[0].RequestID == oldID {
		t.Fatal("pending proof was not bound to the exact owned stop")
	}
	e.journal.Pending = []*Receipt{&receipt}
	wrong := core.ManagedResult{Run: core.ManagedRun{ID: r.ID, Generation: r.Generation, State: "completed"}, Command: core.ManagedCommand{ID: "foreign", RunID: r.ID, Generation: r.Generation, DispatchID: receipt.DispatchID, Status: "applied"}}
	if e.AckReport(receipt, wrong) == nil {
		t.Fatal("foreign response consumed the durable terminal receipt")
	}
	if e.Pending() == nil {
		t.Fatal("invalid ACK erased recovery facts")
	}
}
