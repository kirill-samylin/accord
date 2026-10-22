package worker

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
		run.Error = "owned process tree termination could not be proven; local recovery required"
		e.reportFor(commandID, "delivery_unknown", run, run.Error, "")
		_ = e.save()
	}
}
