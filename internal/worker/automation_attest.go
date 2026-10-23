package worker

import (
	"context"
	"errors"
	"time"

	"agent-board/internal/core"
)

func (e *Engine) checkAttestation(id string, c *automationCheck, kind, requestID string) core.WorkflowAttestInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.journal.Runs[id]
	binding := r.Automation.Run.Automation
	preset := e.config.Automation.Workspaces[r.Workspace].Checks[c.Request.PresetID]
	receipt := e.journal.Receipts[r.CommandID]
	return core.WorkflowAttestInput{ManagedPollInput: e.pollInputLocked(requestID), CommandID: receipt.CommandID, DispatchID: receipt.DispatchID, RunID: r.ID, Generation: r.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, TaskKey: c.Request.TaskKey, TaskGeneration: c.Request.TaskGeneration, NativeSessionID: r.SessionID, Kind: kind, CheckRequestID: c.Request.RequestID, PresetID: c.Request.PresetID, PresetVersion: preset.Version, PresetDigest: preset.TrustedOwnerConsent, Observations: c.Observations}
}
func (e *Engine) pollInputLocked(id string) core.ManagedPollInput {
	return core.ManagedPollInput{RequestID: id, HostID: e.journal.HostID, HostSecret: e.journal.HostSecret, WorkerID: e.journal.WorkerID, WorkerEpoch: e.journal.Epoch, WorkerCapabilities: automationCapabilities(e.config)}
}
func (e *Engine) sendAttestation(ctx context.Context, in core.WorkflowAttestInput) (core.WorkflowAttestResult, error) {
	var out core.WorkflowAttestResult
	for {
		err := e.automationPost(ctx, "/automation/attest", in, &out)
		if err == nil {
			a := out.Attestation
			if a.RunID != in.RunID || a.Generation != in.Generation || a.WorkflowID != in.WorkflowID || a.StepID != in.StepID || a.ControlEpoch != in.ControlEpoch || a.TaskKey != in.TaskKey || a.TaskGeneration != in.TaskGeneration || a.NativeSessionID != in.NativeSessionID || !attestationEpochMatches(in.Kind, in.ControlEpoch, out.ControlEpoch) || a.CheckRequestID != in.CheckRequestID || a.PresetID != in.PresetID || a.PresetVersion != in.PresetVersion || a.PresetDigest != in.PresetDigest || !attestationPhaseMatches(in.Kind, a.Kind, a.Status) {
				return out, errors.New("attestation response binding mismatch")
			}
			return out, nil
		}
		var reject statusError
		if errors.As(err, &reject) && reject.code >= 400 && reject.code < 500 {
			return out, err
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func (e *Engine) queueAutomationAttestation(id string, in core.WorkflowAttestInput) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.journal.Runs[id]
	if r == nil || r.Automation == nil || len(r.Automation.Pending) >= 128 {
		return errors.New("automation proof queue unavailable")
	}
	r.Automation.Pending = append(r.Automation.Pending, &in)
	return e.save()
}

// Reports remain on disk until an exact idempotent acknowledgement. No failed
// save or lost HTTP reply is converted into a qualifying result.
func (e *Engine) flushAutomation(ctx context.Context) error {
	if !e.attestMu.TryLock() {
		return nil
	}
	defer e.attestMu.Unlock()
	for sent := 0; sent < 1; sent++ {
		e.mu.Lock()
		var input *core.WorkflowAttestInput
		var runID string
		for id, r := range e.journal.Runs {
			if r.Automation == nil || len(r.Automation.Pending) == 0 {
				continue
			}
			candidate := r.Automation.Pending[0]
			if candidate.Kind == "run_exit" && !r.Automation.ManagedAck {
				continue
			}
			copy := *candidate
			input = &copy
			runID = id
			break
		}
		e.mu.Unlock()
		if input == nil {
			return nil
		}
		bounded, cancel := context.WithTimeout(ctx, 4*time.Second)
		_, err := e.sendAttestation(bounded, *input)
		cancel()
		if err != nil {
			return err
		}
		e.mu.Lock()
		r := e.journal.Runs[runID]
		if r == nil || r.Automation == nil || len(r.Automation.Pending) == 0 || r.Automation.Pending[0].RequestID != input.RequestID {
			e.mu.Unlock()
			return errors.New("automation proof queue changed")
		}
		a := r.Automation
		a.Pending = a.Pending[1:]
		if input.Kind == "check_finished" {
			c := a.Checks[input.CheckRequestID]
			if c != nil {
				c.Status = "completed"
				if c.ErrorCode != "" {
					c.Status = "not_run"
				}
			}
		}
		if input.Kind == "run_exit" {
			a.ExitAck = true
		}
		err = e.save()
		e.mu.Unlock()
		if err != nil {
			return err
		}
		if input.Kind == "run_exit" {
			if err = e.releaseAutomationExit(runID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) automationDelivery(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		err := e.flushAutomation(ctx)
		if err != nil {
			var rejected statusError
			if errors.As(err, &rejected) && rejected.code >= 400 && rejected.code < 500 {
				e.mu.Lock()
				for _, a := range e.active {
					if a.automation != nil {
						a.uncertain = true
						a.automation.cancel()
					}
				}
				e.fatal = errors.New("automation proof rejected; local recovery required")
				e.mu.Unlock()
				_ = e.StopAll("automation authority lost")
				return
			}
		}
	}
}

func attestationPhaseMatches(input, kind, status string) bool {
	switch input {
	case "run_exit":
		return kind == "run_exit" && status == "completed"
	case "check_reserved":
		return kind == "check" && status == "reserved"
	case "check_started":
		return kind == "check" && status == "running"
	case "check_finished":
		return kind == "check" && status == "completed"
	case "check_unknown":
		return kind == "check" && status == "delivery_unknown"
	}
	return false
}

func attestationEpochMatches(kind string, bound, current int64) bool {
	// Terminal proof is bound to the original attempt; current is only workflow
	// metadata after a human control. It never renews this attempt's authority.
	if kind == "run_exit" || kind == "check_finished" || kind == "check_unknown" {
		return current >= bound
	}
	return current == bound
}
