package worker

// A worker restart cannot deliver a run_exit attestation signed by the previous
// incarnation: the server binds it to the old worker ID and epoch, and this
// worker never re-signs old proof. When the native exit itself was proven and
// persisted before the restart (NativeExited+ExitDurable with a terminal local
// state, which implies every owned check also exited), ownership is not in
// doubt; only delivery of the positive proof is lost. Such a run is reported as
// a terminal stop (recovered_stopped), never as a qualifying result, and its
// checkout is released after the server acknowledges that stop.

func provenAutomationExit(run *RunState) bool {
	a := run.Automation
	return a != nil && a.NativeExited && a.ExitDurable && (run.State == "stopped" || run.State == "completed" || run.State == "failed")
}

// abandonUndeliveredProof is called only while opening the journal under the
// exclusive state lock, before any network or process work.
func abandonUndeliveredProof(run *RunState, code string) {
	a := run.Automation
	if automationExitSettled(a) {
		// The server already holds the exit proof, or the exit was already
		// settled as unreportable with its terminal report queued or
		// acknowledged; only the checkout release may be left and it needs no
		// report. Re-abandoning would rewrite an acknowledged terminal state.
		return
	}
	a.Pending = nil
	a.ExitUnreportable = true
	a.ProofAbandoned = true
	run.State = "stopped"
	// RecoveryNotified is never cleared: a recovery report this run already
	// had acknowledged keeps it settled across restarts.
	run.LocalDiagnostic = code
	run.Error = code + ": native exit was proven locally but its proof was not delivered before the worker stopped; reported as a terminal stop"
}

// automationExitSettled reports that no exit proof for this attempt is still
// owed to the server (acknowledged, or explicitly unreportable).
func automationExitSettled(a *automationAttempt) bool {
	return (a.ExitAck || a.ExitUnreportable) && len(a.Pending) == 0
}

// releaseRecoveredCheckouts resets checkout locks still marked with this
// run's launch intent once its terminal state needs no further report. It runs
// only for runs without a live handle in this process.
func (e *Engine) releaseRecoveredCheckouts() {
	type candidate struct{ id, directory string }
	var todo []candidate
	e.mu.Lock()
	pending := map[string]bool{}
	for _, receipt := range e.journal.Pending {
		pending[receipt.RunID] = true
	}
	for id, run := range e.journal.Runs {
		a := run.Automation
		if a == nil || a.LockReleased || !a.NativeExited || !automationExitSettled(a) || pending[id] || e.active[id] != nil || run.State == "needs_recovery" {
			continue
		}
		todo = append(todo, candidate{id, e.config.Workspaces[run.Workspace].Path})
	}
	lockDir := e.workspaceLockDir
	e.mu.Unlock()
	for _, c := range todo {
		if c.directory == "" || resetWorkspaceIntent(c.directory, lockDir, c.id) != nil {
			continue
		}
		e.mu.Lock()
		if run := e.journal.Runs[c.id]; run != nil && run.Automation != nil {
			run.Automation.LockReleased = true
			_ = e.save()
		}
		e.mu.Unlock()
	}
}
