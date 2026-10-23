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
