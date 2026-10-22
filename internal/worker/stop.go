package worker

import "errors"

func (e *Engine) stopOwned(id string, active *activeRun, commandID string) {
	if err := active.process.Stop(); err != nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.active[id] != active || active.uncertain {
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
					if active.process == child {
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
