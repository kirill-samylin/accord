package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// AutomationConfig is local owner consent, never an input to a remote policy.
// Nil preserves manual workers without advertising automation capability.
type AutomationConfig struct {
	Version    int64                          `json:"version"`
	Workspaces map[string]AutomationWorkspace `json:"workspaces"`
}

type AutomationWorkspace struct {
	CodexGitMetadataWrites    bool                   `json:"codex_git_metadata_writes,omitempty"`
	AllowTesterArtifactWrites bool                   `json:"allow_tester_artifact_writes,omitempty"`
	Version                   int64                  `json:"version"`
	Enabled                   bool                   `json:"enabled"`
	ExpiresAt                 time.Time              `json:"expires_at"`
	Roles                     map[string][]string    `json:"roles"`
	MaxAttemptSeconds         int                    `json:"max_attempt_seconds"`
	MaxWorkflowSeconds        int                    `json:"max_workflow_seconds"`
	MaxNativeInvocations      int                    `json:"max_native_invocations"`
	GitExecutable             string                 `json:"git_executable"`
	GitSHA256                 string                 `json:"git_sha256"`
	SourceRoots               []string               `json:"source_roots"`
	ArtifactRoots             []string               `json:"artifact_roots"`
	DependencyRoots           []string               `json:"dependency_roots"`
	DependencySHA256          map[string]string      `json:"dependency_sha256"`
	Checks                    map[string]CheckPreset `json:"checks"`
}

