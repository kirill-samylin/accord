package store

import (
	"agent-board/internal/core"
	"context"
	"time"
	"unicode/utf8"
)

func workflowStopOwned(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	auth, _ := AutomationAuthorityFromContext(ctx)
	h := b.ManagedHosts[w.HostID]
	if h == nil {
		return nil
	}
	owner, ok := auth.Principals[h.Host.OwnerPrincipalID]
	if !ok {
		return nil
	} // Poll/deadman is the failsafe when registry access is gone.
	for _, r := range b.ManagedRuns {
		if r.Automation == nil || r.Automation.WorkflowID != w.ID || !managedActive(r.State) {
			continue
		}
		stopExists := false
		dispatched := false
		for _, c := range b.ManagedCommands {
			if c.Command.RunID != r.ID || c.Command.Generation != r.Generation {
				continue
			}
			if c.Command.Kind == "stop" && (c.Command.Status == "queued" || c.Command.Status == "accepted") {
				stopExists = true
			}
			if c.Command.DispatchID != "" {
				dispatched = true
			}
		}
		if !dispatched {
			for _, c := range b.ManagedCommands {
				if c.Command.RunID == r.ID && c.Command.Status == "queued" {
					c.Command.Status = "cancelled"
				}
			}
			r.State = "stopped"
			r.Version++
			r.UpdatedAt = now
			continue
		}
		if r.State == "needs_recovery" {
			continue
		}
		if !stopExists {
			if _, err := managedNewCommand(b, r, core.Actor{Principal: owner.Principal}, "stop", "", now, owner.Epoch); err != nil {
				return err
			}
		}
		// Repeated scans of an already stopping run are not a new run version.
		if !stopExists || r.State != "stopping" {
			r.State = "stopping"
			r.Version++
			r.UpdatedAt = now
		}
	}
	return nil
}
func workflowOwnedActive(b *board, w *core.Workflow) bool {
	for _, r := range b.ManagedRuns {
		if r.Automation != nil && r.Automation.WorkflowID == w.ID && managedActive(r.State) {
			return true
		}
	}
	return false
}
func workflowTouch(w *core.Workflow, now time.Time) { w.Version++; w.UpdatedAt = now }
func (s *Store) WorkflowCommand(ctx context.Context, a core.Actor, in core.WorkflowCommandInput) (core.WorkflowPage, error) {
	check := func(b *board, now time.Time) error {
		w, err := workflowFor(b, in.WorkflowID)
		if err != nil {
			return err
		}
		if w.Grant == nil && in.Kind == "resume" {
			if _, err = automationHuman(ctx, b, a, now); err != nil {
				return err
			}
			if w.Requester.ID != a.Principal.ID {
				return core.Fail("forbidden", "Только инициатор авторизует draft")
			}
		} else if err = workflowController(ctx, b, a, w, now); err != nil {
			return err
		}
		if in.RefreshPolicyID != "" {
			p, err := policyFor(b, in.RefreshPolicyID)
			if err != nil {
				return err
			}
			if err := policyPermission(ctx, b, a, p, now, true); err != nil {
				return err
			}
			return validateAutomationPolicy(ctx, b, p.AutomationPolicySettings, now)
		}
		if in.Kind == "resume" || in.Kind == "replan" {
			return authorizeWorkflow(ctx, b, w, now)
		}
		return nil
	}
	return automationMutate(s, ctx, a, in.RequestID, "workflow_command", in, check, func(b *board, now time.Time) (core.WorkflowPage, error) {
		w := b.Automation.Workflows[in.WorkflowID]
		if w.Version != in.Version {
			return core.WorkflowPage{}, core.Fail("conflict", "Workflow изменился")
		}
		if len(in.Text) > 4096 || !utf8.ValidString(in.Text) {
			return core.WorkflowPage{}, core.Fail("validation", "Текст ограничен4КиБ")
		}
		switch in.Kind {
		case "pause", "cancel":
			if workflowTerminal(w.State) {
				return core.WorkflowPage{}, core.Fail("conflict", "Workflow завершён")
			}
			w.ControlEpoch++
			revokeWorkflowCapabilities(b, w.ID)
			w.State = "pausing"
			workflowReason(w, "human_pause", "")
			if in.Kind == "cancel" || w.CancelRequested {
				// A cancel is never downgraded by a later pause; it completes
				// once every owned run is proven terminal (see workflowSettleStop).
				w.State = "stopping"
				w.CancelRequested = true
				workflowReason(w, "human_cancel", "")
			}
			if err := workflowStopOwned(ctx, b, w, now); err != nil {
				return core.WorkflowPage{}, err
			}
			if !workflowOwnedActive(b, w) {
				w.State = "paused"
				if w.CancelRequested {
					workflowCancelled(b, w, now)
				}
			}
		case "resume":
			if w.CancelRequested {
				return core.WorkflowPage{}, core.Fail("conflict", "Отмена уже запрошена; она завершится после подтверждённой остановки процессов")
			}
			if w.ReasonCode == "clarification_after_completion" {
				return core.WorkflowPage{}, core.Fail("conflict", "Нужен явный replan после уточнения уже завершённой задачи")
			}
			for _, q := range b.Automation.Questions {
				if q.WorkflowID == w.ID && q.State == "open" {
					return core.WorkflowPage{}, core.Fail("human_required", "Сначала ответьте на открытый вопрос")
				}
			}
			if w.State != "paused" && w.State != "awaiting_authorization" && w.State != "blocked" && w.State != "needs_recovery" {
				return core.WorkflowPage{}, core.Fail("conflict", "Сначала подтвердите остановку")
			}
			if workflowOwnedActive(b, w) {
				return core.WorkflowPage{}, core.Fail("conflict", "Сначала локальное recovery всех процессов")
			}
			if in.RefreshPolicyID != "" {
				p := b.Automation.Policies[in.RefreshPolicyID]
				if p.Version != in.RefreshPolicyVersion {
					return core.WorkflowPage{}, core.Fail("conflict", "Политика изменилась")
				}
				requester := w.Requester
				auth, _ := AutomationAuthorityFromContext(ctx)
				if _, ok := auth.Principals[requester.ID]; !ok {
					return core.WorkflowPage{}, core.Fail("forbidden", "Инициатор больше не активен")
				}
				g, err := workflowGrant(ctx, b, core.Actor{Principal: requester}, p, now)
				if err != nil {
					return core.WorkflowPage{}, err
				}
				if w.HostID != "" && (w.HostID != p.HostID || w.Workspace != p.Workspace) {
					return core.WorkflowPage{}, core.Fail("forbidden", "Resume не переносит checkout")
				}
				w.Grant = g
				w.HostID = p.HostID
				w.Workspace = p.Workspace
				w.DeadlineAt = now.Add(time.Duration(g.Limits.WorkflowSeconds) * time.Second)
			} else if err := authorizeWorkflow(ctx, b, w, now); err != nil {
				return core.WorkflowPage{}, core.Fail("forbidden", "Нужно явное обновление делегации человеком")
			}
			if err := reserveWorkflow(b, w); err != nil {
				return core.WorkflowPage{}, err
			}
			w.ControlEpoch++
			w.State = "queued"
			workflowReason(w, "", "")
			if st := b.Automation.Steps[w.CurrentStepID]; st != nil && st.Status != "completed" {
				st.Status = "interrupted"
			}
		case "clarify":
			if !bounded(in.Text, 4096) || workflowTerminal(w.State) {
				return core.WorkflowPage{}, core.Fail("validation", "Нужно уточнение активного workflow")
			}
			if len(w.Messages) >= 64 {
				return core.WorkflowPage{}, core.Fail("capacity", "Предел уточнений")
			}
			workflowAppendMessage(w, core.WorkflowMessageMeta{Source: core.WorkflowMessageHumanClarification}, in.Text)
		case "replan":
			if err := authorizeWorkflow(ctx, b, w, now); err != nil {
				return core.WorkflowPage{}, err
			}
			if !bounded(in.Text, 4096) {
				return core.WorkflowPage{}, core.Fail("validation", "Replan требует явного описания изменения и согласования прежних результатов")
			}
			for _, q := range b.Automation.Questions {
				if q.WorkflowID == w.ID && q.State == "open" {
					return core.WorkflowPage{}, core.Fail("conflict", "Сначала ответьте на открытые вопросы")
				}
			}
			if w.State != "paused" {
				return core.WorkflowPage{}, core.Fail("conflict", "Replan требует paused workflow")
			}
			if workflowOwnedActive(b, w) {
				return core.WorkflowPage{}, core.Fail("conflict", "Сначала pause и proven stop")
			}
			if w.PlanRevision >= 8 {
				return core.WorkflowPage{}, core.Fail("capacity", "Предел планов")
			}
			cancelWorkflowTasks(b, w, now)
			w.EpicKey = ""
			workflowAppendMessage(w, core.WorkflowMessageMeta{Source: core.WorkflowMessageHumanReplan}, "Согласованный человеком replan: "+in.Text)
			w.ControlEpoch++
			w.State = "queued"
			workflowReason(w, "replan", "")
			w.CurrentStepID = ""
			for _, p := range b.Automation.Plans {
				if p.WorkflowID == w.ID {
					p.Status = "superseded"
				}
			}
		default:
			return core.WorkflowPage{}, core.Fail("validation", "Допустимы pause/cancel/resume/clarify/replan")
		}
		workflowTouch(w, now)
		emit(b, now, "workflow_"+in.Kind, "", a.SessionID)
		return workflowPage(ctx, b, a, w, now)
	})
}

func workflowUncertain(b *board, w *core.Workflow) bool {
	for _, r := range b.ManagedRuns {
		if r.Automation != nil && r.Automation.WorkflowID == w.ID && r.State == "needs_recovery" {
			return true
		}
	}
	return false
}
