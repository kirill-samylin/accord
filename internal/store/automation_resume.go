package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

// workflowSettleStop advances pause/cancel/question/recovery states only from
// proven run states. A cancel requested while ownership is uncertain is kept in
// CancelRequested and completes (reservation released, workflow cancelled) once
// every owned run is terminal or reported recovered_stopped by the worker.
// Unchanged scans never bump the workflow version.
func workflowSettleStop(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	// Check-ownership recovery (check_unknown) stays explicit: only a cancel or
	// native-run uncertainty is settled automatically here.
	if w.State == "needs_recovery" && !w.CancelRequested && w.ReasonCode != "native_ownership_uncertain" {
		return nil
	}
	if err := workflowStopOwned(ctx, b, w, now); err != nil {
		return err
	}
	if workflowUncertain(b, w) {
		if w.State != "needs_recovery" || w.ReasonCode != "native_ownership_uncertain" {
			// Legacy "stopping" rows predate CancelRequested; remember the cancel.
			w.CancelRequested = w.CancelRequested || w.State == "stopping"
			w.State = "needs_recovery"
			workflowReason(w, "native_ownership_uncertain", workflowRunError(b, w, "needs_recovery"))
			workflowTouch(w, now)
		}
		return nil
	}
	if workflowOwnedActive(b, w) {
		return nil
	}
	switch {
	case w.CancelRequested || w.State == "stopping":
		workflowCancelled(b, w, now)
	case w.State == "pausing":
		w.State = "paused"
		workflowTouch(w, now)
	case w.State == "needs_recovery":
		// Every owned process is now proven terminal; continuing still needs
		// an explicit human resume with current authority.
		w.State = "paused"
		workflowReason(w, "native_recovered", "Все процессы workflow подтверждённо остановлены после локального recovery; продолжение — только явным resume")
		workflowTouch(w, now)
	}
	return nil
}
func workflowCancelled(b *board, w *core.Workflow, now time.Time) {
	w.State = "cancelled"
	w.CancelRequested = false
	cancelWorkflowTasks(b, w, now)
	w.ReservedRuns = 0
	delete(b.Automation.Reservations, w.ID)
	revokeWorkflowCapabilities(b, w.ID)
	workflowTouch(w, now)
}

// workflowResumeStep continues an interrupted step after an explicit resume.
func (s *Store) workflowResumeStep(ctx context.Context, b *board, w *core.Workflow, st *core.WorkflowStep, now time.Time) error {
	t := b.Tasks[st.TaskKey]
	if t == nil {
		return core.Fail("not_found", "Задача этапа отсутствует")
	}
	if t.State == "done" {
		return s.workflowCompleteInterrupted(ctx, b, w, st, t, now)
	}
	r := b.ManagedRuns[st.ManagedRunID]
	// Human waiting does not renew the deadline of a step that ran. A step that
	// was never dispatched produced no native work, so while the workflow
	// deadline is open it is replaced by a fresh step instead of a replan.
	if !st.DeadlineAt.After(now) {
		if r != nil && !managedActive(r.State) && workflowStepNeverDispatched(b, st, r) && w.DeadlineAt.After(now) {
			st.Status = "expired"
			if st.EndedAt == nil {
				st.EndedAt = &now
			}
			return s.workflowEnqueue(ctx, b, w, t, now)
		}
		return core.Fail("step_timeout", "Срок этапа истёк; нужны pause и явный replan")
	}
	if r == nil || managedActive(r.State) {
		return core.Fail("conflict", "Предыдущая native попытка не завершена")
	}
	if r.ProviderSessionID != "" {
		return workflowContinueTurn(ctx, b, w, st, r, now, false)
	}
	return s.workflowEnqueue(ctx, b, w, t, now)
}
func workflowStepNeverDispatched(b *board, st *core.WorkflowStep, r *core.ManagedRun) bool {
	if st.NativeSessionID != "" || r.ProviderSessionID != "" {
		return false
	}
	for _, c := range b.ManagedCommands {
		if c.Command.RunID == r.ID && c.Command.DispatchID != "" {
			return false
		}
	}
	return true
}