// Fixed arguments may execute repository code. TrustedOwnerConsent is a
// separate acknowledgement bound to the whole preset and workspace identity.
type CheckPreset struct {
	Version             int64    `json:"version"`
	Enabled             bool     `json:"enabled"`
	Kind                string   `json:"kind"`
	Required            bool     `json:"required"`
	Executable          string   `json:"executable"`
	ExecutableSHA256    string   `json:"executable_sha256"`
	Args                []string `json:"args"`
	Directory           string   `json:"directory"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	ReportFormat        string   `json:"report_format,omitempty"`
	ReportPath          string   `json:"report_path,omitempty"`
	Isolation           string   `json:"isolation"`
	TrustedOwnerConsent string   `json:"trusted_owner_consent,omitempty"`
}

func consentError() error { return errors.New("local automation consent is invalid or unavailable") }

func verifyExecutable(path, digest string) error {
	if !validSHA256(digest) || validateLocalPath(path) != nil {
		return consentError()
	}
	f, err := os.Open(path)
	if err != nil {
		return consentError()
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<30 {
		return consentError()
	}
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, 1<<30+1)); err != nil || hex.EncodeToString(h.Sum(nil)) != digest {
		return consentError()
	}
	return nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(value) == value
}

func consentRelative(path string) bool {
	return path != "" && !filepath.IsAbs(path) && path == filepath.Clean(path) &&
		path != ".." && !strings.HasPrefix(path, ".."+string(filepath.Separator)) &&
		!strings.ContainsAny(path, "\x00\r\n")
}

func withinRoot(path, root string) bool {
	return path == root || root == "." || strings.HasPrefix(path, root+string(filepath.Separator))
}

func automationRoots(a AutomationWorkspace) error {
	if len(a.SourceRoots) == 0 || len(a.SourceRoots)+len(a.ArtifactRoots)+len(a.DependencyRoots) > 64 {
		return consentError()
	}
	seen := map[string]bool{}
	for _, group := range [][]string{a.SourceRoots, a.ArtifactRoots, a.DependencyRoots} {
		for _, root := range group {
			if !consentRelative(root) || withinRoot(root, ".git") || seen[root] {
				return consentError()
			}
			seen[root] = true
		}
	}
	if len(a.DependencySHA256) != len(a.DependencyRoots) {
		return consentError()
	}
	for _, root := range a.DependencyRoots {
		if !validSHA256(a.DependencySHA256[root]) {
			return consentError()
		}
	}
	// A root cannot be simultaneously treated as reviewed source and ignored
	// build/dependency output. Do not turn broad "." into an implicit exclusion.
	for _, source := range a.SourceRoots {
		for _, excluded := range append(slices.Clone(a.ArtifactRoots), a.DependencyRoots...) {
			if withinRoot(source, excluded) || withinRoot(excluded, source) {
				return consentError()
			}
		}
	}
	for _, artifact := range a.ArtifactRoots {
		for _, dependency := range a.DependencyRoots {
			if withinRoot(artifact, dependency) || withinRoot(dependency, artifact) {
				return consentError()
			}
		}
	}
	return nil
}

// presetConsentDigest excludes only the acknowledgement itself. Moving the
// checkout, changing arguments/report roots or replacing a binary needs consent.
func presetConsentDigest(workspace, identity, id string, a AutomationWorkspace, preset CheckPreset) string {
	preset.TrustedOwnerConsent = ""
	raw, _ := json.Marshal(struct {
		Workspace, Identity, ID      string
		Source, Artifact, Dependency []string
		DependencySHA256             map[string]string
		AllowTesterArtifactWrites    bool
		CodexGitMetadataWrites       bool `json:",omitempty"`
		Preset                       CheckPreset
	}{workspace, identity, id, a.SourceRoots, a.ArtifactRoots, a.DependencyRoots, a.DependencySHA256, a.AllowTesterArtifactWrites, a.CodexGitMetadataWrites, preset})
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func validateCheckPreset(workspace, identity, id string, a AutomationWorkspace, p CheckPreset) error {
	if !identifier.MatchString(id) || p.Version < 1 || !slices.Contains([]string{"test", "lint", "build"}, p.Kind) ||
		!consentRelative(p.Directory) || p.TimeoutSeconds < 1 || p.TimeoutSeconds > a.MaxAttemptSeconds ||
		len(p.Args) > 64 || verifyExecutable(p.Executable, p.ExecutableSHA256) != nil {
		return consentError()
	}
	bytes := 0
	for _, arg := range p.Args {
		bytes += len(arg)
		if strings.ContainsRune(arg, 0) {
			return consentError()
		}
	}
	if bytes > 16384 {
		return consentError()
	}
	if p.Kind == "test" && !slices.Contains([]string{"junit", "tap"}, p.ReportFormat) {
		return consentError()
	}
	if (p.ReportFormat == "") != (p.ReportPath == "") {
		return consentError()
	}
	if p.ReportFormat != "" && (!slices.Contains([]string{"junit", "tap"}, p.ReportFormat) || !consentRelative(p.ReportPath)) {
		return consentError()
	}
	if p.ReportPath != "" {
		inside := false
		for _, root := range a.ArtifactRoots {
			inside = inside || withinRoot(p.ReportPath, root)
		}
		if !inside {
			return consentError()
		}
	}
	// Enforced launchers are not silently emulated by a direct process. Their
	// platform-specific implementation must be added before this mode is valid.
	if p.Isolation != "trusted_owner_process" || !validSHA256(p.TrustedOwnerConsent) ||
		p.TrustedOwnerConsent != presetConsentDigest(workspace, identity, id, a, p) {
		return consentError()
	}
	return nil
}

func validateAutomation(config *AutomationConfig, workspaces map[string]Workspace, executables map[string]string, now time.Time) error {
	if config == nil {
		return nil
	}
	if config.Version < 1 || len(config.Workspaces) == 0 || len(config.Workspaces) > 32 {
		return consentError()
	}
	for alias, a := range config.Workspaces {
		w, exists := workspaces[alias]
		if !exists || !identifier.MatchString(alias) || a.Version < 1 {
			return consentError()
		}
		if !a.Enabled {
			continue
		}
		identity, err := directoryIdentity(w.Path)
		if err != nil || !a.ExpiresAt.After(now) || a.MaxAttemptSeconds < 1 || a.MaxAttemptSeconds > 7200 ||
			a.MaxWorkflowSeconds < a.MaxAttemptSeconds || a.MaxWorkflowSeconds > 86400 ||
			a.MaxNativeInvocations < 1 || a.MaxNativeInvocations > 128 ||
			len(a.Roles) != 4 || len(a.Checks) == 0 || len(a.Checks) > 32 ||
			verifyExecutable(a.GitExecutable, a.GitSHA256) != nil || automationRoots(a) != nil {
			return consentError()
		}
		for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
			providers := a.Roles[role]
			if len(providers) == 0 || len(providers) > len(nativeProviders) {
				return consentError()
			}
			seen := map[string]bool{}
			for _, provider := range providers {
				if seen[provider] || !knownProvider(provider) || !slices.Contains(w.Providers, provider) || executables[provider] == "" {
					return consentError()
				}
				seen[provider] = true
			}
		}
		test := false
		for id, preset := range a.Checks {
			if !preset.Enabled {
				continue
			}
			if validateCheckPreset(w.Path, identity, id, a, preset) != nil {
				return consentError()
			}
			test = test || preset.Kind == "test" && preset.Required
		}
		if !test {
			return consentError()
		}
	}
	return nil
}

func cloneAutomation(config *AutomationConfig) *AutomationConfig {
	if config == nil {
		return nil
	}
	copy := *config
	copy.Workspaces = make(map[string]AutomationWorkspace, len(config.Workspaces))
	for alias, a := range config.Workspaces {
		a.SourceRoots = slices.Clone(a.SourceRoots)
		a.ArtifactRoots = slices.Clone(a.ArtifactRoots)
		a.DependencyRoots = slices.Clone(a.DependencyRoots)
		a.DependencySHA256 = maps.Clone(a.DependencySHA256)
		roles := make(map[string][]string, len(a.Roles))
		for role, providers := range a.Roles {
			roles[role] = slices.Clone(providers)
		}
		a.Roles = roles
		checks := make(map[string]CheckPreset, len(a.Checks))
		for id, p := range a.Checks {
			p.Args = slices.Clone(p.Args)
			checks[id] = p
		}
		a.Checks = checks
		copy.Workspaces[alias] = a
	}
	return &copy
}
