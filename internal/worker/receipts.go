package worker

import "errors"

func (e *Engine) Ack(receipt Receipt) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 || e.journal.Pending[0].RequestID != receipt.RequestID {
		return errors.New("report receipt is not current")
	}
	head := e.journal.Pending[0]
	if head.CommandID != receipt.CommandID || head.RunID != receipt.RunID || head.Generation != receipt.Generation || head.DispatchID != receipt.DispatchID || head.Phase != receipt.Phase {
		return errors.New("report receipt is not current")
	}
	e.journal.Pending = e.journal.Pending[1:]
	if receipt.RecoveryWorkerID != "" && recoveryPhase(receipt.Phase) {
		if run := e.journal.Runs[receipt.RunID]; run != nil && run.Generation == receipt.Generation {
			run.RecoveryNotified = true
		}
	}
	if receipt.Phase == "completed" || receipt.Phase == "failed" || receipt.Phase == "stopped" || receipt.Phase == "recovered_stopped" {
		if stored := e.journal.Receipts[receipt.CommandID]; stored != nil {
			stored.Output = ""
		}
	}
	if err := e.save(); err != nil {
		return err
	}
	if receipt.Phase == "accepted" {
		if e.asyncApply {
			e.wg.Add(1)
			go func() {
				defer e.wg.Done()
				if e.beforeApply != nil {
					e.beforeApply()
				}
				e.mu.Lock()
				defer e.mu.Unlock()
				if !e.closed {
					if err := e.apply(receipt.CommandID); err != nil {
						e.fatal = err
					}
				}
			}()
			return nil
		}
		return e.apply(receipt.CommandID)
	}
	if receipt.Phase == "completed" || receipt.Phase == "failed" || receipt.Phase == "stopped" || receipt.Phase == "recovered_stopped" {
		if run := e.journal.Runs[receipt.RunID]; run != nil && run.Automation != nil {
			if automationExitMatches(run.Automation, receipt) {
				run.Automation.ManagedAck = true
			}
			return e.save()
		}
	}
	return nil
}

func (e *Engine) enqueue(receipt Receipt) {
	receipt.RequestID = randomID("report_")
	e.journal.Pending = append(e.journal.Pending, &receipt)
}

func (e *Engine) Pending() *Receipt {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 {
		return nil
	}
	copy := *e.journal.Pending[0]
	return &copy
}

// reportFor queues one report. After a terminal phase was queued for a command,
// delivery_unknown is never sent for it: a terminal report already states that
// the local process exit is proven.
func (e *Engine) reportFor(id, phase string, run *RunState, reason, output string) {
	receipt := e.journal.Receipts[id]
	if receipt == nil {
		return
	}
	if phase == "delivery_unknown" && receipt.TerminalPhase != "" {
		return
	}
	if terminalReceipt(phase) {
		if receipt.TerminalPhase == "" {
			receipt.TerminalPhase = phase
		}
		if run.Notice != "" {
			output = boundedText(run.Notice+"\n"+output, 16000)
		}
	}
	receipt.Phase, receipt.SessionID, receipt.Error, receipt.Output = phase, run.SessionID, reason, output
	e.enqueue(*receipt)
}
