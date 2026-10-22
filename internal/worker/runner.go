package worker

import (
	"context"
	"errors"
	"time"

	"agent-board/internal/core"
)

// Run only opens outbound HTTPS requests. A monotonic local watchdog stops the
// owned tree before the server's 30s ownership lease can expire.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.probe(e.config); err != nil {
		return err
	}
	r, err := newRemote(e.config)
	if err != nil {
		return err
	}
	if e.journal.HostID == "" {
		return errors.New("enroll this local worker before running it")
	}
	if err = r.open(ctx, e.config, e.journal.WorkerID); err != nil {
		return err
	}
	if err = e.reconcile(ctx, r); err != nil {
		return err
	}
	running, cancel := context.WithCancel(ctx)
	defer cancel()
	watchDone := make(chan struct{})
	go func() { defer close(watchDone); e.watchdog(running) }()
	defer func() { cancel(); <-watchDone; _ = e.StopAll("worker connection is stopping") }()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastHeartbeat := time.Now()
	for {
		started := time.Now()
		input := e.pollInput(randomID("poll_"))
		var result core.ManagedPollResult
		err = r.post(running, "/managed/poll", input, &result)
		if err == nil {
			deadline := pollDeadline(started)
			if result.WorkerEpoch < 1 || result.LeaseExpiresAt.IsZero() || !deadline.After(time.Now()) {
				return errors.New("invalid worker ownership lease")
			}
			if err = e.SetLease(result.WorkerEpoch, deadline); err != nil {
				return err
			}
			if result.Command != nil {
				if result.Run == nil {
					return errors.New("managed dispatch has no run")
				}
				if err = e.Accept(*result.Command, *result.Run); err != nil {
					return err
				}
			}
		} else {
			var rejected statusError
			if errors.As(err, &rejected) && (rejected.code == 401 || rejected.code == 403 || rejected.code == 409) {
				return err
			}
		}
		for sent := 0; sent < 8; sent++ {
			receipt := e.Pending()
			if receipt == nil {
				break
			}
			input := core.ManagedReportInput{ManagedPollInput: e.pollInput(receipt.RequestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: receipt.RunID, Generation: receipt.Generation, Phase: receipt.Phase, ProviderSessionID: receipt.SessionID, Error: receipt.Error, Output: outputLines(*receipt), Forced: receipt.Forced}
			var response core.ManagedResult
			if err = r.post(running, "/managed/report", input, &response); err != nil {
				var rejected statusError
				if errors.As(err, &rejected) && rejected.code >= 400 && rejected.code < 500 {
					return err
				}
				break
			}
			if err = e.Ack(*receipt); err != nil {
				return err
			}
		}
		if time.Since(lastHeartbeat) > time.Minute {
			if err = r.heartbeat(running); err == nil {
				lastHeartbeat = time.Now()
			}
		}
		select {
		case <-running.Done():
			return running.Err()
		case <-ticker.C:
		}
	}
}

// Every poll has a fresh random request ID, so it cannot replay a cached lease.
// The server contract grants 30s; reserving 5s covers stop grace and network lag.
// Absolute server clocks are deliberately not used for local process safety.
func pollDeadline(requestStart time.Time) time.Time { return requestStart.Add(25 * time.Second) }

func (e *Engine) pollInput(requestID string) core.ManagedPollInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return core.ManagedPollInput{RequestID: requestID, HostID: e.journal.HostID, HostSecret: e.journal.HostSecret, WorkerID: e.journal.WorkerID, WorkerEpoch: e.journal.Epoch}
}

func (e *Engine) watchdog(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		e.mu.Lock()
		expired := !e.deadline.IsZero() && !time.Now().Before(e.deadline) && len(e.active) > 0
		fatal := e.fatal != nil && len(e.active) > 0
		e.mu.Unlock()
		if expired || fatal {
			_ = e.StopAll("worker ownership uncertain; local process tree stopped")
		}
	}
}
