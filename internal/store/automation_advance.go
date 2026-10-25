package store

import (
	"agent-board/internal/core"
	"context"
	"sort"
	"time"
)

func (s *Store) AutomationCandidates(ctx context.Context, limit int) ([]core.WorkflowAdvanceInput, error) {
	if limit < 1 || limit > 32 {
		return nil, core.Fail("validation", "limit вне1..32")
	}
	return transact(s, ctx, func(b *board, now time.Time) ([]core.WorkflowAdvanceInput, error) {
		auth, ok := AutomationAuthorityFromContext(ctx)
		if !ok || auth.ProjectID == "" {
			return nil, core.Fail("forbidden", "Scheduler требует guarded authority")
		}
		out := []core.WorkflowAdvanceInput{}
		for _, w := range b.Automation.Workflows {
			if !workflowTerminal(w.State) && w.Grant != nil {
				out = append(out, core.WorkflowAdvanceInput{WorkflowID: w.ID, ExpectedVersion: w.Version, ControlEpoch: w.ControlEpoch})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].WorkflowID < out[j].WorkflowID })
		if len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	})
}
func (s *Store) AutomationTick(ctx context.Context, in core.WorkflowAdvanceInput) (core.WorkflowAdvanceResult, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.WorkflowAdvanceResult, error) {
		var zero core.WorkflowAdvanceResult
		auth, ok := AutomationAuthorityFromContext(ctx)
		if !ok || auth.ProjectID == "" {
			return zero, core.Fail("forbidden", "Scheduler требует guarded authority")
		}
		w, err := workflowFor(b, in.WorkflowID)
		if err != nil {
			return zero, err
		}
		if w.Version != in.ExpectedVersion || w.ControlEpoch != in.ControlEpoch {
			return zero, core.Fail("conflict", "Workflow version/epoch изменились")
		}
		if workflowTerminal(w.State) {
			return core.WorkflowAdvanceResult{Workflow: *w}, nil
		}
		candidate, err := automationClone(b)
		if err != nil {
			return zero, err
		}
		w = candidate.Automation.Workflows[w.ID]
		before := w.Version
		if err = authorizeWorkflow(ctx, candidate, w, now); err != nil && (w.State == "queued" || w.State == "running" || w.State == "awaiting_human") {
			w.State = "pausing"
			workflowReason(w, "authorization_changed", "")
			w.ControlEpoch++
			revokeWorkflowCapabilities(candidate, w.ID)
			workflowTouch(w, now)
		}
		if w.Grant != nil && !w.DeadlineAt.After(now) && (w.State == "queued" || w.State == "running") {
			w.State = "pausing"
			workflowReason(w, "workflow_timeout", "")
			w.ControlEpoch++
			revokeWorkflowCapabilities(candidate, w.ID)
			workflowTouch(w, now)
		}
		phaseBase, cloneErr := automationClone(candidate)
		if cloneErr != nil {
			return zero, cloneErr
		}
		phaseErr, stopErr := error(nil), error(nil)
		switch w.State {
		case "pausing", "stopping", "awaiting_human", "needs_recovery":
			stopErr = workflowSettleStop(ctx, candidate, w, now)
		case "queued":
			phaseErr = s.workflowStartOrContinue(ctx, candidate, w, now)
		case "running":
			phaseErr = s.workflowObserve(ctx, candidate, w, now)
		}
		if stopErr != nil {
			return zero, stopErr
		}
		if phaseErr != nil {
			candidate = phaseBase
			w = candidate.Automation.Workflows[in.WorkflowID]
			state, code, message := "blocked", automationErrorCode(phaseErr), automationErrorMessage(phaseErr)
			if w.State == "queued" && code == "CONFLICT" {
				state = "queued"
			}
			// A queued workflow waiting for a busy workspace retries every scan.
			// Unchanged facts are not a new version: human commands carrying the
			// current version must not race a no-op scheduler write.
			if state != w.State || code != w.ReasonCode || workflowReasonText(message) != w.ReasonMessage {
				w.State = state
				workflowReason(w, code, message)
				revokeWorkflowCapabilities(candidate, w.ID)
				workflowTouch(w, now)
			}
		}
		*b = *candidate
		return core.WorkflowAdvanceResult{Workflow: *w, NextDueAt: now.Add(time.Second), Changed: w.Version != before}, nil
	})
}
func automationErrorCode(err error) string {
	if d, ok := err.(*core.Error); ok {
		return d.Code
	}
	return "internal_error"
}
func (s *Store) workflowStartOrContinue(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	if err := reserveWorkflow(b, w); err != nil {
		return err
	}
	if w.EpicKey == "" {
		v, err := s.createAsRoleInBoard(b, "coordinator", core.Actor{Principal: w.Requester}, core.CreateInput{Type: "epic", Title: "Automation " + w.ID, Description: w.IntakeText}, now)
		if err != nil {
			return err
		}
		w.EpicKey = v.Key
	}
	if w.CurrentStepID != "" {
		st := b.Automation.Steps[w.CurrentStepID]
		if st != nil && st.Status == "interrupted" {
			return s.workflowResumeStep(ctx, b, w, st, now)
		}
	}
	if p := currentPlan(b, w); p != nil && p.Status == "materialized" {
		return s.workflowNextDeveloper(ctx, b, w, now)
	}
	t, err := s.workflowNewTask(b, w, "analyst", "Анализ требования "+w.ID, w.IntakeText, nil, now)
	if err != nil {
		return err
	}
	if err = s.workflowEnqueue(ctx, b, w, t, now); err == nil {
		workflowTouch(w, now)
	}
	return err
}
func (s *Store) workflowObserve(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	st := b.Automation.Steps[w.CurrentStepID]
	if st == nil {
		return core.Fail("evidence_missing", "Нет текущего этапа")
	}
	r := b.ManagedRuns[st.ManagedRunID]
	if r == nil {
		return core.Fail("evidence_missing", "Нет native запуска")
	}
	if managedActive(r.State) && r.State != "needs_recovery" && !st.DeadlineAt.After(now) {
		w.ControlEpoch++
		w.State = "pausing"
		workflowReason(w, "step_timeout", "")
		revokeWorkflowCapabilities(b, w.ID)
		workflowTouch(w, now)
		return workflowStopOwned(ctx, b, w, now)
	}
	if r.State == "starting" || r.State == "running" {
		return nil
	}
	if r.State == "needs_recovery" {
		w.State = "needs_recovery"
		workflowReason(w, "native_ownership_uncertain", r.LastError)
		workflowTouch(w, now)
		return nil
	}
	if managedActive(r.State) {
		return nil
	}
	if r.State != "completed" {
		st.Status = "interrupted"
		st.EndedAt = &now
		w.State = "paused"
		workflowReason(w, "native_"+r.State, r.LastError)
		revokeWorkflowCapabilities(b, w.ID)
		workflowTouch(w, now)
		return nil
	}
	exit := workflowFindExit(b, st)
	if exit == nil {
		st.Status = "awaiting_attestation"
		return nil
	}
	st.NativeSessionID = exit.NativeSessionID
	st.TaskGeneration = exit.TaskGeneration
	st.EvidenceRefs = append(st.EvidenceRefs, exit.ID)
	t := b.Tasks[st.TaskKey]
	if t != nil && t.State == "done" && workflowUndelivered(b, w, st) > 0 {
		st.Status = "interrupted"
		st.EndedAt = &now
		w.State = "paused"
		workflowReason(w, "clarification_after_completion", "")
		w.ControlEpoch++
		revokeWorkflowCapabilities(b, w.ID)
		workflowTouch(w, now)
		return nil
	}
	if t == nil || t.State != "done" {
		if st.Continuations < w.Grant.Limits.ContinuationPerStep && r.ProviderSessionID != "" {
			return workflowContinueTurn(ctx, b, w, st, r, now, true)
		}
		return core.Fail("evidence_missing", "Native turn не завершил назначенную задачу")
	}
	if t.Generation != exit.TaskGeneration || t.AuthorSession == "" {
		return core.Fail("evidence_stale", "Task generation не совпадает с worker proof")
	}
	if !workflowGitGate(t, exit) {
		return core.Fail("source_mismatch", "Worker не подтвердил точный чистый SHA")
	}
	st.Status = "completed"
	st.EndedAt = &now
	revokeWorkflowCapabilities(b, w.ID)
	return s.workflowAdvanceAfter(ctx, b, w, st, now)
}
