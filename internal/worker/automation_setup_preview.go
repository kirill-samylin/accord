package worker

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
)

type AutomationPreviewCheck struct {
	Version          int64  `json:"version"`
	ExecutableSHA256 string `json:"executable_sha256"`
	Consent          string `json:"trusted_owner_consent"`
	ConsentChanged   bool   `json:"consent_changed"`
}

type AutomationPreviewWorkspace struct {
	Version          int64                             `json:"version"`
	Identity         string                            `json:"workspace_identity"`
	GitSHA256        string                            `json:"git_sha256"`
	DependencySHA256 map[string]string                 `json:"dependency_sha256,omitempty"`
	Checks           map[string]AutomationPreviewCheck `json:"checks"`
}

// AutomationPreview contains proposed local values, never a grant or a config.
type AutomationPreview struct {
	Applied            bool                                  `json:"applied"`
	RequiresLocalApply bool                                  `json:"requires_local_apply"`
	Version            int64                                 `json:"version"`
	Workspaces         map[string]AutomationPreviewWorkspace `json:"workspaces"`
}

func previewError() error {
	return errors.New("local automation preview unavailable; inspect the private draft configuration and pins")
}

// PreviewAutomation reads a private draft and validates a completed in-memory
// proposal. It does not open an Engine, credential, state directory or network.
func PreviewAutomation(ctx context.Context, path string) (AutomationPreview, error) {
	var zero AutomationPreview
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	config, err := readPreviewConfig(path)
	if err != nil || config.Automation == nil {
		return zero, previewError()
	}
	config.Automation = cloneAutomation(config.Automation)
	if config.Automation.Version < 1 || len(config.Automation.Workspaces) == 0 || len(config.Automation.Workspaces) > 32 {
		return zero, previewError()
	}
	result := AutomationPreview{RequiresLocalApply: true, Version: config.Automation.Version, Workspaces: map[string]AutomationPreviewWorkspace{}}
	for _, alias := range slices.Sorted(maps.Keys(config.Automation.Workspaces)) {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		a := config.Automation.Workspaces[alias]
		if !a.Enabled {
			continue
		}
		space, ok := config.Workspaces[alias]
		if !ok || !identifier.MatchString(alias) || len(a.Checks) == 0 || len(a.Checks) > 32 || automationRootsForPreview(a) != nil {
			return zero, previewError()
		}
		identity, err := AutomationIdentity(space.Path)
		if err != nil {
			return zero, previewError()
		}
		a.GitSHA256, err = previewExecutableHash(ctx, a.GitExecutable, a.GitSHA256)
		if err != nil {
			return zero, previewError()
		}
		if a.DependencySHA256 == nil && len(a.DependencyRoots) > 0 {
			a.DependencySHA256 = map[string]string{}
		}
		for _, root := range a.DependencyRoots {
			if ctx.Err() != nil {
				return zero, ctx.Err()
			}
			digest, err := AutomationDependencyDigest(ctx, space.Path, root)
			if err != nil || (a.DependencySHA256[root] != "" && a.DependencySHA256[root] != digest) {
				return zero, previewError()
			}
			a.DependencySHA256[root] = digest
		}
		proposal := AutomationPreviewWorkspace{Version: a.Version, Identity: identity, GitSHA256: a.GitSHA256,
			DependencySHA256: maps.Clone(a.DependencySHA256), Checks: map[string]AutomationPreviewCheck{}}
		for _, id := range slices.Sorted(maps.Keys(a.Checks)) {
			if ctx.Err() != nil {
				return zero, ctx.Err()
			}
			preset := a.Checks[id]
			if !preset.Enabled {
				continue
			}
			if !identifier.MatchString(id) {
				return zero, previewError()
			}
			preset.ExecutableSHA256, err = previewExecutableHash(ctx, preset.Executable, preset.ExecutableSHA256)
			if err != nil {
				return zero, previewError()
			}
			old := preset.TrustedOwnerConsent
			preset.TrustedOwnerConsent = AutomationPresetDigest(space.Path, identity, id, a, preset)
			proposal.Checks[id] = AutomationPreviewCheck{Version: preset.Version, ExecutableSHA256: preset.ExecutableSHA256,
				Consent: preset.TrustedOwnerConsent, ConsentChanged: old != preset.TrustedOwnerConsent}
			a.Checks[id] = preset
		}
		config.Automation.Workspaces[alias] = a
		result.Workspaces[alias] = proposal
	}
	if len(result.Workspaces) == 0 || ctx.Err() != nil {
		return zero, previewError()
	}
	// Validate does bounded synchronous file reads. Cancellation is checked on
	// both sides; it cannot promise to interrupt blocked kernel filesystem IO.
	if err = config.Validate(); err != nil {
		return zero, previewError()
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > 1<<20 {
		return zero, previewError()
	}
	return result, nil
}

// Root topology must be checked before scanning a draft's dependency paths.
// Only missing dependency digests are substituted; existing pins stay intact.
func automationRootsForPreview(a AutomationWorkspace) error {
	copy := a
	copy.DependencySHA256 = maps.Clone(a.DependencySHA256)
	if copy.DependencySHA256 == nil {
		copy.DependencySHA256 = map[string]string{}
	}
	for _, root := range a.DependencyRoots {
		if copy.DependencySHA256[root] == "" {
			copy.DependencySHA256[root] = "0000000000000000000000000000000000000000000000000000000000000000"
		}
	}
	return automationRoots(copy)
}
