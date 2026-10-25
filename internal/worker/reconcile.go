package worker

import (
	"context"
	"errors"
	"strings"
	"time"

	"agent-board/internal/core"
)

func (e *Engine) prepareReconciliation() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var reports []*Receipt
	seen := map[string]bool{}
	for _, pending := range e.journal.Pending {
		if seen[pending.RunID] {
			continue
		}
		run := e.journal.Runs[pending.RunID]
		if run == nil {
			return errors.New("orphaned local receipt; manual recovery required")
		}
		if pending.Generation != run.Generation {
			continue
		}
		seen[pending.RunID] = true
		copy := *pending
		if copy.RecoveryWorkerID == e.journal.WorkerID && recoveryPhase(copy.Phase) {
			reports = append(reports, &copy)
			continue
		}
		copy.RequestID = randomID("recover_")
		copy.RecoveryWorkerID = e.journal.WorkerID
		copy.SessionID = run.SessionID
		copy.Error = run.Error
		for _, latest := range e.journal.Receipts {
			if latest.RunID == run.ID && latest.Generation == run.Generation && latest.Forced {
				copy.Forced = true
			}
		}
		if run.State == "stopped" || run.State == "completed" || run.State == "failed" || e.journal.Receipts[copy.CommandID] != nil && e.journal.Receipts[copy.CommandID].TerminalPhase != "" {
			// A queued terminal report already proved the local exit.
			copy.Phase = "recovered_stopped"
		} else {
			copy.Phase = "delivery_unknown"
		}
		reports = append(reports, &copy)
	}
	for _, run := range e.journal.Runs {
		// An abandoned exit proof still owes the server one terminal stop.
		if seen[run.ID] || run.RecoveryNotified || run.Automation == nil || !run.Automation.ProofAbandoned || run.Automation.ExitAck || run.State != "stopped" {
			continue
		}
		receipt := e.journal.Receipts[run.CommandID]
		if receipt == nil || receipt.Generation != run.Generation {
			continue
		}
		seen[run.ID] = true
		copy := *receipt
		copy.Phase = "recovered_stopped"
		copy.RequestID = randomID("recover_")
		copy.RecoveryWorkerID = e.journal.WorkerID
		copy.SessionID = run.SessionID
		copy.Error = run.Error
		reports = append(reports, &copy)
	}
	for _, run := range e.journal.Runs {
		if run.State != "needs_recovery" || seen[run.ID] || run.RecoveryNotified {
			continue
		}
		receipt := e.journal.Receipts[run.CommandID]
		if receipt == nil || receipt.Generation != run.Generation {
			// Nothing can be reported for an unknown dispatch; the run stays
			// quarantined locally until `worker recover`.
			continue
		}
		copy := *receipt
		copy.Phase = "delivery_unknown"
		copy.RequestID = randomID("recover_")
		copy.RecoveryWorkerID = e.journal.WorkerID
		copy.SessionID = run.SessionID
		reports = append(reports, &copy)
	}
	e.journal.Pending = reports
	// Recovery acknowledgments cannot apply an earlier accepted launch plan.
	e.plans = map[string]plan{}
	return e.save()
}

func (e *Engine) reconcile(ctx context.Context, r *remote) error {
	if err := e.prepareReconciliation(); err != nil {
		return err
	}
	deadline := time.Now().Add(35 * time.Second)
	for {
		receipt := e.Pending()
		if receipt == nil {
			break
		}
		input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error, Forced: receipt.Forced}
		var response core.ManagedResult
		if err := r.post(ctx, "/managed/report", input, &response); err != nil {
			if dropped, dropErr := e.dropSettledRecovery(*receipt, err); dropped {
				if dropErr != nil {
					return dropErr
				}
				continue
			}
			var status statusError
			if !errors.As(err, &status) || status.code != 409 || time.Now().After(deadline) {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				continue
			}
		}
		if err := e.Ack(*receipt); err != nil {
			return err
		}
	}
	e.releaseRecoveredCheckouts()
	// A needs_recovery run quarantines only its own run/workspace (Accept
	// refuses that checkout); the worker keeps serving other work.
	e.noticeRecovery()
	return nil
}

// definitiveRecoveryRejection reports that Accord will never accept this
// recovery report: the run is gone (404, archived) or the report names an
// attempt the server no longer has (409 for a stale generation or dispatch, or
// a command that is already terminal). A 409 while another worker still holds
// the host lease is transient and is not matched here.
func definitiveRecoveryRejection(err error) bool {
	var status statusError
	if !errors.As(err, &status) {
		return false
	}
	if status.code == 404 {
		// Only Accord's own answer settles the report; a bare proxy 404 must
		// not drop a recovered_stopped for good.
		return strings.EqualFold(status.server, "not_found") || strings.EqualFold(status.server, "run_archived")
	}
	if status.code != 409 {
		return false
	}
	// The server upper-cases error codes on the wire (core.Fail).
	switch strings.ToLower(status.server) {
	case "stale_generation", "stale_dispatch", "run_archived", "not_found", "command_terminal":
		return true
	case "lease_held":
		return false
	}
	message := strings.ToLower(status.message)
	if strings.Contains(message, "lease") {
		return false
	}
	return strings.Contains(message, "generation") || strings.Contains(message, "dispatch") || strings.Contains(message, "terminal")
}

// dropSettledRecovery removes the head recovery report (recovered_stopped or
// delivery_unknown) after a definitive rejection. Nothing more can be told to
// Accord about that attempt, so the local run counts as notified; a
// needs_recovery run still stays quarantined until `worker recover`. It never
// creates authority and never drops a non-recovery phase.
func (e *Engine) dropSettledRecovery(receipt Receipt, err error) (bool, error) {
	if !recoveryPhase(receipt.Phase) || !definitiveRecoveryRejection(err) {
		return false, nil
	}
	e.mu.Lock()
	if len(e.journal.Pending) == 0 || e.journal.Pending[0].RequestID != receipt.RequestID {
		e.mu.Unlock()
		return true, errors.New("report receipt is not current")
	}
	e.journal.Pending = e.journal.Pending[1:]
	if run := e.journal.Runs[receipt.RunID]; run != nil && run.Generation == receipt.Generation {
		run.RecoveryNotified = true
	}
	saveErr := e.save()
	e.mu.Unlock()
	e.logf("Accord no longer accepts %s for run %s (%s); the local run is settled and the report was dropped", receipt.Phase, receipt.RunID, err.Error())
	return true, saveErr
}
