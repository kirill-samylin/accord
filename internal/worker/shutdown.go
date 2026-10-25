package worker

import (
	"errors"
	"time"
)

func (e *Engine) StopAll(reason string) error {
	e.mu.Lock()
	e.stopCalls++
	defer func() { e.mu.Lock(); e.stopCalls--; e.mu.Unlock() }()
	e.stopEpoch++
	var children []process
	for id, active := range e.active {
		if !active.stop && active.stopReason == "" {
			active.stopReason = "worker_stop"
		}
		active.stop = true
		if active.automation != nil {
			active.automation.cancel()
		}
		if !active.uncertain {
			e.journal.Runs[id].State = "stopping"
		}
		e.journal.Runs[id].Error = reason
		if active.process != nil {
			children = append(children, active.process)
		}
		if active.automation != nil {
			for _, check := range active.automation.checks {
				children = append(children, check)
			}
		}
	}
	err := e.save()
	e.mu.Unlock()
	err = errors.Join(err, e.stopBatch(children))
	done := make(chan struct{})
	go func() { e.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		err = errOwnership
	}
	return err
}

func (e *Engine) Close() error {
	e.mu.Lock()
	closed := e.closed
	if !closed {
		e.closing = true
	}
	e.mu.Unlock()
	if closed {
		return nil
	}
	err := e.StopAll("worker_shutdown: local worker is stopping")
	// StopAll bounds the ownership answer at 5s, but exit observers still fsync
	// the exit proof (Git/task observation is bounded at ~35s). Closing the
	// journal under them would turn a proven exit into needs_recovery.
	observed := make(chan struct{})
	go func() { e.wg.Wait(); close(observed) }()
	select {
	case <-observed:
	case <-time.After(40 * time.Second):
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.closed {
		for _, active := range e.active {
			if active.automation != nil && active.workspaceLock != nil {
				_ = active.workspaceLock.release(false)
				active.workspaceLock = nil
			}
		}
		e.closed = true
		e.disk.close()
	}
	return err
}
