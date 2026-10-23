package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/core"
)

type statusFake struct {
	*fakeProcess
	code int
}

func (p *statusFake) ExitCode() (int, bool) { return p.code, true }
func automationTestEngine(t *testing.T) (*Engine, automationipc.Request) {
	t.Helper()
	e := testEngine(t)
	f := newAutomationFixture(t)
	a := f.config.Workspaces["repo"]
	for role := range a.Roles {
		a.Roles[role] = []string{"codex"}
	}
	root := f.spaces["repo"].Path
	e.config.Workspaces["test"] = Workspace{Path: root, Providers: []string{"codex"}}
	e.config.Executables = map[string]string{"codex": f.providers["codex"]}
	e.config.Automation = &AutomationConfig{Version: 1, Workspaces: map[string]AutomationWorkspace{"test": a}}
	e.config.NativeMCPApprovedTools = map[string][]string{"codex": append([]string{}, managedMCPTools...)}
	pins, err := pinPaths(e.config)
	if err != nil {
		t.Fatal(err)
	}
	e.journal.PathPins = pins
	command, managed := testDispatch("start")
	managed.Profile.Role = "tester"
	managed.Automation = &core.ManagedAutomationBinding{WorkflowID: "wf_test", StepID: "ws_test", ControlEpoch: 1, PolicyVersion: 1, TaskGeneration: 3, DeadlineAt: time.Now().Add(time.Minute), TargetSHA: strings.Repeat("a", 40), Checks: []core.AutomationCheckPresetRef{{PresetID: "unit", LocalVersion: 1, LocalDigest: a.Checks["unit"].TrustedOwnerConsent, Kind: "test", Required: true, Isolation: "trusted_owner_process"}}}
	managed.TaskKey = "T-1"
	baseline := gitProof{Head: managed.Automation.TargetSHA, Manifest: strings.Repeat("b", 64), Unknown: strings.Repeat("c", 64)}
	r := &RunState{ID: managed.ID, Provider: managed.Provider, Workspace: managed.Workspace, Generation: managed.Generation, State: "running", CommandID: command.ID, SessionID: syntheticSession, Automation: &automationAttempt{Run: managed, Baseline: baseline, Checks: map[string]*automationCheck{}}}
	e.journal.Runs[r.ID] = r
	e.journal.Receipts[command.ID] = &Receipt{CommandID: command.ID, DispatchID: command.DispatchID, RunID: r.ID, Generation: r.Generation, Phase: "started"}
	ctx, cancel := context.WithCancel(context.Background())
	rt := &automationRuntime{ctx: ctx, cancel: cancel, checks: map[string]process{}, checksDone: make(chan struct{}), deadline: time.Now().Add(time.Minute)}
	e.active[r.ID] = &activeRun{process: &fakeProcess{output: io.NopCloser(strings.NewReader("")), done: make(chan struct{})}, automation: rt}
	e.automationGit = func(context.Context, AutomationWorkspace, string) (gitProof, error) { return baseline, nil }
	request := automationipc.Request{Operation: "start", RequestID: "verification_one", WorkflowID: "wf_test", StepID: "ws_test", RunID: r.ID, ControlEpoch: 1, Generation: 1, SessionID: "board_native_session", TaskKey: "T-1", TaskGeneration: 3, PresetID: "unit"}
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		task := out.(*core.Task)
		*task = core.Task{Key: "T-1", Generation: 3, Lease: &core.Lease{SessionID: request.SessionID, Generation: 3, ExpiresAt: time.Now().Add(time.Minute)}}
		return nil
	}
	e.automationTransport = func(_ context.Context, _ string, input, output any) error {
		in := input.(core.WorkflowAttestInput)
		out := output.(*core.WorkflowAttestResult)
		status := "reserved"
		if in.Kind == "check_started" {
			status = "running"
		}
		if in.Kind == "check_finished" {
			status = "completed"
		}
		out.ControlEpoch = in.ControlEpoch
		out.MaxRuntimeSeconds = 60
		out.ReservationTTLSeconds = 30
		out.Attestation = core.WorkflowAttestation{ID: "server_check", Kind: "check", Status: status, WorkflowID: in.WorkflowID, StepID: in.StepID, ControlEpoch: in.ControlEpoch, RunID: in.RunID, Generation: in.Generation, TaskKey: in.TaskKey, TaskGeneration: in.TaskGeneration, NativeSessionID: in.NativeSessionID, CheckRequestID: in.CheckRequestID, PresetID: in.PresetID, PresetVersion: in.PresetVersion, PresetDigest: in.PresetDigest}
		return nil
	}
	if err = e.save(); err != nil {
		t.Fatal(err)
	}
	return e, request
}
func TestLocalCheckReplayNeverRespawnsAndCompletionWaitsDurableServerACK(t *testing.T) {
	e, request := automationTestEngine(t)
	var starts atomic.Int32
	e.start = func(spec launchSpec) (process, error) {
		starts.Add(1)
		report := filepath.Join(e.config.Workspaces["test"].Path, ".artifacts/test.tap")
		if err := os.WriteFile(report, []byte("TAP version 13\n1..1\nok 1 - actual testcase\n"), 0600); err != nil {
			return nil, err
		}
		done := make(chan struct{})
		close(done)
		return &statusFake{&fakeProcess{output: io.NopCloser(strings.NewReader("bounded output")), done: done}, 0}, nil
	}
	first, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || first.Status != "queued" {
		t.Fatal(first, err)
	}
	e.wg.Wait()
	request.Operation = "status"
	pending, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || pending.Status != "report_pending" || pending.Result == nil || pending.Result.Passed != 1 {
		t.Fatal(pending, err)
	}
	request.Operation = "start"
	retry, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || retry.CheckID != first.CheckID || starts.Load() != 1 {
		t.Fatal("retry spawned another check", retry, err)
	}
	changed := request
	changed.PresetID = "different"
	if _, err = e.automationCheckRequest(context.Background(), request.RunID, changed); err == nil {
		t.Fatal("changed payload reused the receipt")
	}
	if err = e.flushAutomation(context.Background()); err != nil {
		t.Fatal(err)
	}
	request.Operation = "status"
	finished, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || finished.Status != "completed" || finished.Result.ReportKind != "tap" {
		t.Fatal(finished, err)
	}
	if !strings.HasPrefix(finished.Result.ArtifactRefs[0], "sha256:") {
		t.Fatal("non-hash artifact escaped")
	}
}
func TestForeignLeaseAndStatusOnlyRequestsCannotExecute(t *testing.T) {
	e, request := automationTestEngine(t)
	var starts atomic.Int32
	e.start = func(launchSpec) (process, error) { starts.Add(1); return nil, errors.New("must not run") }
	request.Operation = "status"
	value, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || value.Status != "unknown" {
		t.Fatal(value, err)
	}
	if starts.Load() != 0 || len(e.journal.Runs[request.RunID].Automation.Checks) != 0 {
		t.Fatal("status created an execution")
	}
	request.Operation = "start"
	request.SessionID = "foreign"
	if _, err = e.automationCheckRequest(context.Background(), request.RunID, request); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	if starts.Load() != 0 {
		t.Fatal("foreign board session executed trusted code")
	}
	value, err = e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || value.Error != "lease_checkpoint_unavailable" {
		t.Fatal(value, err)
	}
}
func TestExecutionCredentialFileAndBridgeNeverContainOwnerBearer(t *testing.T) {
	e, request := automationTestEngine(t)
	r := e.journal.Runs[request.RunID]
	command, _ := testDispatch("start")
	p := plan{command: command, run: r.Automation.Run}
	e.mu.Lock()
	err := e.createCapabilityLocked(p, r)
	e.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	token, err := ReadToken(r.Automation.TokenFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, core.ExecutionTokenPrefix) || len(token) != len(core.ExecutionTokenPrefix)+64 {
		t.Fatal("capability is not a 256-bit attempt secret")
	}
	digest := sha256.Sum256([]byte(token))
	if r.Automation.CapabilityDigest != hex.EncodeToString(digest[:]) || len(r.Automation.CapabilityID) != 35 {
		t.Fatal("credential digest/ID mismatch")
	}
	r.Automation.Registered = true
	r.Automation.IPC = &automationipc.Config{Endpoint: "private-endpoint", Nonce: strings.Repeat("a", 64), Binding: automationipc.Binding{WorkflowID: "wf_test", StepID: "ws_test", RunID: r.ID, ControlEpoch: 1, Generation: 1}}
	if _, err = e.bridgeArgs(*r); err != nil {
		t.Fatal(err)
	}
	bridgePath := filepath.Join(e.config.StateDir, "runs", r.ID, "attempt-1", "bridge.json")
	raw, err := os.ReadFile(bridgePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), e.config.TokenFile) || strings.Contains(string(raw), "synthetic-private-token") || strings.Contains(string(raw), e.journal.HostSecret) {
		t.Fatal("parent credential leaked into child config")
	}
	config, childToken, err := ReadBridgeConfig(bridgePath)
	if err != nil || childToken != token || config.ExecutionCapabilityID != r.Automation.CapabilityID {
		t.Fatal("scoped bridge cannot read its own exact credential", err)
	}
}
