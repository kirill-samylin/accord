package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"agent-board/internal/core"
)

func permissionFixture(t *testing.T) (configInspection, map[string]any) {
	t.Helper()
	f := newAutomationFixture(t)
	directory := f.spaces["repo"].Path
	if err := os.Mkdir(filepath.Join(directory, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	i, _, err := newConfigInspection(directory, f.providers["codex"], nil)
	if err != nil {
		t.Fatal(err)
	}
	config := map[string]any{"default_permissions": i.Profile, "approval_policy": "never", "permissions": map[string]any{i.Profile: permissionProfile(i)}, "mcp_servers": map[string]any{}}
	r := map[string]any{"config": config, "origins": map[string]any{}, "layers": []any{map[string]any{"name": map[string]any{"type": "sessionFlags"}, "version": "opaque-v1", "config": config}}}
	return i, r
}
func permissionRaw(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

func TestGitMetadataOptInIsLocalDeveloperOnlyAndBindsConsent(t *testing.T) {
	f := newAutomationFixture(t)
	a := f.config.Workspaces["repo"]
	root := f.spaces["repo"].Path
	p := a.Checks["unit"]
	old := presetConsentDigest(root, f.identity, "unit", a, p)
	// Explicitly reconstruct the original pre-opt-in wire, so false cannot
	// invalidate existing owner acknowledgements during a capable update.
	copy := p
	copy.TrustedOwnerConsent = ""
	legacy := permissionDigest(struct {
		Workspace, Identity, ID      string
		Source, Artifact, Dependency []string
		DependencySHA256             map[string]string
		AllowTesterArtifactWrites    bool
		Preset                       CheckPreset
	}{root, f.identity, "unit", a.SourceRoots, a.ArtifactRoots, a.DependencyRoots, a.DependencySHA256, a.AllowTesterArtifactWrites, copy})
	if old != legacy {
		t.Fatal("default-off consent digest changed")
	}
	a.CodexGitMetadataWrites = true
	if presetConsentDigest(root, f.identity, "unit", a, p) == old {
		t.Fatal("opt-in not bound to digest")
	}
	f.config.Workspaces["repo"] = a
	c := Config{Automation: f.config, Executables: f.providers, NativeMCPApprovedTools: map[string][]string{"codex": managedMCPTools, "claude": managedMCPTools}}
	if validateAutomationNativeConsent(c) == nil {
		t.Fatal("missing workspace-write allowed")
	}
	c.CodexSandbox = "workspace-write"
	if validateAutomationNativeConsent(c) == nil {
		t.Fatal("missing Codex developer allowed")
	}
	a.Roles["developer"] = []string{"codex"}
	f.config.Workspaces["repo"] = a
	if err := validateAutomationNativeConsent(c); err != nil {
		t.Fatal(err)
	}
	r := core.ManagedRun{Provider: "codex", Workspace: "repo", Profile: &core.ProfileSnapshot{ProfileSettings: core.ProfileSettings{Role: "developer"}}, Automation: &core.ManagedAutomationBinding{}}
	if !gitMetadataMode(c, r) {
		t.Fatal("explicit eligible local consent missing")
	}
	for _, role := range []string{"analyst", "reviewer", "tester"} {
		r.Profile.Role = role
		if gitMetadataMode(c, r) {
			t.Fatal("other role gained Git write")
		}
	}
	r.Profile.Role = "developer"
	r.Automation = nil
	if gitMetadataMode(c, r) {
		t.Fatal("manual gained permission")
	}
	r.Automation = &core.ManagedAutomationBinding{}
	r.Provider = "claude"
	if gitMetadataMode(c, r) {
		t.Fatal("Claude gained Codex grant")
	}
}

func TestGitPermissionProfileExactCanonicalMetadataOnly(t *testing.T) {
	i, _ := permissionFixture(t)
	args := strings.Join(gitPermissionOverrides(i), " ")
	for _, forbidden := range []string{"sandbox_mode", "ignore-user-config", "approve-for-me", "danger-full-access", "ignore-rules"} {
		if strings.Contains(args, forbidden) {
			t.Fatal(forbidden)
		}
	}
	profile := permissionProfile(i)
	if !reflect.DeepEqual(profile["network"], map[string]any{"enabled": false}) {
		t.Fatal("command network expanded")
	}
	if err := os.Rename(filepath.Join(i.Directory, ".git"), filepath.Join(i.Directory, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i.Directory, ".git"), []byte("gitdir: external"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := gitDirectoryIdentity(i.Directory); err == nil {
		t.Fatal("external gitdir allowed")
	}
}

func TestPermissionProjectionRejectsHiddenOrBroaderConfiguration(t *testing.T) {
	cases := map[string]func(configInspection, map[string]any){
		"missing-effective-profile": func(i configInspection, r map[string]any) { delete(r["config"].(map[string]any), "permissions") },
		"additional-root": func(i configInspection, r map[string]any) {
			r["config"].(map[string]any)["permissions"].(map[string]any)[i.Profile].(map[string]any)["workspace_roots"] = map[string]any{"/outside": true}
		},
		"network": func(i configInspection, r map[string]any) {
			r["config"].(map[string]any)["permissions"].(map[string]any)[i.Profile].(map[string]any)["network"] = map[string]any{"enabled": true}
		},
		"legacy-active-layer": func(i configInspection, r map[string]any) {
			r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)["sandbox_mode"] = "workspace-write"
		},
		"unknown-layer": func(i configInspection, r map[string]any) {
			r["layers"].([]any)[0].(map[string]any)["name"] = map[string]any{"type": "futurePermissions"}
		},
		"active-MCP": func(i configInspection, r map[string]any) {
			r["config"].(map[string]any)["mcp_servers"] = map[string]any{"unknown": map[string]any{"enabled": true}}
		},
		"unknown-managed-profile": func(i configInspection, r map[string]any) {
			r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)["permissions"].(map[string]any)[":workspace"] = map[string]any{"filesystem": map[string]any{"/outside": "write"}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			i, r := permissionFixture(t)
			mutate(i, r)
			if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
				t.Fatal("unsafe/unknown effective grant accepted")
			}
		})
	}
}

func TestPermissionProofIgnoresTrustMetadataButRejectsManagedContradictions(t *testing.T) {
	i, r := permissionFixture(t)
	req := permissionRaw(map[string]any{"requirements": nil})
	p, err := provePermissionConfig(permissionRaw(r), req, i)
	if err != nil {
		t.Fatal(err)
	}
	c := r["config"].(map[string]any)
	c["projects"] = map[string]any{"opaque": "secret trust metadata"}
	c["sandbox_mode"] = nil
	r["layers"].([]any)[0].(map[string]any)["version"] = "unrelated-file-mtime-change"
	next, err := provePermissionConfig(permissionRaw(r), req, i)
	if err != nil || next != p {
		t.Fatal("unrelated config metadata invalidated permission proof", err)
	}
	for _, requirements := range []map[string]any{{"allowedPermissionProfiles": map[string]any{i.Profile: false}}, {"defaultPermissions": ":danger-full-access"}, {"allowedApprovalPolicies": []string{"on-request"}}, {"allowedSandboxModes": []string{"danger-full-access"}}, {"futurePermissions": true}} {
		if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": requirements}), i); err == nil {
			t.Fatal("unproven managed constraint accepted")
		}
	}
}

func TestEffectiveLegacySandboxCannotHideBehindCleanOrEmptyLayers(t *testing.T) {
	for _, key := range []string{"sandbox_mode", "sandbox_workspace_write"} {
		for _, emptyLayers := range []bool{false, true} {
			i, r := permissionFixture(t)
			// fixture config/layer normally share a map. Clone only the effective
			// config so this regression cannot pass through the existing layer guard.
			var effective map[string]any
			if err := json.Unmarshal(permissionRaw(r["config"]), &effective); err != nil {
				t.Fatal(err)
			}
			r["config"] = effective
			var value any = "workspace-write"
			if key == "sandbox_workspace_write" {
				value = map[string]any{"network_access": false}
			}
			effective[key] = value
			if emptyLayers {
				r["layers"] = []any{}
			} else if _, exists := r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)[key]; exists {
				t.Fatal("test accidentally altered the layer")
			}
			if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
				t.Fatal("effective-only legacy mode was accepted", key, emptyLayers)
			}
		}
		i, r := permissionFixture(t)
		var effective map[string]any
		if err := json.Unmarshal(permissionRaw(r["config"]), &effective); err != nil {
			t.Fatal(err)
		}
		r["config"] = effective
		effective[key] = nil
		if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err != nil {
			t.Fatal("nullable effective default was treated as active legacy", key, err)
		}
	}
}
