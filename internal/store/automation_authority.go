package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func automationPrincipal(ctx context.Context, b *board, id string, epoch int64, now time.Time) (AutomationPrincipal, error) {
	auth, ok := AutomationAuthorityFromContext(ctx)
	if !ok || auth.ProjectID == "" {
		return AutomationPrincipal{}, core.Fail("forbidden", "Требуется текущая guarded authority проекта")
	}
	p, ok := auth.Principals[id]
	if !ok || p.Epoch != epoch || !p.ExpiresAt.IsZero() && !p.ExpiresAt.After(now) {
		return p, core.Fail("forbidden", "Участник или credential потерял доступ")
	}
	if p.LegacyTokenID != "" {
		var expiry int64
		var revoked int
		if b.tx.QueryRowContext(ctx, `SELECT expires,revoked FROM tokens WHERE id=?`, p.LegacyTokenID).Scan(&expiry, &revoked) != nil || revoked != 0 || expiry <= now.UnixNano() {
			return p, core.Fail("forbidden", "Legacy credential потерял доступ")
		}
	}
	return p, nil
}
func automationActor(ctx context.Context, b *board, a core.Actor, now time.Time, write bool) (*core.Session, AutomationPrincipal, error) {
	auth, ok := AutomationAuthorityFromContext(ctx)
	if !ok || write && !auth.Writable {
		return nil, AutomationPrincipal{}, core.Fail("forbidden", "Нет полномочий проекта")
	}
	p, ok := auth.Principals[a.Principal.ID]
	if !write && !ok {
		p, ok = auth.ReadPrincipals[a.Principal.ID]
	}
	if !ok {
		return nil, p, core.Fail("forbidden", "Участник не активен")
	}
	if write {
		if _, err := automationPrincipal(ctx, b, a.Principal.ID, p.Epoch, now); err != nil {
			return nil, p, err
		}
	} else if !p.ExpiresAt.IsZero() && !p.ExpiresAt.After(now) {
		return nil, p, core.Fail("forbidden", "Read credential истёк")
	}
	if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
		return nil, p, err
	}
	// A durable receipt may be read with its original expired session. A new
	// mutation checks sessionFor again after replay; authority is never renewed here.
	sess := b.Sessions[a.SessionID]
	if sess == nil || sess.PrincipalID != a.Principal.ID {
		return nil, p, core.Fail("unauthorized", "Session не принадлежит credential")
	}
	return sess, p, nil
}
func automationHuman(ctx context.Context, b *board, a core.Actor, now time.Time) (AutomationPrincipal, error) {
	sess, p, err := automationActor(ctx, b, a, now, true)
	if err != nil {
		return p, err
	}
	if sess.Role != "human" {
		return p, core.Fail("forbidden", "Требуется human-сессия")
	}
	if _, ok := ExecutionIdentityFromContext(ctx); ok {
		return p, core.Fail("forbidden", "Дочерняя попытка не управляет автоматизацией")
	}
	return p, nil
}
func automationApproval(ctx context.Context, b *board, a core.Actor, h *managedHostRecord, p AutomationPrincipal, now time.Time) error {
	if h == nil || h.Host.State != "active" || h.Host.Automation == nil || !h.Host.Automation.Enabled || !h.Host.Automation.ExpiresAt.After(now) {
		return core.Fail("forbidden", "Local automation consent недействителен")
	}
	if _, err := automationPrincipal(ctx, b, h.Host.OwnerPrincipalID, h.Host.OwnerAccessEpoch, now); err != nil {
		return err
	}
	approval, ok := ctx.Value(automationApprovalKey{}).(AutomationPolicyApproval)
	if !ok || approval.PrincipalID != a.Principal.ID || approval.SessionID != a.SessionID || approval.Epoch != p.Epoch || approval.HostID != h.Host.ID || h.Host.Automation == nil || approval.ConsentVersion != h.Host.Automation.Version {
		return core.Fail("forbidden", "Нужно явное подтверждение политики человеком")
	}
	switch approval.Source {
	case "web":
		if approval.ActionID == "" {
			return core.Fail("forbidden", "Нет browser action")
		}
	case "local":
		if !managedProof(h, approval.HostProof) || h.Host.OwnerPrincipalID != a.Principal.ID {
			return core.Fail("forbidden", "Нет локального подтверждения владельца")
		}
	default:
		return core.Fail("forbidden", "Неверное подтверждение политики")
	}
	return nil
}
func workflowFor(b *board, id string) (*core.Workflow, error) {
	if b.Automation == nil || b.Automation.Workflows[id] == nil {
		return nil, core.Fail("not_found", "Workflow не найден")
	}
	return b.Automation.Workflows[id], nil
}
func workflowTerminal(state string) bool { return state == "completed" || state == "cancelled" }
func workflowController(ctx context.Context, b *board, a core.Actor, w *core.Workflow, now time.Time) error {
	sess, _, err := automationActor(ctx, b, a, now, true)
	if err != nil {
		return err
	}
	if _, bound := ExecutionIdentityFromContext(ctx); bound || sess.ManagedRunID != "" {
		return core.Fail("forbidden", "Managed child не управляет workflow")
	}
	if w.Grant == nil {
		if w.Requester.ID == a.Principal.ID {
			return nil
		}
		return core.Fail("forbidden", "Только инициатор управляет draft")
	}
	p := b.Automation.Policies[func() string {
		if w.Grant != nil {
			return w.Grant.PolicyID
		}
		return ""
	}()]
	if p == nil || !managedContains(p.ControlPrincipals, a.Principal.ID) || sess.Role != "human" && !p.ClientIntake {
		return core.Fail("forbidden", "Нет разрешения управления workflow")
	}
	h := b.ManagedHosts[w.HostID]
	if h == nil || !managedControl(h, a.Principal.ID) {
		return core.Fail("forbidden", "Нет локального разрешения владельца")
	}
	return nil
}
func authorizeWorkflow(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	auth, ok := AutomationAuthorityFromContext(ctx)
	if !ok || !auth.Writable {
		return core.Fail("forbidden", "Проект не разрешает исполнение")
	}
	g := w.Grant
	if g == nil {
		return core.Fail("forbidden", "Workflow не авторизован")
	}
	p := b.Automation.Policies[g.PolicyID]
	if p == nil || p.State != "active" || p.Version != g.PolicyVersion || !p.ExpiresAt.After(now) || !g.ExpiresAt.After(now) {
		return core.Fail("forbidden", "Делегация истекла или отозвана")
	}
	for _, v := range []struct {
		id    string
		epoch int64
	}{{g.IssuerID, g.IssuerEpoch}, {g.RequesterID, g.RequesterEpoch}, {g.HostOwnerID, g.HostOwnerEpoch}} {
		if _, err := automationPrincipal(ctx, b, v.id, v.epoch, now); err != nil {
			return err
		}
	}
	h := b.ManagedHosts[w.HostID]
	if h == nil || h.Host.State != "active" || h.Host.OwnerPrincipalID != g.HostOwnerID || h.Host.OwnerAccessEpoch != g.HostOwnerEpoch || !managedControl(h, g.RequesterID) || !managedControl(h, g.IssuerID) {
		return core.Fail("forbidden", "Локальное разрешение изменилось")
	}
	return validateAutomationPolicy(ctx, b, p.AutomationPolicySettings, now)
}
