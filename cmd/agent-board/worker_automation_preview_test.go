package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/worker"
)

func TestWorkerAutomationPreviewDoesNotEnrollOrRequireTokenOrPreparedHashes(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "never-executed")
	if err := os.WriteFile(executable, []byte("pure configuration fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	c := worker.Config{Name: "preview", Machine: "preview", Server: "https://private-value.invalid", Project: "p_fixture",
		StateDir: filepath.Join(root, "no-state"), TokenFile: filepath.Join(root, "no-token"),
		Executables: map[string]string{"codex": executable}, CodexSandbox: "workspace-write",
		Workspaces:             map[string]worker.Workspace{"repo": {Path: root, Providers: []string{"codex"}}},
		NativeMCPApprovedTools: map[string][]string{"codex": {"describe", "next", "get", "transition", "workflow_question", "workflow_plan", "workflow_run_check"}},
		Automation: &worker.AutomationConfig{Version: 1, Workspaces: map[string]worker.AutomationWorkspace{"repo": {
			Version: 1, Enabled: true, ExpiresAt: time.Now().Add(time.Hour),
			Roles:             map[string][]string{"analyst": {"codex"}, "developer": {"codex"}, "reviewer": {"codex"}, "tester": {"codex"}},
			MaxAttemptSeconds: 60, MaxWorkflowSeconds: 240, MaxNativeInvocations: 4,
			GitExecutable: executable, SourceRoots: []string{"src"}, ArtifactRoots: []string{"reports"},
			Checks: map[string]worker.CheckPreset{"unit": {Version: 1, Enabled: true, Required: true, Kind: "test",
				Executable: executable, Args: []string{"private-argument"}, Directory: ".", TimeoutSeconds: 10,
				ReportFormat: "tap", ReportPath: "reports/tests.tap", Isolation: "trusted_owner_process"}},
		}}}}
	raw, _ := json.Marshal(c)
	path := filepath.Join(root, "draft.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := workerCommand([]string{"automation-preview", "--config", path}, &output); err != nil {
		t.Fatal(err)
	}
	var proposal worker.AutomationPreview
	if json.Unmarshal(output.Bytes(), &proposal) != nil || proposal.Applied || !proposal.RequiresLocalApply || len(proposal.Workspaces["repo"].Checks["unit"].Consent) != 64 {
		t.Fatal("missing unapplied preview")
	}
	if strings.Contains(output.String(), "private-value") || strings.Contains(output.String(), "private-argument") {
		t.Fatal("raw config escaped into command output")
	}
	for _, missing := range []string{c.TokenFile, c.StateDir} {
		if _, err := os.Stat(missing); !os.IsNotExist(err) {
			t.Fatal("preview opened live state", err)
		}
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, raw) {
		t.Fatal("preview rewrote config")
	}
}