// A task completed before an interruption is observed exactly like
// workflowObserve would have: the same proof, generation and Git gates, and the
// same completion fields, so later gates (QA verification) see a completed step.
func (s *Store) workflowCompleteInterrupted(ctx context.Context, b *board, w *core.Workflow, st *core.WorkflowStep, t *core.Task, now time.Time) error {
	exit := workflowFindExit(b, st)
	if exit == nil || !workflowGitGate(t, exit) {
		return core.Fail("evidence_missing", "Resume не имеет подтверждённого exit/SHA")
	}
	if t.Generation != exit.TaskGeneration || t.AuthorSession == "" {
		return core.Fail("evidence_stale", "Task generation не совпадает с worker proof")
	}
	st.NativeSessionID = exit.NativeSessionID
	st.TaskGeneration = exit.TaskGeneration
	if !managedContains(st.EvidenceRefs, exit.ID) {
		st.EvidenceRefs = append(st.EvidenceRefs, exit.ID)
	}
	st.Status = "completed"
	st.EndedAt = &now
	revokeWorkflowCapabilities(b, w.ID)
	return s.workflowAdvanceAfter(ctx, b, w, st, now)
}

// workflowContinueTurn resumes the exact native conversation. Automatic
// continuations (turn ended without task completion) spend the per-step
// continuation budget; an explicit human resume does not. Invocations are
// counted when the worker reports started, not here.
func workflowContinueTurn(ctx context.Context, b *board, w *core.Workflow, st *core.WorkflowStep, r *core.ManagedRun, now time.Time, automatic bool) error {
	if w.Counters.Invocations >= w.Grant.Limits.NativeInvocations || w.ReservedRuns <= 0 || automatic && st.Continuations >= w.Grant.Limits.ContinuationPerStep {
		return core.Fail("budget_exhausted", "Бюджет continuation исчерпан")
	}
	if !st.DeadlineAt.After(now) {
		return core.Fail("step_timeout", "Срок этапа истёк; нужны pause и явный replan")
	}
	if !managedSessionID.MatchString(r.ProviderSessionID) || managedActive(r.State) {
		return core.Fail("conflict", "Exact native resume требует terminal attempt и native conversation")
	}
	if b.ManagedHosts[w.HostID].Host.WorkerCapabilities["automation"] < 1 {
		return core.Fail("worker_incompatible", "Worker не поддерживает automation:1")
	}
	profile, ok := w.Grant.Profiles[st.Kind]
	if !ok || profile.ID != st.Profile.ID || profile.Version != st.Profile.Version || w.Grant.RoleBindings[st.Kind].Provider != st.Provider {
		return core.Fail("conflict", "Resume не меняет pinned profile/provider; нужен explicit replan")
	}
	r.Automation.PolicyVersion = w.Grant.PolicyVersion
	r.Generation++
	r.State = "queued"
	r.Version++
	r.UpdatedAt = now
	r.Automation.ControlEpoch = w.ControlEpoch
	r.Automation.MessageCount = len(w.Messages)
	r.Automation.TaskGeneration = b.Tasks[st.TaskKey].Generation + 1
	st.ManagedGeneration = r.Generation
	if automatic {
		st.Continuations++
	}
	st.AttemptNo++
	st.EndedAt = nil
	st.Status = "queued"
	w.ReservedRuns--
	w.State = "running"
	revokeWorkflowCapabilities(b, w.ID)
	lead := "Продолжи назначенную задачу; предыдущий turn завершился без task completion. "
	if !automatic {
		lead = "Продолжи назначенную задачу после остановки и явного продолжения человеком. "
	}
	_, err := managedNewCommand(b, r, core.Actor{Principal: w.Requester}, "resume", lead+workflowPrompt(b, w, st), now, w.Grant.RequesterEpoch)
	if err == nil {
		workflowTouch(w, now)
	}
	return err
}
