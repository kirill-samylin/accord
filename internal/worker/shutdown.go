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
	err := e.StopAll("local worker is stopping")
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
