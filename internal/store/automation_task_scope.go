package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func automationTaskWorkflow(b *board, key string) *core.Workflow {
	if b.Automation == nil {
		return nil
	}
	for _, st := range b.Automation.Steps {
		if st.TaskKey == key {
			return b.Automation.Workflows[st.WorkflowID]
		}
	}
	for _, plan := range b.Automation.Plans {
		for _, task := range plan.TaskKeys {
			if task == key {
				return b.Automation.Workflows[plan.WorkflowID]
			}
		}
	}
	return nil
}
func automationTaskClaimAllowed(ctx context.Context, b *board, sess *core.Session, t *core.Task, now time.Time) bool {
	if q := automationQuestionForTask(b, t.Key); q != nil {
		return sess.Role == "human" && managedContains(q.DesignatedPrincipals, sess.PrincipalID) && q.State == "open"
	}
	w := automationTaskWorkflow(b, t.Key)
	if w == nil {
		return true
	}
	ident, ok := ExecutionIdentityFromContext(ctx)
	if !ok || sess.ExecutionCapabilityID != ident.Capability.ID {
		return false
	}
	st := b.Automation.Steps[w.CurrentStepID]
	return w.State == "running" && st != nil && st.TaskKey == t.Key && ident.Capability.StepID == st.ID && ident.Capability.ControlEpoch == w.ControlEpoch && executionCurrent(ctx, b, b.Automation.Capabilities[ident.Capability.ID], now) == nil
}
func automationTaskMutationGuard(ctx context.Context, b *board, a core.Actor, method string, input any, now time.Time) error {
	key := ""
	switch in := input.(type) {
	case core.UpdateInput:
		key = in.TaskKey
	case core.TransitionInput:
		key = in.TaskKey
	case core.LeaseInput:
		key = in.TaskKey
	case core.CommentInput:
		key = in.TaskKey
	case core.AssignInput:
		key = in.TaskKey
	case core.LinkInput:
		key = in.To
	}
	if q := automationQuestionForTask(b, key); q != nil {
		sess := b.Sessions[a.SessionID]
		if sess == nil || sess.Role != "human" || !managedContains(q.DesignatedPrincipals, a.Principal.ID) {
			return core.Fail("forbidden", "Вопрос назначен другому человеку")
		}
		return nil
	}
	if key == "" || automationTaskWorkflow(b, key) == nil {
		return nil
	}
	sess := b.Sessions[a.SessionID]
	if sess == nil || !automationTaskClaimAllowed(ctx, b, sess, b.Tasks[key], now) {
		return core.Fail("forbidden", "Workflow task изменяет только текущий scoped native holder; используйте workflow controls")
	}
	return nil
}

func automationQuestionForTask(b *board, key string) *core.WorkflowQuestionRecord {
	if b.Automation == nil {
		return nil
	}
	for _, q := range b.Automation.Questions {
		if q.HumanTaskKey == key {
			return q
		}
	}
	return nil
}
func automationHumanAnswered(b *board, t *core.Task, now time.Time) {
	q := automationQuestionForTask(b, t.Key)
	if q == nil || q.State != "open" {
		return
	}
	q.State = "answered"
	q.Version++
	q.AnswerRef = t.Key
	w := b.Automation.Workflows[q.WorkflowID]
	if w != nil {
		workflowAppendMessage(w, core.WorkflowMessageMeta{Source: core.WorkflowMessageHumanAnswer}, "Ответ человека "+q.ID+": "+t.Evidence.Answer)
		if !workflowOwnedActive(b, w) {
			w.State = "paused"
		} else {
			w.State = "pausing"
			workflowReason(w, "human_answer_wait_stop", "")
		}
		workflowTouch(w, now)
	}
}
func cancelWorkflowTasks(b *board, w *core.Workflow, now time.Time) {
	for _, t := range b.Tasks {
		if t.EpicKey == w.EpicKey && t.Lease == nil && (t.State == "ready" || t.State == "created") {
			t.State = "cancelled"
			t.Version++
			emit(b, now, "task_cancelled", t.Key, "")
		}
	}
	for _, q := range b.Automation.Questions {
		if q.WorkflowID == w.ID && q.State == "open" {
			q.State = "cancelled"
			q.Version++
		}
	}
	if epic := b.Tasks[w.EpicKey]; epic != nil && epic.State == "open" {
		epic.State = "cancelled"
		epic.Version++
	}
}

func automationRunStarted(b *board, r *core.ManagedRun, now time.Time) {
	if r.Automation == nil || b.Automation == nil {
		return
	}
	w := b.Automation.Workflows[r.Automation.WorkflowID]
	st := b.Automation.Steps[r.Automation.StepID]
	if w != nil {
		// The native invocation budget counts actual starts of this workflow,
		// including a late start of a fenced attempt; queued launches do not.
		w.Counters.Invocations++
		workflowTouch(w, now)
	}
	if w != nil && st != nil && w.State == "running" && w.CurrentStepID == st.ID && w.ControlEpoch == r.Automation.ControlEpoch {
		w.Counters.Started++
		st.Status = "running"
		st.NativeSessionID = r.ProviderSessionID
		st.DeliveredMessages = r.Automation.MessageCount
		workflowTouch(w, now)
	}
}

// A proven TASK013 terminal report closes every owned scope of this attempt.
// Unfinished check intents remain audit, never qualifying positive test proof.
func automationRunTerminal(b *board, r *core.ManagedRun, now time.Time) {
	if r.Automation == nil || b.Automation == nil {
		return
	}
	for _, c := range b.Automation.Capabilities {
		if c.RunID == r.ID && c.Generation == r.Generation {
			c.State = "revoked"
		}
	}
	for _, a := range b.Automation.Attestations {
		if a.RunID == r.ID && a.Generation == r.Generation && a.Kind == "check" && (a.Status == "reserved" || a.Status == "running") {
			a.Status = "cancelled"
			a.UpdatedAt = now
		}
	}
}

func automationTaskClaimed(b *board, t *core.Task, sess *core.Session) {
	if b.Automation == nil || sess.WorkflowStepID == "" {
		return
	}
	if st := b.Automation.Steps[sess.WorkflowStepID]; st != nil && st.TaskKey == t.Key && st.ManagedRunID == sess.ManagedRunID && st.ManagedGeneration == sess.ManagedGeneration {
		st.TaskGeneration = t.Generation
		st.TaskVersion = t.Version
	}
}
