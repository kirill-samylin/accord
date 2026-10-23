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
	e.mu.Unlock()
	proofCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	proof, proofErr := e.observeAutomationGit(proofCtx, config.Automation.Workspaces[runCopy.Workspace], config.Workspaces[runCopy.Workspace].Path)
	cancel()
	var task core.Task
	taskCtx, taskCancel := context.WithTimeout(context.Background(), 4*time.Second)
	taskErr := e.automationRead(taskCtx, "/tasks/"+a.Run.TaskKey, &task)
	taskCancel()
	role := a.Run.Profile.Role
	clean := proofErr == nil && unchangedProof(a.Baseline, proof, role != "developer") == nil
	if (role == "reviewer" || role == "tester") && proof.Head != a.Run.Automation.TargetSHA {
		clean = false
	}
	if taskErr != nil || task.Key != a.Run.TaskKey || task.Role != role || (task.Generation != a.Run.Automation.TaskGeneration && task.Generation != a.Run.Automation.TaskGeneration-1) {
		clean = false
	}
	if pinErr := e.checkAutomationExitPins(runCopy.Provider, runCopy.Workspace); pinErr != nil {
		clean = false
	}
	if role == "developer" && task.State == "done" && task.Evidence.Commit != proof.Head {
		clean = false
	}
	observations := core.WorkflowObservations{ActualHead: proof.Head, TargetSHA: a.Run.Automation.TargetSHA, SourceClean: &clean, ManifestSHA256: proof.Manifest}
	if role == "tester" && clean && task.Review != nil && task.Review.Outcome == "not_run" && config.Automation.Workspaces[runCopy.Workspace].AllowTesterArtifactWrites {
		refs := qualifyingDesignRefs(context.Background(), config.Workspaces[runCopy.Workspace].Path, config.Automation.Workspaces[runCopy.Workspace], a.Baseline.Artifacts, proof.Artifacts)
		if len(refs) > 0 {
			observations.TestDesign = true
			observations.ArtifactRefs = refs
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if errors.Is(proofErr, errOwnership) {
		active.uncertain = true
		run.State = "needs_recovery"
		run.Error = "Git observation process ownership is uncertain"
		e.reportFor(run.CommandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
		return
	}
	a.ExitDurable = true
	a.ExitObservations = &observations
	// Missing provider identity or an unavailable task fact cannot authenticate an
	// exit attestation. Preserve a negative local proof and stop automatic reuse.
	attestable := sessionID.MatchString(run.SessionID) && taskErr == nil && task.Key == a.Run.TaskKey
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
	if active.stop {
		run.State = "stopped"
		if !active.observed {
			e.reportFor(run.CommandID, "failed", run, "stopped before native turn was observed", "")
		}
		if active.stopCommand != "" {
			e.journal.Receipts[active.stopCommand].Forced = child.Forced()
			e.reportFor(active.stopCommand, "stopped", run, "", "")
		} else {
			e.journal.Receipts[run.CommandID].Forced = child.Forced()
			e.reportFor(run.CommandID, "recovered_stopped", run, run.Error, "")
		}
	} else if active.complete && active.observed && !active.failed && readError == nil && exitError == nil {
		run.State = "completed"
		e.reportFor(run.CommandID, "completed", run, "", run.Output)
	} else {
		run.State = "failed"
		if run.Error == "" {
			run.Error = "native provider failed or no complete workflow turn was observed"
		}
		e.reportFor(run.CommandID, "failed", run, run.Error, "")
	}
	if err := e.save(); err != nil {
		active.uncertain = true
		return
	}
	if a.ExitUnreportable {
		run.State = "needs_recovery"
		run.Error = "native exit is proven locally but task or provider identity is unavailable"
		e.reportFor(run.CommandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
	}
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
	if active.uncertain || !a.NativeExited || !a.ExitDurable || !a.ExitAck || len(a.Pending) > 0 {
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
