package worker

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomationRealHTTPPausedTerminalACKReleasesCheckoutBeforeExactResume(t *testing.T) {
	f := newAutomationHTTPFixture(t)
	w, err := f.s.WorkflowSubmit(f.ctx, f.a, core.WorkflowSubmitInput{RequestID: f.id(), PolicyID: f.p.ID, PolicyVersion: f.p.Version, Text: "Isolated stop and exact resume regression"})
	f.check(err)
	f.tick(w.Workflow.ID)
	first := f.poll(0)
	f.report(first, "accepted", "")
	binding := first.Run.Automation
	secret := core.ExecutionTokenPrefix + strings.Repeat("c", 64)
	digest := sha256.Sum256([]byte(secret))
	_, err = f.s.ExecutionCapabilityRegister(f.ctx, f.a, core.ExecutionCapabilityRegisterInput{ManagedPollInput: core.ManagedPollInput{RequestID: f.id(), HostID: f.h.ID, HostSecret: strings.Repeat("a", 64), WorkerID: "http-worker", WorkerEpoch: first.WorkerEpoch}, CommandID: first.Command.ID, DispatchID: first.Command.DispatchID, RunID: first.Run.ID, Generation: first.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, CapabilityID: "ec_" + strings.Repeat("d", 32), Digest: hex.EncodeToString(digest[:])})
	f.check(err)
	identity, err := f.s.AuthenticateExecutionCapability(f.ctx, secret)
	f.check(err)
	childctx := store.WithExecutionCapability(f.ctx, identity)
	session, err := f.s.Describe(childctx, f.a.Principal, core.OpenSession{RequestID: f.id(), Role: "analyst", RunID: first.Run.ID, Machine: "fixture", ManagedRunID: first.Run.ID, ManagedGeneration: first.Run.Generation, ProfileID: first.Run.Profile.ID, ProfileVersion: first.Run.Profile.Version})
	f.check(err)
	child := core.Actor{Principal: f.a.Principal, SessionID: session.SessionID}
	claim, err := f.s.Next(childctx, child, core.NextInput{RequestID: f.id(), TaskKey: first.Run.TaskKey})
	f.check(err)
	f.report(first, "started", syntheticSession)
	page := f.page(w.Workflow.ID)
	_, err = f.s.WorkflowCommand(f.ctx, f.a, core.WorkflowCommandInput{RequestID: f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, Kind: "pause"})
	f.check(err)
	stop := f.poll(first.WorkerEpoch)
	if stop.Command == nil || stop.Command.Kind != "stop" {
		t.Fatal("pause did not dispatch an owned stop")
	}
	f.report(stop, "accepted", "")
	terminal := f.report(stop, "stopped", "")
	input := core.WorkflowAttestInput{ManagedPollInput: core.ManagedPollInput{RequestID: f.id(), HostID: f.h.ID, HostSecret: strings.Repeat("a", 64), WorkerID: "http-worker", WorkerEpoch: first.WorkerEpoch}, CommandID: stop.Command.ID, DispatchID: stop.Command.DispatchID, RunID: first.Run.ID, Generation: first.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, TaskKey: first.Run.TaskKey, TaskGeneration: claim.Lease.Generation, NativeSessionID: syntheticSession, Kind: "run_exit"}
	responses := make(chan core.WorkflowAttestResult, 1)
	server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/p/p_00000000000000000000000000000001/automation/attest" || r.Header.Get("X-Session") != f.a.SessionID {
			http.Error(out, "binding", 400)
			return
		}
		principal, e := f.s.Authenticate(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if e != nil || principal.ID != f.a.Principal.ID {
			http.Error(out, "unauthorized", 401)
			return
		}
		var in core.WorkflowAttestInput
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(out, "json", 400)
			return
		}
		value, e := f.s.WorkflowAttest(f.ctx, f.a, in)
		if e != nil {
			http.Error(out, e.Error(), 409)
			return
		}
		responses <- value
		out.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(out).Encode(value)
	}))
	defer server.Close()
	e, _ := automationTestEngine(t)
	tokenPath := filepath.Join(e.config.StateDir, "http.token")
	f.check(os.WriteFile(tokenPath, []byte(f.token.Secret), 0600))
	rpc, err := newRemote(Config{Server: server.URL, Project: "p_00000000000000000000000000000001", TokenFile: tokenPath})
	f.check(err)
	rpc.session = core.SessionView{SessionID: f.a.SessionID}
	e.automationTransport = rpc.post
	// The native process is already proven exited; exercise durable terminal
	// ACK, the actual HTTP result, and checkout release rather than spawning CLI.
	e.active = map[string]*activeRun{}
	e.journal.Runs = map[string]*RunState{}
	lock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "http fixture")
	f.check(err)
	run := &RunState{ID: first.Run.ID, Provider: "codex", Workspace: "test", Generation: first.Run.Generation, CommandID: stop.Command.ID, SessionID: syntheticSession, State: "stopped", Automation: &automationAttempt{Run: *first.Run, NativeExited: true, ExitDurable: true, Pending: []*core.WorkflowAttestInput{&input}}}
	e.journal.Runs[run.ID] = run
	e.active[run.ID] = &activeRun{workspaceLock: lock}
	receipt := Receipt{RequestID: f.id(), CommandID: stop.Command.ID, DispatchID: stop.Command.DispatchID, RunID: run.ID, Generation: run.Generation, Phase: "stopped"}
	e.journal.Receipts[receipt.CommandID] = &receipt
	e.journal.Pending = []*Receipt{&receipt}
	f.check(e.AckReport(receipt, terminal))
	if !run.Automation.ManagedAck {
		t.Fatal("actual terminal response did not acknowledge exit command")
	}
	f.check(e.flushAutomation(context.Background()))
	response := <-responses
	if response.Attestation.ControlEpoch != binding.ControlEpoch || response.ControlEpoch <= binding.ControlEpoch {
		t.Fatal("fixture did not cross pause epoch")
	}
	if !run.Automation.ExitAck || !run.Automation.LockReleased || len(e.active) != 0 {
		t.Fatal("real terminal HTTP ACK left checkout blocked")
	}
	nextLock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "next attempt")
	f.check(err)
	f.check(nextLock.release(true))
	page = f.tick(w.Workflow.ID)
	_, err = f.s.WorkflowCommand(f.ctx, f.a, core.WorkflowCommandInput{RequestID: f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, Kind: "resume"})
	f.check(err)
	f.tick(w.Workflow.ID)
	resumed := f.poll(first.WorkerEpoch)
	if resumed.Command == nil || resumed.Command.Kind != "resume" || resumed.Run.ID != first.Run.ID || resumed.Run.ProviderSessionID != syntheticSession || resumed.Run.Generation != first.Run.Generation+1 {
		t.Fatalf("resume lost exact native conversation or fresh generation: command=%+v run=%+v first_generation=%d", resumed.Command, resumed.Run, first.Run.Generation)
	}
}
