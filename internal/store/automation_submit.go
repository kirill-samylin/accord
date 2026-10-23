package store

import (
	"agent-board/internal/core"
	"context"
	"time"
	"unicode/utf8"
)

func workflowIntakeAllowed(ctx context.Context, b *board, a core.Actor, p *core.AutomationPolicy, now time.Time) error {
	sess, actor, err := automationActor(ctx, b, a, now, true)
	if err != nil {
		return err
	}
	if _, ok := ExecutionIdentityFromContext(ctx); ok || sess.ManagedRunID != "" {
		return core.Fail("forbidden", "Дочерние managed агенты не создают workflow")
	}
	if !p.ClientIntake {
		approval, ok := ctx.Value(automationIntakeKey{}).(AutomationIntakeApproval)
		if !ok || sess.Role != "human" || approval.PrincipalID != a.Principal.ID || approval.SessionID != a.SessionID || approval.ActionID == "" {
			return core.Fail("forbidden", "Client intake выключен; требуется trusted browser action")
		}
	}
	if p.State != "active" || !managedContains(p.IntakePrincipals, a.Principal.ID) {
		return core.Fail("forbidden", "Нет intake-разрешения политики")
	}
	if _, err = automationPrincipal(ctx, b, p.IssuerPrincipalID, p.IssuerEpoch, now); err != nil {
		return err
	}
	if actor.Epoch < 0 {
		return core.Fail("forbidden", "Нет access epoch")
	}
	return validateAutomationPolicy(ctx, b, p.AutomationPolicySettings, now)
}
func workflowGrant(ctx context.Context, b *board, a core.Actor, p *core.AutomationPolicy, now time.Time) (*core.WorkflowGrant, error) {
	authCheck, _ := AutomationAuthorityFromContext(ctx)
	requester, ok := authCheck.Principals[a.Principal.ID]
	if !ok || !managedContains(p.IntakePrincipals, a.Principal.ID) {
		return nil, core.Fail("forbidden", "Requester не разрешён политикой")
	}
	if _, err := automationPrincipal(ctx, b, a.Principal.ID, requester.Epoch, now); err != nil {
		return nil, err
	}
	if err := validateAutomationPolicy(ctx, b, p.AutomationPolicySettings, now); err != nil {
		return nil, err
	}
	auth, _ := AutomationAuthorityFromContext(ctx)
	h := b.ManagedHosts[p.HostID]
	expiry := now.Add(time.Duration(p.Limits.GrantSeconds) * time.Second)
	for _, deadline := range []time.Time{p.ExpiresAt, h.Host.Automation.ExpiresAt, h.Host.Automation.Scopes[p.Workspace].ExpiresAt} {
		if deadline.Before(expiry) {
			expiry = deadline
		}
	}
	g := &core.WorkflowGrant{PolicyID: p.ID, PolicyVersion: p.Version, IssuerID: p.IssuerPrincipalID, IssuerEpoch: p.IssuerEpoch, RequesterID: a.Principal.ID, RequesterEpoch: auth.Principals[a.Principal.ID].Epoch, HostOwnerID: h.Host.OwnerPrincipalID, HostOwnerEpoch: h.Host.OwnerAccessEpoch, HostConsentVersion: p.HostConsentVersion, RoleBindings: p.RoleBindings, Checks: p.Checks, Limits: p.Limits, Profiles: map[string]core.ProfileSnapshot{}, ApprovedAt: now, ExpiresAt: expiry}
	for role, v := range p.RoleBindings {
		profile, _ := managedSnapshot(b, core.ManagedStartInput{Provider: v.Provider, ProfileID: v.ProfileID, ProfileVersion: v.ProfileVersion})
		g.Profiles[role] = *profile
	}
	return g, nil
}
func workflowReservedRuns(b *board) int {
	n := 0
	for _, w := range b.Automation.Workflows {
		n += w.ReservedRuns
	}
	return n
}
func reserveWorkflowBudget(b *board, w *core.Workflow) error {
	need := w.Grant.Limits.NativeInvocations - w.Counters.Invocations
	if need < 0 {
		need = 0
	}
	if len(b.ManagedRuns)+workflowReservedRuns(b)-w.ReservedRuns+need > 128 {
		return core.Fail("capacity", "Недостаточно lifetime managed slots")
	}
	w.ReservedRuns = need
	return nil
}
func reserveWorkflow(b *board, w *core.Workflow) error {
	for _, r := range b.Automation.Reservations {
		if r.WorkflowID != w.ID && r.HostID == w.HostID && r.Workspace == w.Workspace {
			return core.Fail("conflict", "Workspace зарезервирован другим workflow")
		}
	}
	for _, r := range b.ManagedRuns {
		if r.HostID == w.HostID && r.Workspace == w.Workspace && managedActive(r.State) {
			return core.Fail("conflict", "Workspace занят managed процессом")
		}
	}
	if err := reserveWorkflowBudget(b, w); err != nil {
		return err
	}
	b.Automation.Reservations[w.ID] = &workflowReservation{WorkflowID: w.ID, HostID: w.HostID, Workspace: w.Workspace}
	return nil
}
func (s *Store) WorkflowSubmit(ctx context.Context, a core.Actor, in core.WorkflowSubmitInput) (core.WorkflowPage, error) {
	check := func(b *board, now time.Time) error {
		if _, ok := ExecutionIdentityFromContext(ctx); ok {
			return core.Fail("forbidden", "Child intake запрещён")
		}
		if in.PolicyID == "" {
			sess, _, err := automationActor(ctx, b, a, now, true)
			if err != nil {
				return err
			}
			if sess.ManagedRunID != "" {
				return core.Fail("forbidden", "Managed child intake запрещён")
			}
			return nil
		}
		p, err := policyFor(b, in.PolicyID)
		if err != nil {
			return err
		}
		return workflowIntakeAllowed(ctx, b, a, p, now)
	}
	return automationMutate(s, ctx, a, in.RequestID, "workflow_submit", in, check, func(b *board, now time.Time) (core.WorkflowPage, error) {
		if !bounded(in.Text, 16384) || !utf8.ValidString(in.Text) || len(b.Automation.Workflows) >= 32 {
			return core.WorkflowPage{}, core.Fail("validation", "Нужно требование до16КиБ и свободный workflow slot")
		}
		id, err := managedID("wf_")
		if err != nil {
			return core.WorkflowPage{}, err
		}
		w := &core.Workflow{ID: id, Version: 1, ControlEpoch: 1, State: "awaiting_authorization", IntakeText: managedClean(in.Text), IntakeHash: managedHash(in.Text), Requester: a.Principal, Counters: core.WorkflowCounters{FixPerTask: map[string]int{}}, CreatedAt: now, UpdatedAt: now}
		b.Automation.Workflows[id] = w
		if in.PolicyID != "" {
			p := b.Automation.Policies[in.PolicyID]
			if p.Version != in.PolicyVersion {
				return core.WorkflowPage{}, core.Fail("conflict", "Политика изменилась")
			}
			g, err := workflowGrant(ctx, b, a, p, now)
			if err != nil {
				return core.WorkflowPage{}, err
			}
			w.Grant = g
			w.HostID = p.HostID
			w.Workspace = p.Workspace
			w.DeadlineAt = now.Add(time.Duration(g.Limits.WorkflowSeconds) * time.Second)
			if err = reserveWorkflowBudget(b, w); err != nil {
				return core.WorkflowPage{}, err
			}
			w.State = "queued"
		}
		emit(b, now, "workflow_submitted", "", a.SessionID)
		return workflowPage(ctx, b, a, w, now)
	})
}
