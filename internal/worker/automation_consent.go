package worker

import (
	"agent-board/internal/core"
	"errors"
	"slices"
	"time"
)

func automationHostConsent(config Config, now time.Time) (*core.AutomationHostConsent, error) {
	if config.Automation == nil {
		return nil, nil
	}
	if err := validateAutomation(config.Automation, config.Workspaces, config.Executables, now); err != nil {
		return nil, err
	}
	consent := &core.AutomationHostConsent{Version: config.Automation.Version, Enabled: true, Scopes: map[string]core.AutomationWorkspaceScope{}}
	for alias, a := range config.Automation.Workspaces {
		if !a.Enabled {
			continue
		}
		identity, err := directoryIdentity(config.Workspaces[alias].Path)
		if err != nil {
			return nil, consentError()
		}
		scope := core.AutomationWorkspaceScope{Version: a.Version, ExpiresAt: a.ExpiresAt, Roles: map[string][]string{}, MaxStepSeconds: int64(a.MaxAttemptSeconds), MaxWorkflowSeconds: int64(a.MaxWorkflowSeconds), MaxNativeInvocations: a.MaxNativeInvocations}
		for role, providers := range a.Roles {
			scope.Roles[role] = slices.Clone(providers)
			for _, p := range providers {
				if !slices.Contains(consent.Providers, p) {
					consent.Providers = append(consent.Providers, p)
				}
			}
		}
		for id, p := range a.Checks {
			if !p.Enabled {
				continue
			}
			scope.Checks = append(scope.Checks, core.AutomationCheckPresetRef{PresetID: id, LocalVersion: p.Version, LocalDigest: presetConsentDigest(config.Workspaces[alias].Path, identity, id, a, p), Kind: p.Kind, Required: p.Required, Isolation: p.Isolation})
		}
		slices.SortFunc(scope.Checks, func(a, b core.AutomationCheckPresetRef) int {
			if a.PresetID < b.PresetID {
				return -1
			}
			if a.PresetID > b.PresetID {
				return 1
			}
			return 0
		})
		consent.Scopes[alias] = scope
		consent.Workspaces = append(consent.Workspaces, alias)
		if consent.ExpiresAt.IsZero() || a.ExpiresAt.After(consent.ExpiresAt) {
			consent.ExpiresAt = a.ExpiresAt
		}
		consent.MaxStepSeconds = max(consent.MaxStepSeconds, int64(a.MaxAttemptSeconds))
	}
	if len(consent.Scopes) == 0 {
		return nil, nil
	}
	consent.Roles = []string{"analyst", "developer", "reviewer", "tester"}
	slices.Sort(consent.Workspaces)
	slices.Sort(consent.Providers)
	return consent, nil
}

func automationCapabilities(config Config) map[string]int {
	if config.Automation == nil {
		return nil
	}
	for _, workspace := range config.Automation.Workspaces {
		if workspace.Enabled {
			return map[string]int{"automation": 1}
		}
	}
	return nil
}

// Remote role/profile choices are intersected with local consent at execution,
// including unchanged static verification presets and the checkout identity.
func (e *Engine) automationAllowed(run core.ManagedRun, now time.Time) (AutomationWorkspace, error) {
	var zero AutomationWorkspace
	if run.Automation == nil {
		return zero, nil
	}
	if e.config.Automation == nil || run.Profile == nil {
		return zero, consentError()
	}
	a, ok := e.config.Automation.Workspaces[run.Workspace]
	if !ok || !a.Enabled || !a.ExpiresAt.After(now) || !run.Automation.DeadlineAt.After(now) ||
		!slices.Contains(a.Roles[run.Profile.Role], run.Provider) {
		return zero, consentError()
	}
	identity, err := directoryIdentity(e.config.Workspaces[run.Workspace].Path)
	if err != nil {
		return zero, consentError()
	}
	seen := map[string]bool{}
	for _, required := range run.Automation.Checks {
		p, ok := a.Checks[required.PresetID]
		if seen[required.PresetID] || !ok || !p.Enabled || p.Version != required.LocalVersion || p.Kind != required.Kind || p.Required != required.Required || p.Isolation != required.Isolation ||
			required.LocalDigest != presetConsentDigest(e.config.Workspaces[run.Workspace].Path, identity, required.PresetID, a, p) ||
			validateCheckPreset(e.config.Workspaces[run.Workspace].Path, identity, required.PresetID, a, p) != nil {
			return zero, consentError()
		}
		seen[required.PresetID] = true
	}
	if run.Profile.Role != "analyst" {
		for id, p := range a.Checks {
			if p.Enabled && p.Required && !seen[id] {
				return zero, consentError()
			}
		}
	}
	if run.Automation.TaskGeneration < 1 || len(run.Automation.Checks) > 32 {
		return zero, consentError()
	}
	if run.Automation.WorkflowID == "" || run.Automation.StepID == "" || run.Automation.ControlEpoch < 1 || run.Automation.PolicyVersion < 1 {
		return zero, errors.New("immutable workflow execution binding is required")
	}
	return a, nil
}
