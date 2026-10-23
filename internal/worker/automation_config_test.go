package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type automationFixture struct {
	config    *AutomationConfig
	spaces    map[string]Workspace
	providers map[string]string
	now       time.Time
	identity  string
}

func newAutomationFixture(t *testing.T) automationFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "fixed-check")
	contents := []byte("synthetic executable bytes; configuration tests never execute them")
	if err := os.WriteFile(executable, contents, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	sha := hex.EncodeToString(digest[:])
	identity, err := directoryIdentity(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	a := AutomationWorkspace{Version: 1, Enabled: true, ExpiresAt: now.Add(time.Hour),
		Roles:             map[string][]string{"analyst": {"codex"}, "developer": {"claude"}, "reviewer": {"codex"}, "tester": {"claude"}},
		MaxAttemptSeconds: 300, MaxWorkflowSeconds: 1200, MaxNativeInvocations: 40,
		GitExecutable: executable, GitSHA256: sha, SourceRoots: []string{"src", "tests", "package.json"}, ArtifactRoots: []string{".artifacts"},
		Checks: map[string]CheckPreset{"unit": {Version: 1, Enabled: true, Kind: "test", Required: true, Executable: executable, ExecutableSHA256: sha,
			Directory: ".", TimeoutSeconds: 60, ReportFormat: "tap", ReportPath: ".artifacts/test.tap", Isolation: "trusted_owner_process"}}}
	p := a.Checks["unit"]
	p.TrustedOwnerConsent = presetConsentDigest(root, identity, "unit", a, p)
	a.Checks["unit"] = p
	return automationFixture{&AutomationConfig{Version: 1, Workspaces: map[string]AutomationWorkspace{"repo": a}},
		map[string]Workspace{"repo": {Path: root, Providers: []string{"codex", "claude"}}},
		map[string]string{"codex": executable, "claude": executable}, now, identity}
}

func (f automationFixture) validate() error {
	return validateAutomation(f.config, f.spaces, f.providers, f.now)
}

func TestAutomationConsentIsOptInAndAllowsMixedLocalProviders(t *testing.T) {
	f := newAutomationFixture(t)
	if err := f.validate(); err != nil {
		t.Fatal(err)
	}
	if err := validateAutomation(nil, nil, nil, f.now); err != nil {
		t.Fatal("manual workers changed", err)
	}
	a := f.config.Workspaces["repo"]
	a.Enabled = false
	a.Checks = nil
	a.ExpiresAt = time.Time{}
	f.config.Workspaces["repo"] = a
	if err := f.validate(); err != nil {
		t.Fatal("disabled opt-in should confer no authority", err)
	}
}

func TestAutomationConsentRejectsUnsafeConfiguration(t *testing.T) {
	cases := map[string]func(*AutomationWorkspace){
		"expired":                  func(a *AutomationWorkspace) { a.ExpiresAt = time.Time{} },
		"missing-role":             func(a *AutomationWorkspace) { delete(a.Roles, "reviewer") },
		"unknown-role":             func(a *AutomationWorkspace) { a.Roles["human"] = []string{"codex"} },
		"duplicate-provider":       func(a *AutomationWorkspace) { a.Roles["tester"] = []string{"codex", "codex"} },
		"unsupported-provider":     func(a *AutomationWorkspace) { a.Roles["tester"] = []string{"remote"} },
		"runtime":                  func(a *AutomationWorkspace) { a.MaxAttemptSeconds = 7201 },
		"workflow-runtime":         func(a *AutomationWorkspace) { a.MaxWorkflowSeconds = 1 },
		"invocations":              func(a *AutomationWorkspace) { a.MaxNativeInvocations = 129 },
		"source-exclusion-overlap": func(a *AutomationWorkspace) { a.SourceRoots = []string{"."} },
		"artifact-traversal":       func(a *AutomationWorkspace) { a.ArtifactRoots = []string{"../outside"} },
		"git-artifact":             func(a *AutomationWorkspace) { a.ArtifactRoots = []string{".git"} },
		"unpinned-dependencies":    func(a *AutomationWorkspace) { a.DependencyRoots = []string{"node_modules"} },
		"no-tests":                 func(a *AutomationWorkspace) { p := a.Checks["unit"]; p.Enabled = false; a.Checks["unit"] = p },
		"unapproved-binary":        func(a *AutomationWorkspace) { a.GitSHA256 = string(make([]byte, 64)) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newAutomationFixture(t)
			a := f.config.Workspaces["repo"]
			mutate(&a)
			f.config.Workspaces["repo"] = a
			if err := f.validate(); err == nil {
				t.Fatal("unsafe local consent accepted")
			}
		})
	}
}

