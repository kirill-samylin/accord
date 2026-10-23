package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func previewFixture(t *testing.T) (Config, string) {
	t.Helper()
	f := newAutomationFixture(t)
	root := f.spaces["repo"].Path
	tools := []string{"describe", "next", "get", "transition", "workflow_question", "workflow_plan", "workflow_run_check"}
	c := Config{Automation: f.config, Server: "https://secret.invalid/private-value", Project: "p_fixture",
		Name: "preview", Machine: "preview", TokenFile: filepath.Join(root, "unread-secret-token"),
		StateDir: filepath.Join(root, "never-created-state"), Executables: f.providers, Workspaces: f.spaces,
		CodexSandbox: "workspace-write", NativeMCPApprovedTools: map[string][]string{"codex": tools, "claude": tools}}
	return c, filepath.Join(root, "draft.json")
}

func writePreviewDraft(t *testing.T, path string, c Config) []byte {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil || os.WriteFile(path, raw, 0600) != nil {
		t.Fatal("write private fixture draft", err)
	}
	return raw
}

func TestAutomationPreviewProposesValuesWithoutApplyingOrReadingCredentials(t *testing.T) {
	c, path := previewFixture(t)
	a := c.Automation.Workspaces["repo"]
	a.GitSHA256, a.DependencySHA256 = "", nil
	a.DependencyRoots = []string{"deps"}
	root := c.Workspaces["repo"].Path
	if err := os.Mkdir(filepath.Join(root, "deps"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deps", "input"), []byte("pinned input"), 0600); err != nil {
		t.Fatal(err)
	}
	p := a.Checks["unit"]
	p.ExecutableSHA256, p.TrustedOwnerConsent = "", ""
	p.Args = []string{"--private-argument-value"}
	a.Checks["unit"] = p
	c.Automation.Workspaces["repo"] = a
	before := writePreviewDraft(t, path, c)
	result, err := PreviewAutomation(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	w := result.Workspaces["repo"]
	if result.Applied || !result.RequiresLocalApply || !validSHA256(w.GitSHA256) || !validSHA256(w.DependencySHA256["deps"]) || !w.Checks["unit"].ConsentChanged {
		t.Fatal("missing proposal or false apply", result)
	}
	a.GitSHA256, a.DependencySHA256 = w.GitSHA256, w.DependencySHA256
	p.ExecutableSHA256 = w.Checks["unit"].ExecutableSHA256
	if w.Checks["unit"].Consent != AutomationPresetDigest(root, w.Identity, "unit", a, p) {
		t.Fatal("proposal did not bind completed local inputs")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("preview modified the draft")
	}
	for _, missing := range []string{c.TokenFile, c.StateDir} {
		if _, err := os.Stat(missing); !os.IsNotExist(err) {
			t.Fatal("preview opened/created credential or state", err)
		}
	}
	encoded, _ := json.Marshal(result)
	for _, secret := range []string{"secret.invalid", "private-value", "unread-secret-token", "--private-argument-value"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("private config value in proposal")
		}
	}
}

func TestAutomationPreviewRejectsChangedPinsAndUnsafeDrafts(t *testing.T) {
	cases := map[string]func(*Config){
		"git-pin": func(c *Config) {
			a := c.Automation.Workspaces["repo"]
			a.GitSHA256 = strings.Repeat("0", 64)
			c.Automation.Workspaces["repo"] = a
		},
		"check-pin": func(c *Config) {
			a := c.Automation.Workspaces["repo"]
			p := a.Checks["unit"]
			p.ExecutableSHA256 = strings.Repeat("0", 64)
			a.Checks["unit"] = p
		},
		"dependency-escape": func(c *Config) {
			a := c.Automation.Workspaces["repo"]
			a.DependencyRoots = []string{"../outside"}
			c.Automation.Workspaces["repo"] = a
		},
		"root-overlap": func(c *Config) {
			a := c.Automation.Workspaces["repo"]
			a.SourceRoots = []string{"."}
			c.Automation.Workspaces["repo"] = a
		},
		"missing-native-consent": func(c *Config) { c.NativeMCPApprovedTools = nil },
		"missing-automation":     func(c *Config) { c.Automation = nil },
		"git-optin-role": func(c *Config) {
			a := c.Automation.Workspaces["repo"]
			a.CodexGitMetadataWrites = true
			c.Automation.Workspaces["repo"] = a
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, path := previewFixture(t)
			mutate(&c)
			before := writePreviewDraft(t, path, c)
			out, err := PreviewAutomation(context.Background(), path)
			if err == nil || len(out.Workspaces) != 0 || strings.Contains(err.Error(), "private-value") {
				t.Fatal("unsafe draft accepted or private error", out, err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("failed preview changed input")
			}
		})
	}
}

func TestAutomationPreviewConsentTracksChangesButDoesNotApplyThem(t *testing.T) {
	c, path := previewFixture(t)
	writePreviewDraft(t, path, c)
	first, err := PreviewAutomation(context.Background(), path)
	if err != nil || first.Workspaces["repo"].Checks["unit"].ConsentChanged {
		t.Fatal("existing matching consent changed", err)
	}
	a := c.Automation.Workspaces["repo"]
	p := a.Checks["unit"]
	p.Args = []string{"different-reviewed-arguments"}
	a.Checks["unit"] = p
	writePreviewDraft(t, path, c)
	second, err := PreviewAutomation(context.Background(), path)
	if err != nil || !second.Workspaces["repo"].Checks["unit"].ConsentChanged || second.Applied ||
		second.Workspaces["repo"].Checks["unit"].Consent == first.Workspaces["repo"].Checks["unit"].Consent {
		t.Fatal("changed preset not surfaced as unapplied proposal", err)
	}
}

func TestAutomationPreviewStrictPrivateInputAndCancellation(t *testing.T) {
	c, path := previewFixture(t)
	raw := writePreviewDraft(t, path, c)
	for _, content := range []string{`{"name":"a","name":"b"}`, `{"unknown":"secret"}`, string(raw) + "{}", strings.Repeat(" ", 65537)} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := PreviewAutomation(context.Background(), path); err == nil {
			t.Fatal("malformed private input accepted")
		}
	}
	writePreviewDraft(t, path, c)
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewAutomation(context.Background(), path); err == nil {
		t.Fatal("public draft accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(path), "draft-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewAutomation(context.Background(), link); err == nil {
		t.Fatal("symlink draft accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PreviewAutomation(ctx, path); err != context.Canceled {
		t.Fatal("cancelled preview continued", err)
	}
}

func TestAutomationPreviewDoesNotReplaceAnExistingDependencyPin(t *testing.T) {
	c, path := previewFixture(t)
	root := c.Workspaces["repo"].Path
	if err := os.Mkdir(filepath.Join(root, "deps"), 0700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "deps", "input")
	if err := os.WriteFile(input, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	a := c.Automation.Workspaces["repo"]
	a.DependencyRoots = []string{"deps"}
	c.Automation.Workspaces["repo"] = a
	writePreviewDraft(t, path, c)
	first, err := PreviewAutomation(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	a.DependencySHA256 = first.Workspaces["repo"].DependencySHA256
	c.Automation.Workspaces["repo"] = a
	before := writePreviewDraft(t, path, c)
	if err := os.WriteFile(input, []byte("replaced dependency"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := PreviewAutomation(context.Background(), path); err == nil || len(out.Workspaces) != 0 {
		t.Fatal("preview silently replaced owner dependency pin")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed preview modified draft")
	}
}
