package worker

import (
	"context"
	"errors"
	"strings"
	"time"

	"agent-board/internal/core"
)

// Run only opens outbound HTTPS requests. A monotonic local watchdog stops the
// owned process group before the server's 30s ownership lease can expire.
func (e *Engine) Run(ctx context.Context) error {
	return e.run(ctx, 2*time.Second, 35*time.Second)
}

func (e *Engine) run(ctx context.Context, pollPeriod, acquisitionGrace time.Duration) error {
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
	e.mu.Lock()
	e.automationTransport = r.post
	e.automationReadTransport = r.request
	e.asyncApply = true
	e.mu.Unlock()
	running, cancel := context.WithCancel(ctx)
	defer cancel()
	// Proof delivery outlives the poll loop: on shutdown owned processes are
	// stopped first, then pending reports and exit proofs are flushed within a
	// bounded budget while the lease is still valid, and only then delivery
	// stops and the caller closes the journal.
	delivering, stopDelivery := context.WithCancel(context.Background())
	deliveryDone := make(chan struct{})
	go func() { defer close(deliveryDone); e.automationDelivery(delivering) }()
	watchDone := make(chan struct{})
	go func() { defer close(watchDone); e.watchdog(running) }()
	defer func() {
		cancel()
		<-watchDone
		_ = e.StopAll("worker_shutdown: local worker is stopping")
		e.drainOnShutdown(r.post, 10*time.Second)
		stopDelivery()
		<-deliveryDone
	}()
	ticker := time.NewTicker(pollPeriod)
	defer ticker.Stop()
	lastHeartbeat := time.Now()
	acquireUntil := time.Now().Add(acquisitionGrace)
	acquired := false
	conflictStreak := 0
	for {
		blocked, recoveryErr := e.recoverExpiredReports(running, r)
		if recoveryErr != nil {
			return recoveryErr
		}
		if blocked {
			select {
			case <-running.Done():
				return running.Err()
			case <-ticker.C:
				continue
			}
		}
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
			acquired = true
			conflictStreak = 0
			if result.Command != nil {
				if result.Run == nil {
					return errors.New("managed dispatch has no run")
				}
				if err = e.Accept(*result.Command, *result.Run); errors.Is(err, errRunQuarantined) {
					e.noticeRecovery()
				} else if err != nil {
					return err
				}
			}
		} else {
			var rejected statusError
			if errors.As(err, &rejected) && (rejected.code == 401 || rejected.code == 403 || rejected.code == 409) {
				switch {
				case rejected.code == 409 && !acquired && time.Now().Before(acquireUntil) && e.canWaitForAcquisition():
					// A newly opened local incarnation may wait for the previous
					// host lease to expire. No authority or native work exists here.
				case rejected.code == 409 && e.idleWithoutAuthority() && (e.hasLocalRecovery() || recoveryConflict(rejected)):
					// Accord refuses the lease while it still has uncertain
					// work of this host (a needs_recovery run awaiting `worker
					// recover`, or reconciliation after a lease lapse). Nothing
					// runs here, so the worker waits and keeps polling instead
					// of exiting; it never launches without a granted lease.
					conflictStreak++
					if conflictStreak == 1 || conflictStreak%30 == 0 {
						e.logf("Accord refused the worker lease (%s); no local agent is running. Waiting and retrying; if a run needs recovery, run `agent-board worker recover --run <id>`", rejected.Error())
					}
					e.noticeRecovery()
					if waitErr := waitBackoff(running, pollPeriod, conflictStreak); waitErr != nil {
						return waitErr
					}
					continue
				default:
					return err
				}
			}
		}
		if err = e.sendPendingReports(running, r.post, 8); err != nil {
			return err
		}
		e.noticeRecovery()
		if time.Since(lastHeartbeat) > 5*time.Second {
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

// idleWithoutAuthority reports that no native process or launch plan exists
// here, so a refused lease cannot leave anything running unsupervised.
func (e *Engine) idleWithoutAuthority() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.active) != 0 || len(e.plans) != 0 {
		return false
	}
	for _, run := range e.journal.Runs {
		if run.State == "accepted" || run.State == "launching" || run.State == "running" || run.State == "stopping" {
			return false
		}
	}
	return true
}

// hasLocalRecovery reports a needs_recovery run in the private journal.
func (e *Engine) hasLocalRecovery() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, run := range e.journal.Runs {
		if run.State == "needs_recovery" {
			return true
		}
	}
	return false
}

// recoveryConflict recognises Accord's refusal of the lease while this host
// still has uncertain work that needs local reconciliation or recovery.
func recoveryConflict(rejected statusError) bool {
	message := strings.ToLower(rejected.message)
	return strings.EqualFold(rejected.server, "managed_uncertain") || strings.Contains(message, "recovery") || strings.Contains(message, "сверк")
}

// waitBackoff waits pollPeriod doubled per consecutive refusal, capped at 30s.
func waitBackoff(ctx context.Context, period time.Duration, streak int) error {
	wait := period
	for i := 1; i < streak && wait < 30*time.Second; i++ {
		wait *= 2
	}
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

func (e *Engine) canWaitForAcquisition() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.active) != 0 || len(e.journal.Pending) != 0 {
		return false
	}
	// A needs_recovery run has no process handle here; waiting for the
	// previous incarnation's lease grants nothing and launches nothing.
	for _, run := range e.journal.Runs {
		if run.State != "stopped" && run.State != "completed" && run.State != "failed" && run.State != "needs_recovery" {
			return false
		}
	}
	return true
}

// Every poll has a fresh random request ID, so it cannot replay a cached lease.
// The server contract grants 30s; reserving 5s covers stop grace and network lag.
// Absolute server clocks are deliberately not used for local process safety.
func pollDeadline(requestStart time.Time) time.Time { return requestStart.Add(25 * time.Second) }

func (e *Engine) pollInput(requestID string) core.ManagedPollInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return core.ManagedPollInput{RequestID: requestID, HostID: e.journal.HostID, HostSecret: e.journal.HostSecret, WorkerID: e.journal.WorkerID, WorkerEpoch: e.journal.Epoch, WorkerCapabilities: automationCapabilities(e.config)}
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
		for id, active := range e.active {
			if active.automation != nil && !active.stop && !active.automation.deadline.IsZero() && !time.Now().Before(active.automation.deadline) {
				// The local deadline stop is explicit: its exit becomes the
				// live worker's terminal recovered_stopped, never ambiguity.
				active.stop = true
				active.stopReason = "local_deadline"
				if run := e.journal.Runs[id]; run != nil {
					run.LocalDiagnostic = "local_deadline_stop"
					run.Error = "local_deadline: the worker-local attempt deadline was reached; the native turn was stopped"
				}
				active.automation.cancel()
				go e.stopOwnedByDeadline(active)
			}
		}
		expired := !e.deadline.IsZero() && !time.Now().Before(e.deadline) && len(e.active) > 0
		fatal := e.fatal != nil && len(e.active) > 0
		e.mu.Unlock()
		if expired || fatal {
			_ = e.StopAll("worker ownership uncertain; local process tree stopped")
		}
	}
}