func TestCheckConsentBindsExactExecutionAndDoesNotEmulateIsolation(t *testing.T) {
	cases := map[string]func(*CheckPreset){
		"no-consent":              func(p *CheckPreset) { p.TrustedOwnerConsent = "" },
		"changed-args":            func(p *CheckPreset) { p.Args = []string{"--different"} },
		"changed-directory":       func(p *CheckPreset) { p.Directory = "src" },
		"changed-timeout":         func(p *CheckPreset) { p.TimeoutSeconds++ },
		"unimplemented-isolation": func(p *CheckPreset) { p.Isolation = "enforced_launcher" },
		"missing-report":          func(p *CheckPreset) { p.ReportFormat = ""; p.ReportPath = "" },
		"report-outside-artifact": func(p *CheckPreset) { p.ReportPath = "src/test.tap" },
		"mismatched-report":       func(p *CheckPreset) { p.ReportPath = "" },
		"nul-argument":            func(p *CheckPreset) { p.Args = []string{"\x00"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newAutomationFixture(t)
			a := f.config.Workspaces["repo"]
			p := a.Checks["unit"]
			mutate(&p)
			if name == "unimplemented-isolation" || name == "missing-report" || name == "report-outside-artifact" || name == "mismatched-report" || name == "nul-argument" {
				p.TrustedOwnerConsent = presetConsentDigest(f.spaces["repo"].Path, f.identity, "unit", a, p)
			}
			a.Checks["unit"] = p
			f.config.Workspaces["repo"] = a
			if err := f.validate(); err == nil {
				t.Fatal("altered preset accepted")
			}
		})
	}
}

func TestCheckConsentCannotMoveToAnotherWorkspace(t *testing.T) {
	f := newAutomationFixture(t)
	other, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.spaces["repo"] = Workspace{Path: other, Providers: []string{"codex", "claude"}}
	if f.validate() == nil {
		t.Fatal("consent moved to another checkout")
	}
}

func TestAutomationConsentRejectsChangedExecutable(t *testing.T) {
	f := newAutomationFixture(t)
	if err := os.WriteFile(f.config.Workspaces["repo"].GitExecutable, []byte("replacement"), 0700); err != nil {
		t.Fatal(err)
	}
	if f.validate() == nil {
		t.Fatal("replaced executable accepted")
	}
}

func TestSnapshotDoesNotShareMutableAutomationConsent(t *testing.T) {
	f := newAutomationFixture(t)
	a := f.config.Workspaces["repo"]
	a.DependencyRoots = []string{"deps"}
	a.DependencySHA256 = map[string]string{"deps": a.GitSHA256}
	p := a.Checks["unit"]
	p.Args = []string{"--unit"}
	p.TrustedOwnerConsent = presetConsentDigest(f.spaces["repo"].Path, f.identity, "unit", a, p)
	a.Checks["unit"] = p
	f.config.Workspaces["repo"] = a
	copy := snapshotConfig(Config{Automation: f.config, Workspaces: f.spaces, Executables: f.providers})
	a.Roles["analyst"][0] = "remote"
	a.SourceRoots[0] = ".git"
	a.DependencySHA256["deps"] = "changed"
	p.Args[0] = "--changed"
	if validateAutomation(copy.Automation, copy.Workspaces, copy.Executables, f.now) != nil {
		t.Fatal("immutable preparation snapshot changed through original consent maps/slices")
	}
	if f.validate() == nil {
		t.Fatal("original mutation was not exercised")
	}
}
