package worker

import (
	"context"
	"errors"
	"time"

	"agent-board/internal/core"
)

func recoveryPhase(phase string) bool {
	return phase == "recovered_stopped" || phase == "delivery_unknown"
}

// An expired poll cannot renew authority until the server learns about locally
// stopped/uncertain work. Network retries preserve the durable recovery payload;
// acknowledging recovery never applies an accepted launch or resume plan.
func (e *Engine) recoverExpiredReports(ctx context.Context, r *remote) (bool, error) {
	e.mu.Lock()
	needed := !e.deadline.IsZero() && !time.Now().Before(e.deadline) && len(e.active) == 0 && len(e.journal.Pending) > 0
	e.mu.Unlock()
	if !needed {
		return false, nil
	}
	if err := e.prepareReconciliation(); err != nil {
		return true, err
	}
	if !r.sessionSnapshot().ExpiresAt.After(time.Now().Add(5 * time.Second)) {
		if err := r.open(ctx, e.config, e.journal.WorkerID); err != nil {
			return true, recoveryRequestError(err)
		}
	}
	for receipt := e.Pending(); receipt != nil; receipt = e.Pending() {
		if !recoveryPhase(receipt.Phase) || receipt.RecoveryWorkerID != e.journal.WorkerID {
			return true, errors.New("unsafe local recovery receipt; execution remains quarantined")
		}
		input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error, Forced: receipt.Forced}
		var response core.ManagedResult
		if err := r.post(ctx, "/managed/report", input, &response); err != nil {
			if dropped, dropErr := e.dropSettledRecovery(*receipt, err); dropped {
				if dropErr != nil {
					return true, dropErr
				}
				continue
			}
			return true, recoveryRequestError(err)
		}
		if err := e.Ack(*receipt); err != nil {
			return true, err
		}
	}
	e.releaseRecoveredCheckouts()
	e.noticeRecovery()
	return false, nil
}

func recoveryRequestError(err error) error {
	var rejected statusError
	if errors.As(err, &rejected) && rejected.code >= 400 && rejected.code < 500 {
		return err
	}
	return nil
}
