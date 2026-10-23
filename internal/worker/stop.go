package worker

import "errors"

func (e *Engine) stopOwned(id string, active *activeRun, commandID string) {
	e.mu.Lock()
	if e.active[id] != active {
		e.mu.Unlock()
		return
	}
	var children []process
	if active.process != nil {
		children = append(children, active.process)
	}
	if active.automation != nil {
		active.automation.cancel()
		for _, check := range active.automation.checks {
			children = append(children, check)
		}
	}
	e.mu.Unlock()
	if err := e.stopBatch(children); err != nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.active[id] != active {
			return
		}
		active.uncertain = true
		run := e.journal.Runs[id]
		run.State = "needs_recovery"
		run.LocalDiagnostic = ownershipDiagnostic(err)
		if run.LocalDiagnostic == "" {
			run.LocalDiagnostic = "stop_unclassified_failure"
		}
		run.Error = "owned process tree termination could not be proven; local recovery required"
		e.reportFor(commandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
	}
}
func (e *Engine) stopOwnedByDeadline(active *activeRun) {
	e.mu.Lock()
	for id, current := range e.active {
		if current == active {
			command := e.journal.Runs[id].CommandID
			e.mu.Unlock()
			e.stopOwned(id, active, command)
			return
		}
	}
	e.mu.Unlock()
}

// All owned groups receive the failsafe together. Serial per-process grace
// periods would let later groups run beyond the shared ownership deadline.
func (e *Engine) stopBatch(children []process) error {
	results := make(chan error, len(children))
	for _, child := range children {
		go func(child process) {
			err := child.Stop()
			if err == nil {
				if waitErr := child.Wait(); errors.Is(waitErr, errOwnership) {
					err = waitErr
				}
			}
			if err != nil {
				e.mu.Lock()
				for id, active := range e.active {
					if active.process == child || ownsAutomationChild(active, child) {
						active.uncertain = true
						run := e.journal.Runs[id]
						run.State = "needs_recovery"
						run.LocalDiagnostic = ownershipDiagnostic(err)
						if run.LocalDiagnostic == "" {
							run.LocalDiagnostic = "stop_unclassified_failure"
						}
					}
				}
				_ = e.save()
				e.mu.Unlock()
			}
			results <- err
		}(child)
	}
	var failures []error
	for range children {
		if err := <-results; err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func ownsAutomationChild(active *activeRun, child process) bool {
	if active.automation != nil {
		for _, owned := range active.automation.checks {
			if owned == child {
				return true
			}
		}
	}
	return false
}
