package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func (s *Store) WorkflowQuestion(ctx context.Context, a core.Actor, in core.WorkflowQuestionInput) (core.WorkflowPage, error) {
	return automationMutate(s, ctx, a, in.RequestID, "workflow_question", in, func(b *board, now time.Time) error {
		_, _, err := workflowChild(ctx, b, a, in.WorkflowID, in.TaskGeneration, now)
		return err
	}, func(b *board, now time.Time) (core.WorkflowPage, error) {
		w := b.Automation.Workflows[in.WorkflowID]
		if w.Version != in.Version {
			return core.WorkflowPage{}, core.Fail("conflict", "Workflow изменился")
		}
		if !bounded(in.Question, 4096) || len(in.Options) > 8 {
			return core.WorkflowPage{}, core.Fail("validation", "Неверный вопрос")
		}
		for _, v := range in.Options {
			if !bounded(v, 512) {
				return core.WorkflowPage{}, core.Fail("validation", "Неверный вариант ответа")
			}
		}
		count := 0
		for _, q := range b.Automation.Questions {
			if q.WorkflowID == w.ID {
				count++
			}
		}
		if count >= 16 {
			return core.WorkflowPage{}, core.Fail("capacity", "Предел вопросов")
		}
		id, err := managedID("wq_")
		if err != nil {
			return core.WorkflowPage{}, err
		}
		task, err := s.createAsRoleInBoard(b, "coordinator", core.Actor{Principal: w.Requester}, core.CreateInput{Type: "task", EpicKey: w.EpicKey, Role: "human", Title: "Вопрос по " + w.ID, Description: managedClean(in.Question)}, now)
		if err != nil {
			return core.WorkflowPage{}, err
		}
		p := b.Automation.Policies[w.Grant.PolicyID]
		q := &core.WorkflowQuestionRecord{ID: id, WorkflowID: w.ID, StepID: w.CurrentStepID, Version: 1, HumanTaskKey: task.Key, DesignatedPrincipals: p.ControlPrincipals, State: "open", Question: managedClean(in.Question), Options: in.Options, CreatedAt: now}
		b.Automation.Questions[id] = q
		w.State = "awaiting_human"
		w.ControlEpoch++
		w.ReasonCode = "human_question"
		revokeWorkflowCapabilities(b, w.ID)
		if err = workflowStopOwned(ctx, b, w, now); err != nil {
			return core.WorkflowPage{}, err
		}
		if st := b.Automation.Steps[w.CurrentStepID]; st != nil {
			st.Status = "interrupted"
		}
		workflowTouch(w, now)
		return workflowPage(ctx, b, a, w, now)
	})
}
func (s *Store) WorkflowAnswer(ctx context.Context, a core.Actor, in core.WorkflowAnswerInput) (core.WorkflowPage, error) {
	check := func(b *board, now time.Time) error {
		if _, err := automationHuman(ctx, b, a, now); err != nil {
			return err
		}
		w, err := workflowFor(b, in.WorkflowID)
		if err != nil {
			return err
		}
		if err = workflowController(ctx, b, a, w, now); err != nil {
			return err
		}
		q := b.Automation.Questions[in.QuestionID]
		if q == nil || q.WorkflowID != w.ID || !managedContains(q.DesignatedPrincipals, a.Principal.ID) {
			return core.Fail("forbidden", "Вопрос не назначен участнику")
		}
		if in.Resume {
			return authorizeWorkflow(ctx, b, w, now)
		}
		return nil
	}
	return automationMutate(s, ctx, a, in.RequestID, "workflow_answer", in, check, func(b *board, now time.Time) (core.WorkflowPage, error) {
		w := b.Automation.Workflows[in.WorkflowID]
		q := b.Automation.Questions[in.QuestionID]
		if w.Version != in.Version || q.Version != in.QuestionVersion || q.State != "open" {
			return core.WorkflowPage{}, core.Fail("conflict", "Вопрос или workflow изменился")
		}
		if !bounded(in.Answer, 16384) {
			return core.WorkflowPage{}, core.Fail("validation", "Нужен ответ до16КиБ")
		}
		t := b.Tasks[q.HumanTaskKey]
		if t == nil {
			return core.WorkflowPage{}, core.Fail("not_found", "Human task не найден")
		}
		if t.Lease != nil && t.Lease.SessionID != a.SessionID {
			return core.WorkflowPage{}, core.Fail("lease_held", "Вопрос уже занят другим человеком")
		}
		if t.State != "ready" && t.State != "in_progress" {
			return core.WorkflowPage{}, core.Fail("conflict", "Вопрос уже завершён")
		}
		// This is an actual authenticated human claim + completion in one transaction.
		if t.Lease == nil {
			t.Generation++
			t.State = "in_progress"
			t.Lease = &core.Lease{SessionID: a.SessionID, Generation: t.Generation, ExpiresAt: now.Add(s.cfg.LeaseTTL)}
			t.Attempts = append(t.Attempts, core.Attempt{SessionID: a.SessionID, PrincipalID: a.Principal.ID, Machine: a.Principal.Machine, RunID: b.Sessions[a.SessionID].RunID, Generation: t.Generation, StartedAt: now})
			emit(b, now, "claimed", t.Key, a.SessionID)
		}
		evidence := core.Evidence{Answer: managedClean(in.Answer)}
		if err := completion(b, t, b.Sessions[a.SessionID], core.TransitionInput{Evidence: &evidence}); err != nil {
			return core.WorkflowPage{}, err
		}
		t.AuthorSession = a.SessionID
		finishAttempt(t, now, "completed", "")
		t.Lease = nil
		t.State = "done"
		t.Version++
		if t.FirstDoneAt == nil {
			t.FirstDoneAt = &now
		}
		q.State = "answered"
		q.Version++
		q.AnswerRef = t.Key
		w.Messages = appendBoundedWorkflowMessage(w.Messages, "Ответ человека "+q.ID+": "+evidence.Answer)
		if in.Resume {
			if workflowOwnedActive(b, w) {
				return core.WorkflowPage{}, core.Fail("conflict", "Сначала proven stop текущего процесса")
			}
			if err := authorizeWorkflow(ctx, b, w, now); err != nil {
				return core.WorkflowPage{}, core.Fail("forbidden", "Для resume нужно явное обновление делегации")
			}
			w.ControlEpoch++
			w.State = "queued"
			w.ReasonCode = ""
		} else {
			if workflowOwnedActive(b, w) {
				w.State = "pausing"
				w.ReasonCode = "human_answer_wait_stop"
			} else {
				w.State = "paused"
			}
		}
		workflowTouch(w, now)
		emit(b, now, "workflow_question_answered", t.Key, a.SessionID)
		return workflowPage(ctx, b, a, w, now)
	})
}
