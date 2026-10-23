package worker

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"agent-board/internal/core"
)

func TestWorkflowProofQueueSurvivesRestartAndNeverAdoptsOldExecution(t *testing.T) {
	e, request := automationTestEngine(t)
	r := e.journal.Runs[request.RunID]
	r.State = "completed"
	r.Automation.NativeExited = true
	r.Automation.ExitDurable = true
	r.Automation.Pending = []*core.WorkflowAttestInput{{ManagedPollInput: core.ManagedPollInput{RequestID: "pending-proof", WorkerID: e.journal.WorkerID, WorkerEpoch: 1}, RunID: r.ID, Kind: "run_exit"}}
	if err := e.save(); err != nil {
		t.Fatal(err)
	}
	stateDir := e.config.StateDir
	e.disk.close()
	e.closed = true
	disk, j, err := openDisk(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	defer disk.close()
	restored := j.Runs[r.ID]
	if restored.State != "needs_recovery" || len(restored.Automation.Pending) != 1 || restored.Automation.Pending[0].RequestID != "pending-proof" || restored.Automation.Pending[0].WorkerID == j.WorkerID {
		t.Fatal("restart discarded or reauthorized an unacknowledged proof")
	}
}
func TestPassingExitAndReportCannotQualifyChangedReviewedSource(t *testing.T) {
	e, request := automationTestEngine(t)
	var observations atomic.Int32
	baseline := e.journal.Runs[request.RunID].Automation.Baseline
	e.automationGit = func(context.Context, AutomationWorkspace, string) (gitProof, error) {
		copy := baseline
		if observations.Add(1) > 1 {
			copy.Manifest = strings.Repeat("d", 64)
		}
		return copy, nil
	}
	e.start = func(launchSpec) (process, error) {
		report := filepath.Join(e.config.Workspaces["test"].Path, ".artifacts/test.tap")
		if err := os.WriteFile(report, []byte("1..1\nok 1\n"), 0600); err != nil {
			return nil, err
		}
		done := make(chan struct{})
		close(done)
		return &statusFake{&fakeProcess{output: io.NopCloser(strings.NewReader("")), done: done}, 0}, nil
	}
	if _, err := e.automationCheckRequest(context.Background(), request.RunID, request); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	c := e.journal.Runs[request.RunID].Automation.Checks[request.RequestID]
	if c.Observations.SourceClean == nil || *c.Observations.SourceClean || c.ErrorCode != "source_changed" {
		t.Fatal("changed reviewed source was treated as a valid passing check")
	}
	if err := e.flushAutomation(context.Background()); err != nil {
		t.Fatal(err)
	}
	request.Operation = "status"
	result, err := e.automationCheckRequest(context.Background(), request.RunID, request)
	if err != nil || result.Status != "not_run" {
		t.Fatal(result, err)
	}
}
func TestTesterArtifactConsentIsSeparateAndSourceProofStillApplies(t *testing.T) {
	f := newAutomationFixture(t)
	a := f.config.Workspaces["repo"]
	a.Roles["tester"] = []string{"codex"}
	f.config.Workspaces["repo"] = a
	c := Config{Automation: f.config, CodexSandbox: "workspace-write", NativeMCPApprovedTools: map[string][]string{"codex": append([]string{}, managedMCPTools...), "claude": append([]string{}, managedMCPTools...)}}
	if got := automationSandbox(c, "tester", "repo"); got != "read-only" {
		t.Fatal("manual write consent silently enabled tester edits")
	}
	a.AllowTesterArtifactWrites = true
	f.config.Workspaces["repo"] = a
	if got := automationSandbox(c, "tester", "repo"); got != "workspace-write" {
		t.Fatal("explicit artifact design consent ignored")
	}
	if got := automationSandbox(c, "reviewer", "repo"); got != "read-only" {
		t.Fatal("reviewer widened")
	}
	c.CodexSandbox = "read-only"
	if validateAutomationNativeConsent(c) == nil {
		t.Fatal("artifact write flag bypassed native sandbox consent")
	}
	if f.validate() == nil {
		t.Fatal("changing artifact write scope reused old preset acknowledgement")
	}
	before := gitProof{Head: strings.Repeat("a", 40), Manifest: "before", Unknown: "same"}
	after := before
	after.Manifest = "mutated"
	if unchangedProof(before, after, true) == nil {
		t.Fatal("artifact consent allowed source mutation")
	}
	valid := []byte("diff --git a/src/check_test.go b/src/check_test.go\n--- a/src/check_test.go\n+++ b/src/check_test.go\n@@ -1 +1 @@\n-old\n+new\n")
	if !validDesignPatch(valid, []string{"src"}) {
		t.Fatal("bounded test design rejected")
	}
	if validDesignPatch([]byte("+++ b/../secrets\n@@ -1 +1 @@\n"), []string{"src"}) {
		t.Fatal("artifact patch escaped approved source roots")
	}
}
func TestNativeAutomationConsentDoesNotExpandManualToolGrants(t *testing.T) {
	f := newAutomationFixture(t)
	c := Config{Automation: f.config, NativeMCPApprovedTools: map[string][]string{"codex": {"describe", "next"}, "claude": {"describe", "next"}}}
	if validateAutomationNativeConsent(c) == nil {
		t.Fatal("workflow tools implicitly approved")
	}
	c.Automation = nil
	if err := validateAutomationNativeConsent(c); err != nil {
		t.Fatal("legacy manual consent changed", err)
	}
	if err := validateNativeMCPConsent(map[string][]string{"codex": {"workflow_plan", "workflow_question", "workflow_run_check"}}, map[string]string{"codex": "local"}); err != nil {
		t.Fatal("explicit new operation opt-in rejected", err)
	}
	if err := validateNativeMCPConsent(map[string][]string{"codex": {"workflow_submit"}}, map[string]string{"codex": "local"}); err == nil {
		t.Fatal("child recursively received workflow intake authority")
	}
}
