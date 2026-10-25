package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-board/internal/core"
)

// SetLogger receives owner-facing operational notes (never secrets or paths).
func (e *Engine) SetLogger(log func(string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = log
}

func (e *Engine) logf(format string, args ...any) {
	e.mu.Lock()
	log := e.logger
	e.mu.Unlock()
	if log != nil {
		log(fmt.Sprintf(format, args...))
	}
}

// noticeRecovery logs each needs_recovery run once per process. Such a run
// stays quarantined (its workspace refuses new launches) while other work runs.
func (e *Engine) noticeRecovery() {
	var ids []string
	unrecorded := map[string]bool{}
	e.mu.Lock()
	if e.noticed == nil {
		e.noticed = map[string]bool{}
	}
	for id, run := range e.journal.Runs {
		if run.State == "needs_recovery" && !e.noticed[id] {
			e.noticed[id] = true
			ids = append(ids, id)
			unrecorded[id] = recoveryNeedsGroups && len(run.ProcessGroups) == 0
		}
	}
	e.mu.Unlock()
	for _, id := range ids {
		flag := ""
		if unrecorded[id] {
			flag = " --unverified (no process groups were recorded for it)"
		}
		e.logf("local run %s needs recovery; its workspace is quarantined. Stop this worker, make sure no agent process of that run is still working, then run `agent-board worker recover --config <worker.json> --run %s`%s and start the worker again", id, id, flag)
	}
}

// RecoverOptions selects how `worker recover` verifies that no process of the
// run is alive.
type RecoverOptions struct {
	// Unverified allows recovery of a run without recorded process groups
	// (created by an older worker). The owner confirms that no agent of this
	// run is still working; the worker still scans local processes read-only
	// for native provider processes of this workspace or native session and
	// refuses on any match.
	Unverified bool
}

// errNoRecordedGroups refuses a verified recovery that would verify nothing.
var errNoRecordedGroups = errors.New("this run has no recorded process groups (it was started by an older worker), so nothing can be verified automatically; nothing was changed. Make sure no agent process of this run is still working, then re-run with --unverified")

// Recover is the verified `worker recover` (see RecoverRun).
func (e *Engine) Recover(ctx context.Context, runID string) error {
	return e.RecoverRun(ctx, runID, RecoverOptions{})
}

// RecoverRun is an explicit owner operation for one needs_recovery run. The
// caller holds the exclusive state lock (Open), so no other worker process of
// this state can own a process handle. It verifies by a read-only census that
// no recorded owned process group is alive (on Unix a run without recorded
// groups is refused unless opts.Unverified, which instead requires a clean
// read-only scan for native provider processes of this run), returns the
// checkout to idle only when the intent names this run, moves the run to
// stopped, and reports recovered_stopped. It never signals any process.
func (e *Engine) RecoverRun(ctx context.Context, runID string, opts RecoverOptions) error {
	e.mu.Lock()
	run := e.journal.Runs[runID]
	if e.journal.HostID == "" {
		e.mu.Unlock()
		return errors.New("enroll this local worker before recovering a run")
	}
	if run == nil {
		e.mu.Unlock()
		return errors.New("run is not known to this local worker")
	}
	if run.State != "needs_recovery" {
		state := run.State
		e.mu.Unlock()
		return fmt.Errorf("run is not awaiting local recovery (local state %s)", state)
	}
	receipt := e.journal.Receipts[run.CommandID]
	if receipt == nil || receipt.Generation != run.Generation {
		e.mu.Unlock()
		return errors.New("run has no reportable launch identity; it cannot be reconciled with Accord")
	}
	groups := append([]int{}, run.ProcessGroups...)
	census, scan := e.groupCensus, e.processScan
	directory := e.config.Workspaces[run.Workspace].Path
	sessionID := run.SessionID
	names := providerProcessNames(e.config)
	lockDir := e.workspaceLockDir
	e.mu.Unlock()
	if census == nil {
		census = processGroupsAlive
	}
	if scan == nil {
		scan = scanLocalProcesses
	}
	if len(groups) == 0 && recoveryNeedsGroups && !opts.Unverified {
		return errNoRecordedGroups
	}
	alive, err := census(groups)
	if err != nil {
		return errors.New("cannot verify local process groups of this run; nothing was changed")
	}
	if len(alive) > 0 {
		return fmt.Errorf("owned process groups %v of this run are still alive; stop them and retry (nothing was changed)", alive)
	}
	if opts.Unverified {
		processes, scanErr := scan(names)
		if scanErr != nil {
			return errors.New("cannot scan local processes to check for a still running agent; nothing was changed")
		}
		if err = refuseRunProcesses(processes, directory, sessionID, names); err != nil {
			return err
		}
	}
	if directory != "" {
		if err = resetWorkspaceIntent(directory, lockDir, runID); err != nil && !errors.Is(err, errForeignIntent) {
			return errors.New("checkout lock is held by another local worker; stop it and retry")
		}
	}
	e.mu.Lock()
	run.State = "stopped"
	run.LocalDiagnostic = "owner_recovered"
	run.Error = "owner_recovered: owner ran `agent-board worker recover`; no owned process group was alive"
	if opts.Unverified {
		run.LocalDiagnostic = "owner_recovered_unverified"
		run.Error = "owner_recovered_unverified: owner ran `agent-board worker recover --unverified` and confirmed no agent of this run is running; no recorded process group could be checked and no native provider process of this run was found"
	}
	if a := run.Automation; a != nil {
		// Undelivered proofs belong to a previous incarnation and are never
		// re-signed; the server receives only this terminal stop. The run is
		// settled (unreportable exit, released lock) across restarts.
		a.Pending = nil
		a.ExitUnreportable = true
		a.ProofAbandoned = true
		a.LockReleased = true
	}
	kept := e.journal.Pending[:0]
	for _, pending := range e.journal.Pending {
		if pending.RunID != runID {
			kept = append(kept, pending)
		}
	}
	report := *receipt
	report.Phase, report.Error, report.Output, report.SessionID = "recovered_stopped", run.Error, "", run.SessionID
	report.RequestID, report.RecoveryWorkerID = randomID("recover_"), e.journal.WorkerID
	e.journal.Pending = append([]*Receipt{&report}, kept...)
	if stored := e.journal.Receipts[run.CommandID]; stored != nil && stored.TerminalPhase == "" {
		// After this terminal stop delivery_unknown is never sent for it.
		stored.TerminalPhase = "recovered_stopped"
	}
	err = e.save()
	e.mu.Unlock()
	if err != nil {
		return err
	}
	return e.sendRecovery(ctx, report)
}

// sendRecovery delivers the durable recovered_stopped report. A 409 while the
// previous worker's 30s host lease is still live is retried; any failure leaves
// the report queued for the next `worker run`.
func (e *Engine) sendRecovery(ctx context.Context, receipt Receipt) error {
	r, err := newRemote(e.config)
	if err != nil {
		return err
	}
	if err = r.open(ctx, e.config, e.journal.WorkerID); err != nil {
		return fmt.Errorf("local run is stopped; the report stays queued for `worker run`: %w", err)
	}
	deadline := time.Now().Add(40 * time.Second)
	for {
		input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error}
		var response core.ManagedResult
		err = r.post(ctx, "/managed/report", input, &response)
		if err == nil {
			return e.Ack(receipt)
		}
		if dropped, dropErr := e.dropSettledRecovery(receipt, err); dropped {
			// Accord no longer has this attempt (archived run, newer
			// generation): the local stop is final and nothing is owed.
			return dropErr
		}
		var status statusError
		if !errors.As(err, &status) || status.code != 409 || time.Now().After(deadline) {
			return fmt.Errorf("local run is stopped; the report stays queued for `worker run`: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// groupRecorder persists process groups reported for runID. It locks mu, so it
// must be called from a goroutine that does not hold it.
func (e *Engine) groupRecorder(runID string) func(int) {
	return func(group int) {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.recordGroupLocked(runID, group)
	}
}

// recordSupervisorLocked records the helper group of a freshly started child.
func (e *Engine) recordSupervisorLocked(runID string, child process) {
	if owned, ok := child.(interface{ supervisorGroup() int }); ok {
		e.recordGroupLocked(runID, owned.supervisorGroup())
	}
}

func (e *Engine) recordGroupLocked(runID string, group int) {
	run := e.journal.Runs[runID]
	if run == nil || e.closed || group <= 1 {
		return
	}
	for _, known := range run.ProcessGroups {
		if known == group {
			return
		}
	}
	run.ProcessGroups = append(run.ProcessGroups, group)
	if len(run.ProcessGroups) > 64 {
		run.ProcessGroups = run.ProcessGroups[len(run.ProcessGroups)-64:]
	}
	_ = e.save()
}

// errRunQuarantined marks a dispatch for a needs_recovery run. It is left
// unacknowledged (nothing can be proven about that run) and the worker keeps
// serving other runs.
var errRunQuarantined = errors.New("local process identity is ambiguous; run `agent-board worker recover`")

// refuseQuarantinedStartLocked accepts and immediately fails a start for a
// checkout quarantined by another needs_recovery run. No process is started.
func (e *Engine) refuseQuarantinedStartLocked(command core.ManagedCommand, local *RunState, blocking string) error {
	local.State = "failed"
	local.LocalDiagnostic = "workspace_quarantined"
	local.Error = "workspace_quarantined: this checkout has an unresolved local run " + blocking + "; the owner runs `agent-board worker recover --run " + blocking + "` / рабочая папка заблокирована незавершённым запуском, нужен worker recover"
	receipt := &Receipt{RequestID: randomID("report_"), CommandID: command.ID, RunID: local.ID, DispatchID: command.DispatchID, Generation: command.Generation, Phase: "accepted"}
	e.journal.Receipts[command.ID] = receipt
	e.enqueue(*receipt)
	e.reportFor(command.ID, "failed", local, local.Error, "")
	return e.save()
}
