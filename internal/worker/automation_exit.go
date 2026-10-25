package worker

import (
	"context"
	"errors"
	"net/http"
	"time"

	"agent-board/internal/core"
)

func (e *Engine) isAutomation(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.journal.Runs[id]
	return r != nil && r.Automation != nil
}
func (e *Engine) observeAutomationExit(id string, child process, readError, exitError error) {
	e.mu.Lock()
	run, active := e.journal.Runs[id], e.active[id]
	if run == nil || active == nil || active.automation == nil {
		e.mu.Unlock()
		return
	}
	a, rt := run.Automation, active.automation
	a.NativeExited = true
	rt.closing = true
	rt.cancel()
	if rt.checkCount == 0 {
		close(rt.checksDone)
	}
	checks := []process{}
	for _, check := range rt.checks {
		checks = append(checks, check)
	}
	saveErr := e.save()
	e.mu.Unlock()
	if rt.ipc != nil {
		_ = rt.ipc.Close()
	}
	checkStopError := e.stopBatch(checks)
	checksExited := false
	select {
	case <-rt.checksDone:
		checksExited = true
	case <-time.After(9 * time.Second):
	}
	e.mu.Lock()
	uncertain := active.uncertain || errors.Is(exitError, errOwnership) || checkStopError != nil || !checksExited || saveErr != nil
	if uncertain {
		active.uncertain = true
		run.State = "needs_recovery"
		run.Error = "automation process trees or proof are uncertain; local recovery required"
		e.reportFor(run.CommandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
		e.mu.Unlock()
		return
	}
	runCopy := *run
	config := snapshotConfig(e.config)
	stopped := active.stop
	e.mu.Unlock()
	consent, directory := config.Automation.Workspaces[runCopy.Workspace], config.Workspaces[runCopy.Workspace].Path
	role := a.Run.Profile.Role
	proofCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	proof, proofErr := e.observeAutomationGit(proofCtx, consent, directory)
	var interrupted *interruptedWork
	var dirtyErr error
	if role == "developer" && stopped && !errors.Is(proofErr, errOwnership) {
		interrupted, dirtyErr = e.observeInterruptedWork(proofCtx, consent, directory, a.Run.Automation)
	}
	cancel()
	hasSession := sessionID.MatchString(runCopy.SessionID)
	var task core.Task
	var taskErr error = errors.New("no native session")
	if hasSession {
		task, taskErr = e.readExitTask(a.Run.TaskKey)
	}
	clean := proofErr == nil && unchangedProof(a.Baseline, proof, role != "developer") == nil
	if (role == "reviewer" || role == "tester") && proof.Head != a.Run.Automation.TargetSHA {
		clean = false
	}
	if taskErr != nil || task.Key != a.Run.TaskKey || task.Role != role || (task.Generation != a.Run.Automation.TaskGeneration && task.Generation != a.Run.Automation.TaskGeneration-1) {
		clean = false
	}
	// SourceClean describes the checkout only. A replaced provider executable
	// (auto-update, new inode) is a local diagnostic, never dirty source; a
	// changed workspace directory identity still makes the proof unusable.
	providerChanged := false
	var pin pinError
	if pinErr := e.checkAutomationExitPins(runCopy.Provider, runCopy.Workspace); errors.As(pinErr, &pin) && pin.kind == "provider" {
		providerChanged = true
	} else if pinErr != nil {
		clean = false
	}
	if role == "developer" && task.State == "done" && task.Evidence.Commit != proof.Head {
		clean = false
	}
	observations := core.WorkflowObservations{ActualHead: proof.Head, TargetSHA: a.Run.Automation.TargetSHA, SourceClean: &clean, ManifestSHA256: proof.Manifest}
	if role == "tester" && clean && task.Review != nil && task.Review.Outcome == "not_run" && consent.AllowTesterArtifactWrites {
		refs := qualifyingDesignRefs(context.Background(), directory, consent, a.Baseline.Artifacts, proof.Artifacts)
		if len(refs) > 0 {
			observations.TestDesign = true
			observations.ArtifactRefs = refs
		}
	}
	e.mu.Lock()
	if errors.Is(proofErr, errOwnership) || errors.Is(dirtyErr, errOwnership) {
		active.uncertain = true
		run.State = "needs_recovery"
		run.Error = "Git observation process ownership is uncertain"
		e.reportFor(run.CommandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
		e.mu.Unlock()
		return
	}
	a.ExitDurable = true
	a.ExitObservations = &observations
	if interrupted != nil {
		a.Interrupted = interrupted
	}
	if providerChanged {
		const note = "provider executable identity changed; run `agent-board worker consent`"
		run.LocalDiagnostic = "provider_pin_changed"
		run.Error = joinNote(run.Error, note)
		run.Notice = joinNote(run.Notice, note)
	}
	// Missing provider identity or an unavailable task fact cannot authenticate an
	// exit attestation. The proven local exit then needs no proof: it is reported
	// as a plain terminal failure/stop and never followed by delivery_unknown.
	attestable := hasSession && taskErr == nil && task.Key == a.Run.TaskKey
	if attestable {
		commandID := run.CommandID
		if active.stopCommand != "" {
			commandID = active.stopCommand
		}
		receipt := e.journal.Receipts[commandID]
		input := core.WorkflowAttestInput{ManagedPollInput: e.pollInputLocked(randomID("exit_")), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: run.ID, Generation: run.Generation, WorkflowID: a.Run.Automation.WorkflowID, StepID: a.Run.Automation.StepID, ControlEpoch: a.Run.Automation.ControlEpoch, TaskKey: a.Run.TaskKey, TaskGeneration: task.Generation, NativeSessionID: run.SessionID, Kind: "run_exit", Observations: observations}
		a.Pending = append(a.Pending, &input)
	} else {
		a.ExitUnreportable = true
	}
	e.reportAutomationExitLocked(run, active, child, readError, exitError, hasSession, attestable)
	if err := e.save(); err != nil {
		active.uncertain = true
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()
	if !attestable {
		_ = e.releaseAutomationExit(id)
	}
}

// reportAutomationExitLocked queues exactly one terminal report per command.
func (e *Engine) reportAutomationExitLocked(run *RunState, active *activeRun, child process, readError, exitError error, hasSession, attestable bool) {
	switch {
	case active.stop && active.stopCommand != "":
		run.State = "stopped"
		if !active.observed {
			e.reportFor(run.CommandID, "failed", run, "stopped before native turn was observed", "")
		}
		e.journal.Receipts[active.stopCommand].Forced = child.Forced()
		e.reportFor(active.stopCommand, "stopped", run, "", "")
	case active.stop:
		// A worker-local stop (local deadline, shutdown) is this live worker's
		// normal terminal stop of the execution command.
		run.State = "stopped"
		e.journal.Receipts[run.CommandID].Forced = child.Forced()
		e.reportFor(run.CommandID, "recovered_stopped", run, run.Error, "")
	case !hasSession:
		run.State = "failed"
		run.LocalDiagnostic = "native_exit_before_session"
		run.Error = "native_exit_before_session: native provider exited before announcing a session; check local login (codex login / claude), CLI version and flags"
		e.reportFor(run.CommandID, "failed", run, run.Error, "")
	case !attestable:
		run.State = "failed"
		run.LocalDiagnostic = "exit_task_unavailable"
		run.Error = "exit_task_unavailable: native exit is proven locally but the task facts needed for its proof were unavailable"
		e.reportFor(run.CommandID, "failed", run, run.Error, "")
	case active.complete && active.observed && !active.failed && readError == nil && exitError == nil:
		run.State = "completed"
		e.reportFor(run.CommandID, "completed", run, "", run.Output)
	default:
		run.State = "failed"
		if run.Error == "" {
			run.Error = "native provider failed or no complete workflow turn was observed"
		}
		e.reportFor(run.CommandID, "failed", run, run.Error, "")
	}
}

func joinNote(text, note string) string {
	if text == "" {
		return note
	}
	return boundedText(text+"; "+note, 3000)
}

func (e *Engine) automationRead(ctx context.Context, path string, output any) error {
	e.mu.Lock()
	read := e.automationReadTransport
	e.mu.Unlock()
	if read == nil {
		return consentError()
	}
	return read(ctx, http.MethodGet, path, nil, output)
}
func (e *Engine) releaseAutomationExit(id string) error {
	e.mu.Lock()
	active, run := e.active[id], e.journal.Runs[id]
	if active == nil || run == nil || run.Automation == nil {
		e.mu.Unlock()
		return nil
	}
	a := run.Automation
	if active.uncertain || !a.NativeExited || !a.ExitDurable || !automationExitSettled(a) {
		e.mu.Unlock()
		return errors.New("automation exit has unresolved proof")
	}
	lock := active.workspaceLock
	e.mu.Unlock()
	var err error
	if lock != nil {
		err = lock.release(true)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		active.uncertain = true
		run.State = "needs_recovery"
		run.Error = "workspace proof release failed"
		_ = e.save()
		return err
	}
	run.Automation.LockReleased = true
	if compactAutomation(run.Automation) {
		run.Output = ""
	}
	delete(e.active, id)
	return e.save()
}

func (e *Engine) checkAutomationExitPins(provider, workspace string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.checkPins(provider, workspace)
}

// readExitTask retries transport failures briefly; a definitive 4xx or a
// mismatched fact is final. Without it no exit attestation can be built.
func (e *Engine) readExitTask(key string) (core.Task, error) {
	var task core.Task
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		task = core.Task{}
		err = e.automationRead(ctx, "/tasks/"+key, &task)
		cancel()
		var status statusError
		e.mu.Lock()
		offline := e.automationReadTransport == nil
		e.mu.Unlock()
		if err == nil || errors.As(err, &status) || offline {
			return task, err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return task, err
}

// observeInterruptedWork fingerprints a stopped developer step's uncommitted
// tree. Only provable trees (regular files, no unmerged entries) are recorded.
func (e *Engine) observeInterruptedWork(ctx context.Context, a AutomationWorkspace, directory string, binding *core.ManagedAutomationBinding) (*interruptedWork, error) {
	tree, err := readDirtyTree(ctx, a, directory, e.gitRunnerFor())
	if err != nil || tree.files == 0 || !tree.provable || binding == nil {
		if errors.Is(err, errOwnership) {
			return nil, err
		}
		return nil, nil
	}
	return &interruptedWork{WorkflowID: binding.WorkflowID, StepID: binding.StepID, Head: tree.head, Fingerprint: tree.fingerprint, Files: tree.files}, nil
}
