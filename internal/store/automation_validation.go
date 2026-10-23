package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/hex"
	"reflect"
	"time"
)

var automationRoles = []string{"analyst", "developer", "reviewer", "tester"}

func automationLimits(v core.AutomationLimits) core.AutomationLimits {
	// An absent limits object receives visible defaults. Explicit zero optional
	// budgets remain zero; normalization must never silently grant more retries.
	if v == (core.AutomationLimits{}) {
		return core.AutomationLimits{PlanTasks: 8, FixPerTask: 2, FixTotal: 2, NativeInvocations: 40, ContinuationPerStep: 1, StepSeconds: 1800, WorkflowSeconds: 28800, GrantSeconds: 86400}
	}
	if v.PlanTasks == 0 {
		v.PlanTasks = 8
	}
	if v.NativeInvocations == 0 {
		v.NativeInvocations = 40
	}
	if v.StepSeconds == 0 {
		v.StepSeconds = 1800
	}
	if v.WorkflowSeconds == 0 {
		v.WorkflowSeconds = 28800
	}
	if v.GrantSeconds == 0 {
		v.GrantSeconds = 86400
	}
	return v
}

func automationLimitsValid(v core.AutomationLimits) bool {
	return v.PlanTasks >= 1 && v.PlanTasks <= 16 && v.FixPerTask >= 0 && v.FixPerTask <= 5 && v.FixTotal >= 0 && v.FixTotal <= 10 && v.NativeInvocations >= 1 && v.NativeInvocations <= 128 && v.ContinuationPerStep >= 0 && v.ContinuationPerStep <= 1 && v.StepSeconds >= 1 && v.StepSeconds <= 7200 && v.WorkflowSeconds >= v.StepSeconds && v.WorkflowSeconds <= 86400 && v.GrantSeconds >= v.WorkflowSeconds && v.GrantSeconds <= 604800
}
func digestValid(v string) bool {
	raw, err := hex.DecodeString(v)
	return err == nil && len(raw) == 32 && len(v) == 64
}
func automationConsentValid(v *core.AutomationHostConsent, workspaces, providers []string, now time.Time) error {
	if v == nil {
		return nil
	}
	if v.ScopesTruncated || v.Version < 1 || !v.Enabled || len(v.Scopes) == 0 || len(v.Scopes) > 32 || !v.ExpiresAt.After(now) {
		return core.Fail("validation", "Неверное локальное разрешение автоматизации")
	}
	for alias, scope := range v.Scopes {
		if !managedContains(workspaces, alias) || scope.Version < 1 || !scope.ExpiresAt.After(now) || scope.ExpiresAt.After(v.ExpiresAt) || scope.MaxStepSeconds < 1 || scope.MaxStepSeconds > 7200 || scope.MaxWorkflowSeconds < scope.MaxStepSeconds || scope.MaxWorkflowSeconds > 86400 || scope.MaxNativeInvocations < 1 || scope.MaxNativeInvocations > 128 || len(scope.Roles) != 4 || len(scope.Checks) == 0 || len(scope.Checks) > 32 {
			return core.Fail("validation", "Неверный workspace scope")
		}
		for _, role := range automationRoles {
			allowed := scope.Roles[role]
			if len(allowed) < 1 || len(allowed) > 2 {
				return core.Fail("validation", "Нужны локальные роли/providers")
			}
			seen := map[string]bool{}
			for _, provider := range allowed {
				if seen[provider] || !managedContains(providers, provider) {
					return core.Fail("validation", "Provider не разрешён workspace")
				}
				seen[provider] = true
			}
		}
		checks := map[string]bool{}
		hasTest := false
		for _, c := range scope.Checks {
			if !bounded(c.PresetID, 128) || checks[c.PresetID] || c.LocalVersion < 1 || !digestValid(c.LocalDigest) || (c.Kind != "test" && c.Kind != "lint" && c.Kind != "build") || (c.Isolation != "trusted_owner_process" && c.Isolation != "enforced_launcher") {
				return core.Fail("validation", "Неверный static check preset")
			}
			checks[c.PresetID] = true
			hasTest = hasTest || c.Kind == "test" && c.Required
		}
		if !hasTest {
			return core.Fail("validation", "Требуется обязательный test preset")
		}
	}
	return nil
}
func validateAutomationPolicy(ctx context.Context, b *board, in core.AutomationPolicySettings, now time.Time) error {
	if !bounded(in.Name, 120) || len(in.RoleBindings) != 4 || !automationLimitsValid(in.Limits) || in.GitPolicy != "local_commits" || !in.ExpiresAt.After(now) || in.ExpiresAt.After(now.Add(7*24*time.Hour)) {
		return core.Fail("validation", "Проверьте имя, срок, роли, local_commits и бюджеты")
	}
	h := b.ManagedHosts[in.HostID]
	if h == nil || h.Host.State != "active" || h.Host.Automation == nil || !h.Host.Automation.Enabled || h.Host.Automation.Version != in.HostConsentVersion {
		return core.Fail("forbidden", "Нет текущего local automation consent")
	}
	consent := h.Host.Automation
	scope, ok := consent.Scopes[in.Workspace]
	if !ok || !consent.ExpiresAt.After(now) || !scope.ExpiresAt.After(now) || in.ExpiresAt.After(consent.ExpiresAt) || in.ExpiresAt.After(scope.ExpiresAt) || in.Limits.StepSeconds > scope.MaxStepSeconds || in.Limits.WorkflowSeconds > scope.MaxWorkflowSeconds || in.Limits.NativeInvocations > scope.MaxNativeInvocations {
		return core.Fail("forbidden", "Политика выходит за конкретный workspace scope")
	}
	if _, err := automationPrincipal(ctx, b, h.Host.OwnerPrincipalID, h.Host.OwnerAccessEpoch, now); err != nil {
		return err
	}
	for _, role := range automationRoles {
		binding, ok := in.RoleBindings[role]
		if !ok || !managedContains(scope.Roles[role], binding.Provider) {
			return core.Fail("forbidden", "Роль/provider не разрешены для workspace")
		}
		p, err := managedSnapshot(b, core.ManagedStartInput{Provider: binding.Provider, ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion})
		if err != nil {
			return err
		}
		if p.Role != role {
			return core.Fail("validation", "Роль профиля не совпадает с шагом")
		}
	}
	if len(in.IntakePrincipals) == 0 || len(in.ControlPrincipals) == 0 || len(in.IntakePrincipals) > 128 || len(in.ControlPrincipals) > 128 {
		return core.Fail("validation", "Нужны явные intake/control участники")
	}
	auth, _ := AutomationAuthorityFromContext(ctx)
	for _, ids := range [][]string{in.IntakePrincipals, in.ControlPrincipals} {
		seen := map[string]bool{}
		for _, id := range ids {
			p, exists := auth.Principals[id]
			if !exists || seen[id] || !managedControl(h, id) {
				return core.Fail("forbidden", "Участник не имеет local host control")
			}
			seen[id] = true
			if _, err := automationPrincipal(ctx, b, id, p.Epoch, now); err != nil {
				return err
			}
		}
	}
	if len(in.Checks) < 1 || len(in.Checks) > 32 {
		return core.Fail("validation", "Нужны локальные checks")
	}
	seen := map[string]bool{}
	requiredTest := false
	for _, c := range in.Checks {
		match := false
		for _, local := range scope.Checks {
			if reflect.DeepEqual(c, local) {
				match = true
			}
		}
		if !match || seen[c.PresetID] {
			return core.Fail("forbidden", "Check не совпадает с workspace consent")
		}
		seen[c.PresetID] = true
		requiredTest = requiredTest || c.Required && c.Kind == "test"
	}
	for _, c := range scope.Checks {
		if c.Required && !seen[c.PresetID] {
			return core.Fail("forbidden", "Нельзя исключить обязательную локальную проверку")
		}
	}
	if !requiredTest {
		return core.Fail("validation", "Нужен обязательный test")
	}
	return nil
}
