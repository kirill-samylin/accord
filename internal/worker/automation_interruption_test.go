package worker

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"agent-board/internal/core"
)

// Ported from the TASK022 independent review (W1). A graceful local shutdown
// while an automation step runs: the owned process exits under the worker's
// own stop, so the next start must not quarantine the host.
func TestGracefulShutdownDuringAutomationStepIsTerminalStopAfterRestart(t *testing.T) {
	e, request := automationTestEngine(t)
	run := e.journal.Runs[request.RunID]
	child := e.active[run.ID].process
	e.wg.Add(1)
	go e.observe(run.ID, child)
	_ = e.Close() // No delivery: emulates a network that is down during shutdown.
	e2, err := Open(e.config)
	if err != nil {
		t.Fatal(err)
	}
	defer e2.Close()
	restored := e2.journal.Runs[run.ID]
	if restored.State == "needs_recovery" {
		t.Fatal("clean worker-initiated stop of an owned automation process is ownership-uncertain after restart")
	}
	if err = e2.prepareReconciliation(); err != nil {
		t.Fatal(err)
	}
	report := e2.Pending()
	if report == nil || report.Phase != "recovered_stopped" || report.CommandID != run.CommandID {
		t.Fatal("restart did not report the proven exit as a terminal stop", report)
	}
	for _, p := range e2.journal.Pending {
		if p.Phase == "delivery_unknown" {
			t.Fatal("proven exit reported as delivery_unknown")
		}
	}
}

// Ported from review W2: the provider app auto-updates its CLI (atomic replace,
// new inode) during a step. That is a provider diagnostic, not dirty source.
func TestProviderExecutableReplacedDuringStepKeepsSourceClean(t *testing.T) {
	e, request := automationTestEngine(t)
	run := e.journal.Runs[request.RunID]
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		*out.(*core.Task) = core.Task{Key: "T-1", Role: "tester", Generation: 3}
		return nil
	}
	exe := e.config.Executables["codex"]
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(exe+".new", data, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(exe+".new", exe); err != nil {
		t.Fatal(err)
	}
	child := e.active[run.ID].process
	e.wg.Add(1)
	go e.observe(run.ID, child)
	_ = child.Stop()
	e.wg.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	o := run.Automation.ExitObservations
	if o == nil || o.SourceClean == nil || !*o.SourceClean {
		t.Fatal("provider executable replacement was folded into SourceClean=false")
	}
	if run.LocalDiagnostic != "provider_pin_changed" || !strings.Contains(run.Error, "provider executable identity changed; run `agent-board worker consent`") {
		t.Fatal("provider change was not recorded as a local diagnostic", run.LocalDiagnostic, run.Error)
	}
	var pin pinError
	if err = e.checkPins(run.Provider, run.Workspace); err == nil || !strings.Contains(err.Error(), "local workspace or provider identity changed") || !asPin(err, &pin) || pin.kind != "provider" {
		t.Fatal("launch pin check does not name the changed provider pin", err)
	}
}

func asPin(err error, pin *pinError) bool {
	p, ok := err.(pinError)
	*pin = p
	return ok
}

// Ported from review W3: the native CLI exits before announcing a provider
// session (not logged in, unknown flag after an update).
func TestNativeExitBeforeSessionIsTerminalFailure(t *testing.T) {
	e, request := automationTestEngine(t)
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		*out.(*core.Task) = core.Task{Key: "T-1", Role: "tester", Generation: 3}
		return nil
	}
	lock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "worker run_test")
	if err != nil {
		t.Fatal(err)
	}
	run := e.journal.Runs[request.RunID]
	run.SessionID = ""
	run.State = "launching"
	e.active[run.ID].workspaceLock = lock
	child := e.active[run.ID].process
	_ = child.Stop()
	e.wg.Add(1)
	go e.observe(run.ID, child)
	e.wg.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	phases := []string{}
	for _, p := range e.journal.Pending {
		phases = append(phases, p.Phase)
	}
	if run.State != "failed" || run.LocalDiagnostic != "native_exit_before_session" || len(run.Automation.Pending) != 0 {
		t.Fatal("a proven, owned, pre-session exit is not a terminal failure", run.State, run.Error)
	}
	if len(phases) != 1 || phases[0] != "failed" {
		t.Fatal("pre-session exit must queue exactly one failed report", phases)
	}
	if !run.Automation.LockReleased || e.active[run.ID] != nil {
		t.Fatal("pre-session exit kept the checkout blocked")
	}
	e.reportFor(run.CommandID, "delivery_unknown", run, "late", "")
	if len(e.journal.Pending) != 1 {
		t.Fatal("delivery_unknown was queued after a terminal report for the same command")
	}
}

// Ported from review W4: a human pause lands between check_reserved and
// check_started; the server definitively rejects check_started (403).
func TestDefinitiveCheckStartRejectionCancelsCheckAndKeepsRunHealthy(t *testing.T) {
	e, request := automationTestEngine(t)
	inner := e.automationTransport
	e.automationTransport = func(ctx context.Context, path string, input, output any) error {
		if in, ok := input.(core.WorkflowAttestInput); ok && in.Kind == "check_started" {
			return statusError{code: 403, server: "FORBIDDEN_ROLE", message: "Workflow остановлен"}
		}
		return inner(ctx, path, input, output)
	}
	e.start = func(spec launchSpec) (process, error) {
		return &statusFake{&fakeProcess{output: io.NopCloser(strings.NewReader("")), done: make(chan struct{})}, 0}, nil
	}
	if _, err := e.automationCheckRequest(context.Background(), request.RunID, request); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	run := e.journal.Runs[request.RunID]
	c := run.Automation.Checks[request.RequestID]
	if run.State == "needs_recovery" || e.active[run.ID].uncertain || c.Status != "not_run" || c.ErrorCode != "check_cancelled" {
		t.Fatal("definitive 403 on check_started (process stopped and reaped) is treated as ownership-uncertain", run.State, c.Status, c.ErrorCode)
	}
}
