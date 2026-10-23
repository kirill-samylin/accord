package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"agent-board/internal/automationipc"
)

func checkPayload(in automationipc.Request) string {
	in.Operation = ""
	in.Nonce = ""
	raw, _ := json.Marshal(in)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
func (e *Engine) automationCheckRequest(_ context.Context, id string, in automationipc.Request) (automationipc.Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, active := e.journal.Runs[id], e.active[id]
	if run == nil || run.Automation == nil || active == nil || active.automation == nil {
		return automationipc.Response{}, consentError()
	}
	a, rt := run.Automation, active.automation
	if e.closed || e.fatal != nil || active.stop || active.uncertain || a.NativeExited || rt.ctx.Err() != nil || !time.Now().Before(rt.deadline) || !time.Now().Before(e.deadline) {
		return automationipc.Response{}, consentError()
	}
	binding := a.Run.Automation
	if (in.Operation != "start" && in.Operation != "status") || binding == nil || in.TaskKey != a.Run.TaskKey || in.TaskGeneration != binding.TaskGeneration || in.SessionID == "" || !identifier.MatchString(in.RequestID) || !identifier.MatchString(in.PresetID) || in.WorkflowID != binding.WorkflowID || in.StepID != binding.StepID || in.ControlEpoch != binding.ControlEpoch || in.Generation != run.Generation || in.RunID != run.ID {
		return automationipc.Response{}, consentError()
	}
	if old := a.Checks[in.RequestID]; old != nil {
		if old.PayloadHash != checkPayload(in) {
			return automationipc.Response{}, errors.New("check request payload changed")
		}
		return checkResponse(old), nil
	}
	if in.Operation == "status" {
		return automationipc.Response{Status: "unknown"}, nil
	}
	if a.NativeExited || active.stop || rt.closing || rt.ctx.Err() != nil || active.process == nil || !time.Now().Before(rt.deadline) || !time.Now().Before(e.deadline) || run.SessionID == "" || !sessionID.MatchString(run.SessionID) || rt.checkCount > 0 || len(a.Checks) >= 64 {
		return automationipc.Response{}, consentError()
	}
	role := a.Run.Profile.Role
	if role != "tester" && role != "developer" {
		return automationipc.Response{}, consentError()
	}
	approved := false
	for _, preset := range binding.Checks {
		approved = approved || preset.PresetID == in.PresetID
	}
	if !approved {
		return automationipc.Response{}, consentError()
	}
	local := e.config.Automation.Workspaces[run.Workspace]
	preset, ok := local.Checks[in.PresetID]
	if !ok || !preset.Enabled {
		return automationipc.Response{}, consentError()
	}
	in.Nonce = ""
	check := &automationCheck{ID: randomID("check_"), PayloadHash: checkPayload(in), Request: in, Status: "queued", ReservedRequestID: randomID("verify_reserve_"), StartedRequestID: randomID("verify_start_"), FinishedRequestID: randomID("verify_finish_")}
	a.Checks[in.RequestID] = check
	rt.checkCount++
	if err := e.save(); err != nil {
		rt.checkCount--
		return automationipc.Response{}, err
	}
	e.wg.Add(1)
	go e.executeAutomationCheck(id, check, preset)
	return checkResponse(check), nil
}
func checkResponse(c *automationCheck) automationipc.Response {
	return automationipc.Response{CheckID: c.ID, Status: c.Status, Result: c.Result, Error: c.ErrorCode}
}
func (e *Engine) finishCheckCount(id string, rt *automationRuntime) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rt.checkCount--
	if rt.checkCount == 0 && rt.closing {
		select {
		case <-rt.checksDone:
		default:
			close(rt.checksDone)
		}
	}
}
func (e *Engine) failCheck(id string, c *automationCheck, code string, uncertain bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r, a := e.journal.Runs[id], e.active[id]
	if r == nil || a == nil {
		return
	}
	c.ErrorCode = code
	c.Status = "not_run"
	if uncertain {
		a.uncertain = true
		a.stop = true
		a.automation.cancel()
		r.State = "needs_recovery"
		r.Error = "verification ownership or delivery is uncertain; local recovery required"
		c.Status = "needs_recovery"
		go e.stopOwned(id, a, r.CommandID)
	}
	if err := e.save(); err != nil {
		a.automation.cancel()
	}
}
