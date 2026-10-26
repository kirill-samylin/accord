package worker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"agent-board/internal/automationipc"
)

// Called with mu held. Blocking consent hashing, Git, IPC and registration run
// outside mu; the ownership poll and independent stop watchdog stay live.
func (e *Engine) launchAutomationLocked(p plan) error {
	run := e.journal.Runs[p.run.ID]
	if err := e.checkPins(run.Provider, run.Workspace); err != nil {
		run.State = "failed"
		run.LocalDiagnostic, run.Error = classifyPreparation(err)
		e.reportFor(p.command.ID, "failed", run, run.Error, "")
		return e.save()
	}
	config := snapshotConfig(e.config)
	nativeEnv := providerEnv()
	epoch := e.stopEpoch
	ctx, cancel := context.WithCancel(context.Background())
	runtime := &automationRuntime{ctx: ctx, cancel: cancel, checks: map[string]process{}, checksDone: make(chan struct{})}
	active := &activeRun{automation: runtime}
	e.active[run.ID] = active
	fail := func(cause error) error { return e.failAutomationPreparationLocked(p, active, cause) }
	var consent AutomationWorkspace
	var prepErr error
	e.mu.Unlock()
	consent, prepErr = (&Engine{config: config}).automationAllowed(p.run, time.Now())
	if prepErr == nil {
		prepErr = validateAutomationNativeConsent(config)
	}
	e.mu.Lock()
	if prepErr != nil {
		return fail(errConsent)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	remaining := min(time.Until(p.run.Automation.DeadlineAt), time.Duration(consent.MaxAttemptSeconds)*time.Second, time.Until(consent.ExpiresAt))
	if remaining <= 0 {
		return fail(errConsent)
	}
	runtime.deadline = time.Now().Add(remaining)
	budget, err := e.automationBudgetLocked(p.run.Automation, consent)
	if err != nil {
		return fail(err)
	}
	if budget.DeadlineAt.Before(runtime.deadline) {
		runtime.deadline = budget.DeadlineAt
	}
	if run.Automation != nil && !automationExitSettled(run.Automation) {
		return fail(errProofOpen)
	}
	previous := run.Automation
	run.Automation = &automationAttempt{Run: p.run, Checks: map[string]*automationCheck{}}
	if previous != nil && previous.Interrupted != nil && previous.Interrupted.Stash == "" {
		run.Automation.Interrupted = previous.Interrupted
	}
	run.ProcessGroups, run.RecoveryNotified, run.Notice = nil, false, ""
	if err := e.save(); err != nil {
		return fail(err)
	}
	directory := config.Workspaces[run.Workspace].Path
	e.mu.Unlock()
	lock, err := acquireWorkspace(directory, e.workspaceLockDir, e.journal.WorkerID+" "+run.ID)
	e.mu.Lock()
	active.workspaceLock = lock
	if err != nil {
		return fail(errBusy)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	interrupted := e.interruptedCandidatesLocked(run)
	proofCtx, proofCancel := context.WithTimeout(runtime.ctx, 40*time.Second)
	e.mu.Unlock()
	cleaned, err := e.prepareCleanCheckout(proofCtx, consent, directory, run.Workspace, p.run.Automation, interrupted)
	var baseline gitProof
	if err == nil {
		baseline, err = e.observeAutomationGit(proofCtx, consent, directory)
		if err != nil && !errors.Is(err, errOwnership) {
			err = errSource
		}
	}
	proofCancel()
	e.mu.Lock()
	e.recordStashLocked(run, cleaned)
	if err != nil {
		return fail(err)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	if p.run.Profile == nil {
		return fail(errConsent)
	}
	role := p.run.Profile.Role
	if (role == "reviewer" || role == "tester") && (baseline.Head != p.run.Automation.TargetSHA || !fullCommit(p.run.Automation.TargetSHA)) {
		return fail(errTargetSHA)
	}
	run.Automation.Baseline = baseline
	if err = e.createCapabilityLocked(p, run); err != nil {
		return fail(err)
	}
	attemptDir := filepath.Join(config.StateDir, "runs", run.ID, "attempt-"+strconv.FormatInt(run.Generation, 10))
	binding := automationipc.Binding{WorkflowID: p.run.Automation.WorkflowID, StepID: p.run.Automation.StepID, RunID: run.ID, ControlEpoch: p.run.Automation.ControlEpoch, Generation: run.Generation}
	e.mu.Unlock()
	server, err := automationipc.Listen(attemptDir, binding, func(ctx context.Context, in automationipc.Request) (automationipc.Response, error) {
		return e.automationCheckRequest(ctx, p.run.ID, in)
	})
	e.mu.Lock()
	if err != nil {
		return fail(errIPC)
	}
	runtime.ipc = server
	ipcConfig := server.Config()
	run.Automation.IPC = &ipcConfig
	if err = e.save(); err != nil {
		return fail(err)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	e.mu.Unlock()
	err = e.registerCapability(runtime.ctx, p, run.Automation)
	e.mu.Lock()
	if err != nil {
		return fail(registrationCause(err))
	}
	run.Automation.Registered = true
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	args, err := providerArgs(run.Provider, run.SessionID)
	if err != nil {
		return fail(err)
	}
	bridge, err := e.prepareBridgeLocked(p, run)
	if err != nil {
		return fail(err)
	}
	args = append(bridge.Args, args...)
	nativeEnv = append(nativeEnv, bridge.Env...)
	permission, permissionArgs, err := e.prepareGitPermissionsLocked(p, active, epoch, config, nil, nativeEnv)
	if err != nil {
		return fail(err)
	}
	if run.Provider == "codex" {
		sandbox := automationSandbox(config, role, run.Workspace)
		if permission != nil {
			args = append(permissionArgs, args...)
			sandbox = ""
		}
		if sandbox != "" {
			args = append([]string{"-c", fmt.Sprintf("sandbox_mode=%q", sandbox)}, args...)
		}
	}
	if run.Provider == "claude" && config.ClaudePermissionMode != "" {
		mode := config.ClaudePermissionMode
		if role != "developer" {
			mode = "default"
		}
		args = append(args, "--permission-mode", mode)
	}
	if err = e.appendAutomationModel(p, &args); err != nil {
		return fail(err)
	}
	if permission != nil {
		permission, _, err = e.prepareGitPermissionsLocked(p, active, epoch, config, permission, nativeEnv)
		if err != nil {
			return fail(err)
		}
		run.Automation.PermissionProof = permission
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(errStopped)
	}
	// Pins are re-checked right before spawn: a provider auto-update during
	// preparation is named precisely instead of surfacing later as dirty source.
	if err = e.checkPins(run.Provider, run.Workspace); err != nil {
		return fail(err)
	}
	if err = budgetRemaining(budget, consent); err != nil {
		return fail(err)
	}
	run.State, run.CommandID, run.Generation = "launching", p.command.ID, p.command.Generation
	if err = e.save(); err != nil {
		return fail(err)
	}
	// The durable intent and active preparation handle fence a stop racing spawn.
	// A returned child is always adopted into this attempt and stopped if revoked.
	e.mu.Unlock()
	child, err := e.start(launchSpec{executable: config.Executables[run.Provider], directory: directory, args: args, env: nativeEnv, prompt: e.prompt(p), onGroup: e.groupRecorder(run.ID)})
	e.mu.Lock()
	if err != nil {
		return fail(errStart)
	}
	// Only an actually started native process consumes a local invocation.
	budget.Invocations++
	e.recordSupervisorLocked(run.ID, child)
	_ = e.save() // A failed save sets fatal; the watchdog then stops this child.
	active.process = child
	if !e.automationPreparingAfterSpawn(p, active, epoch, config) {
		active.stop = true
		go e.stopOwned(run.ID, active, p.command.ID)
	}
	e.wg.Add(1)
	go e.observe(run.ID, child)
	return nil
}
func (e *Engine) automationPreparing(p plan, a *activeRun, epoch uint64, c Config) bool {
	r := e.journal.Runs[p.run.ID]
	return e.active[p.run.ID] == a && !e.closed && e.fatal == nil && !a.stop && a.automation.ctx.Err() == nil && e.stopEpoch == epoch && time.Now().Before(e.deadline) && e.allowedController(p.command.CreatedBy) && reflect.DeepEqual(e.config, c) && r != nil && r.Generation == p.run.Generation && (a.automation.deadline.IsZero() || time.Now().Before(a.automation.deadline))
}
func (e *Engine) automationPreparingAfterSpawn(p plan, a *activeRun, epoch uint64, c Config) bool {
	return e.automationPreparing(p, a, epoch, c)
}
func (e *Engine) appendAutomationModel(p plan, args *[]string) error {
	profile := p.run.Profile
	if profile == nil {
		return consentError()
	}
	expected := profileProviderFor(p.run.Provider)
	if expected == "" || profile.Provider != expected && profile.Provider != "unconfigured" {
		return consentError()
	}
	if profile.Model != "" {
		found := false
		for _, model := range e.config.Models[p.run.Provider] {
			found = found || model == profile.Model
		}
		if !found {
			return consentError()
		}
		*args = append(*args, "--model", profile.Model)
	}
	return nil
}
func (e *Engine) failAutomationPreparationLocked(p plan, active *activeRun, cause error) error {
	run := e.journal.Runs[p.run.ID]
	active.automation.cancel()
	proven := !errors.Is(cause, errOwnership)
	e.mu.Unlock()
	if active.automation.ipc != nil {
		_ = active.automation.ipc.Close()
	}
	var releaseErr error
	if active.workspaceLock != nil {
		releaseErr = active.workspaceLock.release(proven)
	}
	e.mu.Lock()
	delete(e.active, run.ID)
	if !proven || releaseErr != nil {
		run.State = "needs_recovery"
		run.Error = "automation preparation ownership is uncertain; local recovery required"
		e.reportFor(p.command.ID, "delivery_unknown", run, run.Error, "")
	} else {
		run.State = "failed"
		// The specific cause survives into the failed report and local state.
		run.LocalDiagnostic, run.Error = classifyPreparation(cause)
		e.reportFor(p.command.ID, "failed", run, run.Error, "")
		if active.stopCommand != "" {
			run.State = "stopped"
			run.Error = ""
			e.reportFor(active.stopCommand, "stopped", run, "", "")
		}
	}
	return e.save()
}
