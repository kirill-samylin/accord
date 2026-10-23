package worker

import (
	"agent-board/internal/core"
	"errors"
)

// AckReport records the server's exact terminal receipt. A completed report can
// race a stop and leave the server in stopping; that is not a terminal ACK.
func (e *Engine) AckReport(receipt Receipt, result core.ManagedResult) error {
	e.mu.Lock()
	run := e.journal.Runs[receipt.RunID]
	automated := run != nil && run.Automation != nil
	e.mu.Unlock()
	if automated && terminalReceipt(receipt.Phase) {
		c, r := result.Command, result.Run
		if c.ID != receipt.CommandID || c.DispatchID != receipt.DispatchID || c.RunID != receipt.RunID || c.Generation != receipt.Generation || r.ID != receipt.RunID || r.Generation != receipt.Generation {
			return errors.New("managed terminal acknowledgement binding mismatch")
		}
		valid := false
		switch receipt.Phase {
		case "completed":
			valid = c.Status == "applied" && (r.State == "completed" || r.State == "stopping")
		case "failed":
			valid = c.Status == "failed" && r.State == "failed"
		case "stopped":
			valid = c.Status == "applied" && r.State == "stopped"
		case "recovered_stopped":
			valid = c.Status == "cancelled" && r.State == "stopped"
		}
		if !valid {
			return errors.New("managed terminal acknowledgement phase mismatch")
		}
		receipt.State = r.State
	}
	return e.Ack(receipt)
}
func terminalReceipt(phase string) bool {
	return phase == "completed" || phase == "failed" || phase == "stopped" || phase == "recovered_stopped"
}
func automationExitMatches(a *automationAttempt, receipt Receipt) bool {
	if receipt.State != "completed" && receipt.State != "failed" && receipt.State != "stopped" {
		return false
	}
	for _, proof := range a.Pending {
		if proof.Kind == "run_exit" && proof.CommandID == receipt.CommandID && proof.DispatchID == receipt.DispatchID && proof.RunID == receipt.RunID && proof.Generation == receipt.Generation {
			return true
		}
	}
	return false
}
func (e *Engine) bindAutomationStopProofLocked(run *RunState, stop *Receipt) {
	a := run.Automation
	if a == nil || a.ManagedAck {
		return
	}
	for _, proof := range a.Pending {
		if proof.Kind == "run_exit" {
			proof.RequestID = randomID("exit_stop_")
			proof.CommandID = stop.CommandID
			proof.DispatchID = stop.DispatchID
		}
	}
}
