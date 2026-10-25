package store

import (
	"agent-board/internal/core"
	"context"
	"strconv"
	"time"
)

func policyFor(b *board, id string) (*core.AutomationPolicy, error) {
	p := b.Automation.Policies[id]
	if p == nil {
		return nil, core.Fail("not_found", "Политика не найдена")
	}
	return p, nil
}
func policyPermission(ctx context.Context, b *board, a core.Actor, p *core.AutomationPolicy, now time.Time, approval bool) error {
	actor, err := automationHuman(ctx, b, a, now)
	if err != nil {
		return err
	}
	h := b.ManagedHosts[p.HostID]
	if h == nil || !managedControl(h, a.Principal.ID) || p.IssuerPrincipalID != "" && p.IssuerPrincipalID != a.Principal.ID {
		return core.Fail("forbidden", "Только текущий issuer/controller меняет политику")
	}
	local, hasLocal := ctx.Value(automationApprovalKey{}).(AutomationPolicyApproval)
	if approval || hasLocal && local.Source == "local" {
		return automationApproval(ctx, b, a, h, actor, now)
	}
	return nil
}
func (s *Store) AutomationPolicyCreate(ctx context.Context, a core.Actor, in core.AutomationPolicyCreateInput) (core.AutomationPolicy, error) {
	in.Limits = automationLimits(in.Limits)

	return automationMutate(s, ctx, a, in.RequestID, "policy_create", in, func(b *board, now time.Time) error {
		if err := policyPermission(ctx, b, a, &core.AutomationPolicy{AutomationPolicySettings: in.AutomationPolicySettings}, now, true); err != nil {
			return err
		}
		return automationScopeCurrent(b.ManagedHosts[in.HostID], in.Workspace, in.HostConsentVersion, now)
	}, func(b *board, now time.Time) (core.AutomationPolicy, error) {
		if in.ExpiresAt.IsZero() {
			in.ExpiresAt = now.Add(24 * time.Hour)
		}
		if len(b.Automation.Policies) >= 32 {
			return core.AutomationPolicy{}, core.Fail("capacity", "Достигнут предел политик")
		}
		if err := validateAutomationPolicy(ctx, b, in.AutomationPolicySettings, now); err != nil {
			return core.AutomationPolicy{}, err
		}
		id, err := managedID("apolicy_")
		if err != nil {
			return core.AutomationPolicy{}, err
		}
		auth, _ := AutomationAuthorityFromContext(ctx)
		h := b.ManagedHosts[in.HostID]
		p := core.AutomationPolicy{AutomationPolicySettings: in.AutomationPolicySettings, ID: id, Version: 1, State: "active", IssuerPrincipalID: a.Principal.ID, IssuerEpoch: auth.Principals[a.Principal.ID].Epoch, HostOwnerEpoch: h.Host.OwnerAccessEpoch, CreatedAt: now, UpdatedAt: now}
		b.Automation.Policies[id] = &p
		revision := p
		b.Automation.Revisions[id+"/1"] = &revision
		emit(b, now, "automation_policy_created", "", a.SessionID)
		return p, nil
	})
}
func (s *Store) AutomationPolicyUpdate(ctx context.Context, a core.Actor, in core.AutomationPolicyUpdateInput) (core.AutomationPolicy, error) {
	in.Limits = automationLimits(in.Limits)
	return automationMutate(s, ctx, a, in.RequestID, "policy_update", in, func(b *board, now time.Time) error {
		p, err := policyFor(b, in.PolicyID)
		if err != nil {
			return err
		}
		if p.HostID != in.HostID {
			return core.Fail("validation", "Обновление не переносит полномочия на другой host")
		}
		if err := policyPermission(ctx, b, a, p, now, true); err != nil {
			return err
		}
		return automationScopeCurrent(b.ManagedHosts[in.HostID], in.Workspace, in.HostConsentVersion, now)
	}, func(b *board, now time.Time) (core.AutomationPolicy, error) {
		p := b.Automation.Policies[in.PolicyID]
		if p.Version != in.Version {
			return core.AutomationPolicy{}, core.Fail("conflict", "Политика изменилась")
		}
		if p.Version >= 128 {
			return core.AutomationPolicy{}, core.Fail("capacity", "Достигнут предел ревизий политики")
		}
		if err := validateAutomationPolicy(ctx, b, in.AutomationPolicySettings, now); err != nil {
			return core.AutomationPolicy{}, err
		}
		p.AutomationPolicySettings = in.AutomationPolicySettings
		p.Version++
		p.State = "active"
		p.UpdatedAt = now
		auth, _ := AutomationAuthorityFromContext(ctx)
		p.IssuerEpoch = auth.Principals[a.Principal.ID].Epoch
		p.HostOwnerEpoch = b.ManagedHosts[p.HostID].Host.OwnerAccessEpoch
		copy := *p
		b.Automation.Revisions[p.ID+"/"+strconv.FormatInt(p.Version, 10)] = &copy
		invalidatePolicyWorkflows(b, p.ID, now)
		emit(b, now, "automation_policy_updated", "", a.SessionID)
		return *p, nil
	})
}
func (s *Store) AutomationPolicyRevoke(ctx context.Context, a core.Actor, in core.AutomationPolicyRevokeInput) (core.AutomationPolicy, error) {
	return automationMutate(s, ctx, a, in.RequestID, "policy_revoke", in, func(b *board, now time.Time) error {
		p, err := policyFor(b, in.PolicyID)
		if err != nil {
			return err
		}
		return policyPermission(ctx, b, a, p, now, false)
	}, func(b *board, now time.Time) (core.AutomationPolicy, error) {
		p := b.Automation.Policies[in.PolicyID]
		if p.Version != in.Version {
			return core.AutomationPolicy{}, core.Fail("conflict", "Политика изменилась")
		}
		if !bounded(in.Reason, 4096) {
			return core.AutomationPolicy{}, core.Fail("validation", "Нужна причина")
		}
		// A safety revoke remains possible at the update revision cap. Repeating
		// that already-revoked state must not grow immutable history indefinitely.
		if p.State == "revoked" {
			return *p, nil
		}
		p.State = "revoked"
		p.Version++
		p.UpdatedAt = now
		copy := *p
		b.Automation.Revisions[p.ID+"/"+strconv.FormatInt(p.Version, 10)] = &copy
		invalidatePolicyWorkflows(b, p.ID, now)
		return *p, nil
	})
}
func invalidatePolicyWorkflows(b *board, id string, now time.Time) {
	for _, w := range b.Automation.Workflows {
		if w.Grant != nil && w.Grant.PolicyID == id && !workflowTerminal(w.State) {
			w.State = "pausing"
			workflowReason(w, "authorization_changed", "")
			w.ControlEpoch++
			w.Version++
			w.UpdatedAt = now
			revokeWorkflowCapabilities(b, w.ID)
		}
	}
}
func (s *Store) AutomationPolicyGet(ctx context.Context, a core.Actor, in core.AutomationPolicyInput) (core.AutomationPolicyPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.AutomationPolicyPage, error) {
		var out core.AutomationPolicyPage
		if _, _, err := automationActor(ctx, b, a, now, false); err != nil {
			return out, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return out, err
		}
		p, err := policyFor(b, in.PolicyID)
		if err != nil {
			return out, err
		}
		revisions := []core.AutomationPolicy{}
		for _, r := range b.Automation.Revisions {
			if r.ID == p.ID && (in.Version == 0 || in.Version == r.Version) {
				revisions = append(revisions, *r)
			}
		}
		return automationPolicyPageWindow(*p, revisions, in, automationCursorNamespace(b))
	})
}
func (s *Store) AutomationPermission(ctx context.Context, a core.Actor, in core.AutomationPermissionInput) error {
	_, err := transact(s, ctx, func(b *board, now time.Time) (bool, error) {
		if in.Kind == "submit" && in.PolicyID == "" && in.WorkflowID == "" {
			sess, _, e := automationActor(ctx, b, a, now, true)
			if e == nil && sess.ManagedRunID != "" {
				return false, core.Fail("forbidden", "Managed child intake запрещён")
			}
			if _, bound := ExecutionIdentityFromContext(ctx); bound {
				return false, core.Fail("forbidden", "Child intake запрещён")
			}
			return true, e
		}
		if in.WorkflowID != "" {
			w, e := workflowFor(b, in.WorkflowID)
			if e != nil {
				return false, e
			}
			if e = workflowController(ctx, b, a, w, now); e != nil {
				return false, e
			}
			if in.Kind == "answer" {
				if _, e = automationHuman(ctx, b, a, now); e != nil {
					return false, e
				}
				q := b.Automation.Questions[in.QuestionID]
				if q == nil || q.WorkflowID != w.ID || !managedContains(q.DesignatedPrincipals, a.Principal.ID) {
					return false, core.Fail("forbidden", "Вопрос не назначен участнику")
				}
			}
			if in.RefreshPolicyID != "" {
				p, e := policyFor(b, in.RefreshPolicyID)
				if e != nil {
					return false, e
				}
				if p.Version != in.RefreshPolicyVersion {
					return false, core.Fail("conflict", "Политика изменилась")
				}
				if e = policyPermission(ctx, b, a, p, now, true); e != nil {
					return false, e
				}
				return true, validateAutomationPolicy(ctx, b, p.AutomationPolicySettings, now)
			}
			if in.CommandKind == "resume" || in.CommandKind == "replan" {
				return true, authorizeWorkflow(ctx, b, w, now)
			}
			return true, nil
		}
		if in.PolicyID != "" {
			p, e := policyFor(b, in.PolicyID)
			if e != nil {
				return false, e
			}
			if in.Kind == "submit" {
				return true, workflowIntakeAllowed(ctx, b, a, p, now)
			}
			if e := policyPermission(ctx, b, a, p, now, in.Kind != "revoke"); e != nil {
				return false, e
			}
			if in.Kind != "revoke" && in.Workspace != "" {
				return true, automationScopeCurrent(b.ManagedHosts[p.HostID], in.Workspace, 0, now)
			}
			return true, nil
		}
		_, e := automationHuman(ctx, b, a, now)
		if e != nil {
			return false, e
		}
		h, e := managedAllowed(b, a, in.HostID, "", false)
		if e != nil {
			return false, e
		}
		auth, _ := AutomationAuthorityFromContext(ctx)
		if e := automationApproval(ctx, b, a, h, auth.Principals[a.Principal.ID], now); e != nil {
			return false, e
		}
		if in.Workspace != "" {
			return true, automationScopeCurrent(h, in.Workspace, 0, now)
		}
		return true, nil
	})
	return err
}

func automationScopeCurrent(h *managedHostRecord, workspace string, version int64, now time.Time) error {
	if h == nil || h.Host.Automation == nil || !h.Host.Automation.Enabled || !h.Host.Automation.ExpiresAt.After(now) || version != 0 && h.Host.Automation.Version != version {
		return core.Fail("forbidden", "Local consent изменился/истёк")
	}
	scope, ok := h.Host.Automation.Scopes[workspace]
	if !ok || !scope.ExpiresAt.After(now) {
		return core.Fail("forbidden", "Workspace consent изменился/истёк")
	}
	return nil
}
