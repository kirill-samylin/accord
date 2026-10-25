package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func workflowChild(ctx context.Context, b *board, a core.Actor, id string, generation int64, now time.Time) (*core.Workflow, *core.WorkflowStep, error) {
	ident, ok := ExecutionIdentityFromContext(ctx)
	if !ok {
		return nil, nil, core.Fail("forbidden", "Требуется scoped native attempt")
	}
	c := b.Automation.Capabilities[ident.Capability.ID]
	if err := executionCurrent(ctx, b, c, now); err != nil {
		return nil, nil, err
	}
	w := b.Automation.Workflows[c.WorkflowID]
	st := b.Automation.Steps[c.StepID]
	if id != w.ID {
		return nil, nil, core.Fail("forbidden", "Чужой workflow")
	}
	sess := b.Sessions[a.SessionID]
	if sess == nil || sess.ExecutionCapabilityID != c.ID {
		return nil, nil, core.Fail("forbidden", "Чужая native session")
	}
	t, err := owned(b, a, st.TaskKey, generation)
	if err != nil {
		return nil, nil, err
	}
	if t.Role != st.Kind {
		return nil, nil, core.Fail("forbidden", "Роль задачи изменилась")
	}
	return w, st, nil
}
func validateWorkflowPlan(w *core.Workflow, tasks []core.WorkflowPlanTask) error {
	raw, _ := json.Marshal(tasks)
	if len(raw) > 65536 {
		return core.Fail("validation", "План превышает64КиБ")
	}
	if len(tasks) < 1 || len(tasks) > w.Grant.Limits.PlanTasks {
		return core.Fail("validation", "План вне бюджета задач")
	}
	index := map[string]int{}
	checks := map[string]core.AutomationCheckPresetRef{}
	for _, c := range w.Grant.Checks {
		checks[c.PresetID] = c
	}
	for i, t := range tasks {
		if !bounded(t.ClientKey, 64) || strings.ContainsAny(t.ClientKey, "/\\\x00") || !validTitle(t.Title) || !bounded(t.Description, 12288) || len(t.Areas) > 32 || len(t.Acceptance) < 1 || len(t.Acceptance) > 16 || len(t.Prerequisites) > 16 || len(t.RequiredCheckIDs) < 1 || len(t.RequiredCheckIDs) > 32 {
			return core.Fail("validation", "Неверная структура задачи плана")
		}
		if _, ok := index[t.ClientKey]; ok {
			return core.Fail("validation", "Повтор client_key")
		}
		index[t.ClientKey] = i
		if err := validPlanAreas(t.Areas); err != nil {
			return err
		}
		for _, v := range t.Acceptance {
			if !bounded(v, 1024) {
				return core.Fail("validation", "Пустое или длинное условие плана")
			}
		}
		seen := map[string]bool{}
		test := false
		for _, id := range t.RequiredCheckIDs {
			c, ok := checks[id]
			if !ok || seen[id] {
				return core.Fail("forbidden", "План запросил неразрешённую проверку")
			}
			seen[id] = true
			test = test || c.Kind == "test" && c.Required
		}
		for _, c := range checks {
			if c.Required && !seen[c.PresetID] {
				return core.Fail("validation", "План пропустил обязательную проверку")
			}
		}
		if !test {
			return core.Fail("validation", "Каждая задача требует обязательный kind:test")
		}
	}
	// Require topological input order; this also excludes cycles and missing refs.
	for i, t := range tasks {
		seen := map[string]bool{}
		for _, dep := range t.Prerequisites {
			j, ok := index[dep]
			if !ok || j >= i || seen[dep] {
				return core.Fail("cycle", "Prerequisites должны ссылаться на предыдущие задачи без циклов")
			}
			seen[dep] = true
		}
	}
	return nil
}
func (s *Store) WorkflowPlan(ctx context.Context, a core.Actor, in core.WorkflowPlanInput) (core.WorkflowPage, error) {
	check := func(b *board, now time.Time) error {
		w, st, err := workflowChild(ctx, b, a, in.WorkflowID, in.TaskGeneration, now)
		if err != nil {
			return err
		}
		if st.Kind != "analyst" || w.State != "running" {
			return core.Fail("forbidden", "План принимает только текущий аналитик")
		}
		return nil
	}
	return automationMutate(s, ctx, a, in.RequestID, "workflow_plan", in, check, func(b *board, now time.Time) (core.WorkflowPage, error) {
		w := b.Automation.Workflows[in.WorkflowID]
		if w.Version != in.Version {
			return core.WorkflowPage{}, core.Fail("conflict", "Workflow изменился")
		}
		if w.PlanRevision >= 8 {
			return core.WorkflowPage{}, core.Fail("capacity", "Предел ревизий плана")
		}
		if err := validateWorkflowPlan(w, in.Tasks); err != nil {
			return core.WorkflowPage{}, err
		}
		w.PlanRevision++
		raw, _ := json.Marshal(in.Tasks)
		plan := &core.WorkflowPlanSnapshot{WorkflowID: w.ID, Revision: w.PlanRevision, Hash: managedHash(string(raw)), TaskGeneration: in.TaskGeneration, Tasks: in.Tasks, TaskKeys: map[string]string{}, SubmittedSessionID: a.SessionID, Status: "submitted", CreatedAt: now}
		b.Automation.Plans[w.ID+"/"+strconv.FormatInt(plan.Revision, 10)] = plan
		w.Version++
		w.UpdatedAt = now
		emit(b, now, "workflow_plan_submitted", b.Automation.Steps[w.CurrentStepID].TaskKey, a.SessionID)
		return workflowPage(ctx, b, a, w, now)
	})
}
func currentPlan(b *board, w *core.Workflow) *core.WorkflowPlanSnapshot {
	return b.Automation.Plans[w.ID+"/"+strconv.FormatInt(w.PlanRevision, 10)]
}
func (s *Store) materializePlan(b *board, w *core.Workflow, now time.Time) error {
	p := currentPlan(b, w)
	if p == nil || p.Status != "submitted" {
		return core.Fail("evidence_missing", "Нет native плана")
	}
	actor := core.Actor{Principal: w.Requester}
	for _, v := range p.Tasks {
		binding := w.Grant.RoleBindings["developer"]
		description := v.Description + "\nКритерии:\n" + strings.Join(v.Acceptance, "\n") + "\nОбязательные проверки: " + strings.Join(v.RequiredCheckIDs, ", ")
		t, err := s.createAsRoleInBoard(b, "coordinator", actor, core.CreateInput{Type: "task", EpicKey: w.EpicKey, Role: "developer", Title: v.Title, Description: description, Areas: v.Areas, ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion}, now)
		if err != nil {
			return err
		}
		p.TaskKeys[v.ClientKey] = t.Key
	}
	for _, v := range p.Tasks {
		t := b.Tasks[p.TaskKeys[v.ClientKey]]
		for _, dep := range v.Prerequisites {
			t.Links = append(t.Links, core.Link{From: p.TaskKeys[dep], To: t.Key, Type: "blocks", Condition: "completed"})
		}
	}
	p.Status = "materialized"
	w.Counters.Tasks += len(p.Tasks)
	return nil
}
