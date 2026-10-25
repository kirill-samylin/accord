package worker

import (
	"context"
	"errors"
	"time"

	"agent-board/internal/core"
)

type postFunc func(context.Context, string, any, any) error

// sendPendingReports delivers up to limit durable managed reports in order. A
// transport failure keeps the head queued. A definitive 4xx stops the worker,
// except a 409 for a command this worker already reported terminally (see
// rejectedReport), which must not end service for other runs.
func (e *Engine) sendPendingReports(ctx context.Context, post postFunc, limit int) error {
	for sent := 0; sent < limit; sent++ {
		receipt := e.Pending()
		if receipt == nil {
			return nil
		}
		input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error, Output: outputLines(*receipt), Forced: receipt.Forced}
		var response core.ManagedResult
		err := post(ctx, "/managed/report", input, &response)
		if err != nil {
			var rejected statusError
			if !errors.As(err, &rejected) || rejected.code < 400 || rejected.code >= 500 {
				return nil
			}
			if handled, handleErr := e.forbiddenWhileStopping(*receipt, rejected); handled {
				if handleErr != nil {
					return handleErr
				}
				continue
			}
			if handled, handleErr := e.rejectedReport(*receipt, rejected); handled {
				if handleErr != nil {
					return handleErr
				}
				continue
			}
			return err
		}
		if err = e.AckReport(*receipt, response); err != nil {
			return err
		}
	}
	return nil
}

// rejectedReport resolves a 409 for a command whose local exit is already
// proven. A non-terminal or delivery_unknown report after a terminal one is
// obsolete and dropped. A terminal report the server refuses because the
// command is already terminal (for example a human pause rewrote it) is re-sent
// once as recovered_stopped, the one phase the server accepts for a terminal
// command; a repeated refusal is dropped. Nothing here creates authority.
func (e *Engine) rejectedReport(receipt Receipt, rejected statusError) (bool, error) {
	if rejected.code != 409 {
		return false, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 || e.journal.Pending[0].RequestID != receipt.RequestID {
		return true, errors.New("report receipt is not current")
	}
	stored, run := e.journal.Receipts[receipt.CommandID], e.journal.Runs[receipt.RunID]
	if stored == nil || run == nil || stored.TerminalPhase == "" || run.State == "needs_recovery" {
		return false, nil
	}
	head := e.journal.Pending[0]
	if terminalReceipt(receipt.Phase) && receipt.Phase != "recovered_stopped" && !head.Converted && (run.State == "stopped" || run.State == "completed" || run.State == "failed") {
		head.Phase, head.Converted = "recovered_stopped", true
		head.RequestID = randomID("report_")
		head.Output = ""
		if e.logger != nil {
			e.logger("Accord refused terminal report " + receipt.Phase + " for run " + receipt.RunID + " (" + rejected.Error() + "); re-sending it as recovered_stopped")
		}
		return true, e.save()
	}
	e.journal.Pending = e.journal.Pending[1:]
	if e.logger != nil {
		e.logger("Accord refused an obsolete " + receipt.Phase + " report for already terminal run " + receipt.RunID + " (" + rejected.Error() + "); it was dropped")
	}
	return true, e.save()
}

// forbiddenWhileStopping drops a non-terminal accepted/started report that
// Accord refuses with 403 (the command's initiator lost control permission)
// for a run this worker is already stopping. The refusal only means the server
// will not advance that command; the stop continues and its own terminal
// report follows. A start whose accepted receipt was refused is never
// launched. Any other 403 still ends the worker.
func (e *Engine) forbiddenWhileStopping(receipt Receipt, rejected statusError) (bool, error) {
	if rejected.code != 403 || (receipt.Phase != "accepted" && receipt.Phase != "started") {
		return false, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 || e.journal.Pending[0].RequestID != receipt.RequestID {
		return true, errors.New("report receipt is not current")
	}
	run := e.journal.Runs[receipt.RunID]
	if run == nil {
		return false, nil
	}
	active := e.active[receipt.RunID]
	stopping := active != nil && active.stop || run.State == "stopping" || run.State == "stopped" || run.State == "failed" || run.State == "completed"
	if !stopping {
		return false, nil
	}
	e.journal.Pending = e.journal.Pending[1:]
	if receipt.Phase == "accepted" {
		// Without an acknowledged accept the plan never applies.
		delete(e.plans, receipt.CommandID)
	}
	if e.logger != nil {
		e.logger("Accord refused the " + receipt.Phase + " report for run " + receipt.RunID + " while it is being stopped (" + rejected.Error() + "); the report was dropped and the stop continues")
	}
	return true, e.save()
}

// drainOnShutdown runs after StopAll. It waits for exit observers, then
// alternates managed reports and workflow proofs until nothing is owed or the
// budget ends. The parent context is already cancelled, so requests use their
// own bounded context; the server lease is still valid in this window.
func (e *Engine) drainOnShutdown(post postFunc, budget time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	observed := make(chan struct{})
	go func() { e.wg.Wait(); close(observed) }()
	for {
		done := false
		select {
		case <-observed:
			done = true
		default:
		}
		if err := e.sendPendingReports(ctx, post, 8); err != nil {
			return
		}
		_ = e.flushAutomation(ctx)
		if done && !e.owesDelivery() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (e *Engine) owesDelivery() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.fatal != nil {
		return false // Nothing can be persisted any more; restart reconciles.
	}
	if len(e.journal.Pending) > 0 {
		return true
	}
	for _, run := range e.journal.Runs {
		if a := run.Automation; a != nil && len(a.Pending) > 0 && (a.Pending[0].Kind != "run_exit" || a.ManagedAck) {
			return true
		}
	}
	return false
}
