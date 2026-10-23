package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strconv"
	"time"

	"agent-board/internal/core"
)

func (e *Engine) createCapabilityLocked(p plan, run *RunState) error {
	runDir := filepath.Join(e.config.StateDir, "runs", run.ID)
	if err := privateDirectory(runDir); err != nil {
		return err
	}
	attemptDir := filepath.Join(runDir, "attempt-"+strconv.FormatInt(run.Generation, 10))
	if err := privateDirectory(attemptDir); err != nil {
		return err
	}
	raw := core.ExecutionTokenPrefix + randomID("") + randomID("")
	digest := sha256.Sum256([]byte(raw))
	a := run.Automation
	a.CapabilityID = randomID("ec_")
	a.CapabilityDigest = hex.EncodeToString(digest[:])
	a.CapabilityRequestID = randomID("cap_")
	a.TokenFile = filepath.Join(attemptDir, "execution.token")
	if err := immutableBridgeFile(a.TokenFile, []byte(raw)); err != nil {
		return err
	}
	return e.save()
}
func (e *Engine) registerCapability(parent context.Context, p plan, a *automationAttempt) error {
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	binding := p.run.Automation
	input := core.ExecutionCapabilityRegisterInput{ManagedPollInput: e.pollInput(a.CapabilityRequestID), CommandID: p.command.ID, DispatchID: p.command.DispatchID, RunID: p.run.ID, Generation: p.run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, CapabilityID: a.CapabilityID, Digest: a.CapabilityDigest}
	for {
		var capability core.ExecutionCapability
		err := e.automationPost(ctx, "/automation/capabilities/register", input, &capability)
		if err == nil {
			if capability.ID != a.CapabilityID || capability.RunID != p.run.ID || capability.Generation != p.run.Generation || capability.WorkflowID != binding.WorkflowID || capability.StepID != binding.StepID || capability.WorkerEpoch != input.WorkerEpoch || capability.ControlEpoch != binding.ControlEpoch || capability.Role != p.run.Profile.Role || capability.ProfileID != p.run.Profile.ID || capability.ProfileVersion != p.run.Profile.Version || capability.State != "active" || !capability.ExpiresAt.After(time.Now()) || capability.ExpiresAt.After(binding.DeadlineAt) {
				return errors.New("invalid scoped execution registration")
			}
			return nil
		}
		var rejection statusError
		if errors.As(err, &rejection) && rejection.code >= 400 && rejection.code < 500 {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
