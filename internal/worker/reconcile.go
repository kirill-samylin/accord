package worker

import (
	"context"
	"errors"
	"time"

	"agent-board/internal/core"
)

func (e *Engine) reconcile(ctx context.Context, r *remote) error {
	e.mu.Lock()
	var reports []*Receipt
	seen := map[string]bool{}
	for _, pending := range e.journal.Pending {
		if seen[pending.RunID] {
			continue
		}
		seen[pending.RunID] = true
		run := e.journal.Runs[pending.RunID]
		if run == nil {
			e.mu.Unlock()
			return errors.New("orphaned local receipt; manual recovery required")
		}
		copy := *pending
		copy.RequestID = randomID("recover_")
		copy.SessionID = run.SessionID
		if run.State == "stopped" || run.State == "completed" || run.State == "failed" {
			copy.Phase = "recovered_stopped"
		} else {
			copy.Phase = "delivery_unknown"
		}
		reports = append(reports, &copy)
	}
	for _, run := range e.journal.Runs {
		if run.State != "needs_recovery" || seen[run.ID] {
			continue
		}
		receipt := e.journal.Receipts[run.CommandID]
		if receipt == nil {
			e.mu.Unlock()
			return errors.New("missing launch identity; manual local recovery required")
		}
		copy := *receipt
		copy.Phase = "delivery_unknown"
		copy.RequestID = randomID("recover_")
		copy.SessionID = run.SessionID
		reports = append(reports, &copy)
	}
	e.journal.Pending = reports
	if err := e.save(); err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	deadline := time.Now().Add(35 * time.Second)
	for {
		receipt := e.Pending()
		if receipt == nil {
			break
		}
		input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error, Forced: receipt.Forced}
		var response core.ManagedResult
		if err := r.post(ctx, "/managed/report", input, &response); err != nil {
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
	for _, run := range e.journal.Runs {
		if run.State == "needs_recovery" {
			return errors.New("private journal requires local process recovery; automatic launch is disabled")
		}
	}
	return nil
}
