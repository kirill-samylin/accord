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
		run.Error = "local workspace or provider identity changed"
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
		return fail(prepErr)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(consentError())
	}
	remaining := min(time.Until(p.run.Automation.DeadlineAt), time.Duration(consent.MaxAttemptSeconds)*time.Second, time.Until(consent.ExpiresAt))
	if remaining <= 0 {
		return fail(consentError())
	}
	runtime.deadline = time.Now().Add(remaining)
	if e.journal.AutomationBudgets == nil {
		e.journal.AutomationBudgets = map[string]*automationBudget{}
	}
	budget := e.journal.AutomationBudgets[p.run.Automation.WorkflowID]
	if budget == nil {
		now := time.Now()
		budget = &automationBudget{FirstStartedAt: now, DeadlineAt: now.Add(time.Duration(consent.MaxWorkflowSeconds) * time.Second)}
		e.journal.AutomationBudgets[p.run.Automation.WorkflowID] = budget
	}
	if time.Now().Before(budget.FirstStartedAt) || !time.Now().Before(budget.DeadlineAt) || budget.Invocations >= consent.MaxNativeInvocations {
		return fail(errors.New("local workflow budget exhausted"))
	}
	budget.Invocations++
	if e.automationDeadlines == nil {
		e.automationDeadlines = map[string]time.Time{}
	}
	localDeadline, exists := e.automationDeadlines[p.run.Automation.WorkflowID]
	if !exists {
		localDeadline = time.Now().Add(time.Until(budget.DeadlineAt))
		e.automationDeadlines[p.run.Automation.WorkflowID] = localDeadline
	}
	if localDeadline.Before(runtime.deadline) {
		runtime.deadline = localDeadline
	}
	if budget.DeadlineAt.Before(runtime.deadline) {
		runtime.deadline = time.Now().Add(time.Until(budget.DeadlineAt))
	}
	if run.Automation != nil && (!run.Automation.ExitAck || len(run.Automation.Pending) > 0) {
		return fail(errors.New("previous automation proof is unresolved"))
	}
	run.Automation = &automationAttempt{Run: p.run, Checks: map[string]*automationCheck{}}
	if err := e.save(); err != nil {
		return fail(err)
	}
	directory := config.Workspaces[run.Workspace].Path
	e.mu.Unlock()
	lock, err := acquireWorkspace(directory, e.workspaceLockDir, e.journal.WorkerID+" "+run.ID)
	e.mu.Lock()
	active.workspaceLock = lock
	if err != nil {
		return fail(err)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(consentError())
	}
	proofCtx, proofCancel := context.WithTimeout(runtime.ctx, 20*time.Second)
	e.mu.Unlock()
	baseline, err := e.observeAutomationGit(proofCtx, consent, directory)
	proofCancel()
	e.mu.Lock()
	if err != nil {
		return fail(err)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(consentError())
	}
	if p.run.Profile == nil {
		return fail(consentError())
	}
	role := p.run.Profile.Role
	if (role == "reviewer" || role == "tester") && (baseline.Head != p.run.Automation.TargetSHA || !fullCommit(p.run.Automation.TargetSHA)) {
		return fail(errors.New("reviewed full SHA differs from local HEAD"))
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
		return fail(err)
	}
	runtime.ipc = server
	ipcConfig := server.Config()
	run.Automation.IPC = &ipcConfig
	if err = e.save(); err != nil {
		return fail(err)
	}
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(consentError())
	}
	e.mu.Unlock()
	err = e.registerCapability(runtime.ctx, p, run.Automation)
	e.mu.Lock()
	if err != nil {
		return fail(err)
	}
	run.Automation.Registered = true
	if !e.automationPreparing(p, active, epoch, config) {
		return fail(consentError())
	}
	args, err := providerArgs(run.Provider, run.SessionID)
	if err != nil {
		return fail(err)
	}
	bridgeArgs, err := e.prepareBridgeLocked(p, run)
	if err != nil {
		return fail(err)
	}
	args = append(bridgeArgs, args...)
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
		return fail(consentError())
	}
	run.State, run.CommandID, run.Generation = "launching", p.command.ID, p.command.Generation
	if err = e.save(); err != nil {
		return fail(err)
	}
	// The durable intent and active preparation handle fence a stop racing spawn.
	// A returned child is always adopted into this attempt and stopped if revoked.
	e.mu.Unlock()
	child, err := e.start(launchSpec{executable: config.Executables[run.Provider], directory: directory, args: args, env: nativeEnv, prompt: e.prompt(p)})
	e.mu.Lock()
	if err != nil {
		return fail(err)
	}
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
	expected := map[string]string{"codex": "openai", "claude": "anthropic"}[p.run.Provider]
	if profile.Provider != expected && profile.Provider != "unconfigured" {
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
		run.Error = "automation local consent, source proof or scoped registration is unavailable"
		if errors.Is(cause, errPermissionProof) {
			run.Error = errPermissionProof.Error()
			run.LocalDiagnostic = "native_git_permission_unproven"
		}
		e.reportFor(p.command.ID, "failed", run, run.Error, "")
		if active.stopCommand != "" {
			run.State = "stopped"
			run.Error = ""
			e.reportFor(active.stopCommand, "stopped", run, "", "")
		}
	}
	return e.save()
}
