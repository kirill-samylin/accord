package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-board/internal/core"
)

func gitPreparationFixture(t *testing.T) (*Engine, plan, *activeRun) {
	t.Helper()
	e, request := automationTestEngine(t)
	r := e.journal.Runs[request.RunID]
	_ = e.active[request.RunID].process.Stop()
	e.active[request.RunID].process = nil
	r.Automation.Run.Profile.Role = "developer"
	e.config.CodexSandbox = "workspace-write"
	a := e.config.Automation.Workspaces["test"]
	a.CodexGitMetadataWrites = true
	e.config.Automation.Workspaces["test"] = a
	if err := os.Mkdir(filepath.Join(e.config.Workspaces["test"].Path, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	command, _ := testDispatch("start")
	return e, plan{command: command, run: r.Automation.Run}, e.active[request.RunID]
}
func TestGitPermissionPreparationRechecksAuthorityProofAndExactInvocation(t *testing.T) {
	e, p, active := gitPreparationFixture(t)
	calls := 0
	e.inspectPermissions = func(ctx context.Context, s launchSpec) (permissionProof, error) {
		calls++
		if !e.mu.TryLock() {
			t.Fatal("blocking inspection held global worker mutex")
		}
		e.mu.Unlock()
		if s.directory != e.config.Workspaces["test"].Path || s.inspection.Directory != s.directory || strings.Contains(strings.Join(s.args, " "), "sandbox_mode") || strings.Contains(strings.Join(s.args, " "), "enabled=true") {
			t.Fatal("inspection invocation widened or used wrong cwd")
		}
		return permissionProof{Digest: strings.Repeat("a", 64), ExecutableSHA256: s.inspection.ExecutableSHA256, GitIdentity: s.inspection.GitIdentity}, nil
	}
	e.mu.Lock()
	proof, args, err := e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), nil, []string{"SYNTHETIC=one"})
	e.mu.Unlock()
	if err != nil || calls != 1 || proof == nil || len(args) == 0 {
		t.Fatal("first proof", err)
	}
	e.mu.Lock()
	next, _, err := e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), proof, []string{"SYNTHETIC=one"})
	e.mu.Unlock()
	if err != nil || *next != *proof {
		t.Fatal("unchanged second proof rejected", err)
	}
	e.inspectPermissions = func(_ context.Context, s launchSpec) (permissionProof, error) {
		return permissionProof{Digest: strings.Repeat("b", 64), ExecutableSHA256: s.inspection.ExecutableSHA256, GitIdentity: s.inspection.GitIdentity}, nil
	}
	e.mu.Lock()
	_, _, err = e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), proof, nil)
	e.mu.Unlock()
	if err == nil {
		t.Fatal("permission-relevant TOCTOU change accepted")
	}
	e.inspectPermissions = func(_ context.Context, s launchSpec) (permissionProof, error) {
		e.mu.Lock()
		e.stopEpoch++
		e.mu.Unlock()
		return permissionProof{Digest: strings.Repeat("a", 64), ExecutableSHA256: s.inspection.ExecutableSHA256, GitIdentity: s.inspection.GitIdentity}, nil
	}
	e.mu.Lock()
	_, _, err = e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), proof, nil)
	e.mu.Unlock()
	if err == nil {
		t.Fatal("stop barrier was lost across config IO")
	}
}

func TestGitPermissionOwnershipFailureQuarantinesInsteadOfFallback(t *testing.T) {
	e, p, active := gitPreparationFixture(t)
	lock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "synthetic permission intent")
	if err != nil {
		t.Fatal(err)
	}
	active.workspaceLock = lock
	e.inspectPermissions = func(context.Context, launchSpec) (permissionProof, error) { return permissionProof{}, errOwnership }
	e.mu.Lock()
	_, _, err = e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), nil, nil)
	if !errors.Is(err, errOwnership) {
		e.mu.Unlock()
		t.Fatal("ownership lost", err)
	}
	err = e.failAutomationPreparationLocked(p, active, err)
	e.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if e.journal.Runs[p.run.ID].State != "needs_recovery" {
		t.Fatal("uncertainty became ordinary failure")
	}
	if next, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "must fail"); err == nil {
		next.release(true)
		t.Fatal("uncertain inspection released checkout intent")
	}
}

func TestNoGitPermissionInspectionForDefaultManualOrOtherRoles(t *testing.T) {
	e, p, active := gitPreparationFixture(t)
	called := false
	e.inspectPermissions = func(context.Context, launchSpec) (permissionProof, error) {
		called = true
		return permissionProof{}, nil
	}
	for _, role := range []string{"analyst", "reviewer", "tester"} {
		copy := p
		profile := *p.run.Profile
		profile.Role = role
		copy.run.Profile = &profile
		e.mu.Lock()
		proof, args, err := e.prepareGitPermissionsLocked(copy, active, e.stopEpoch, snapshotConfig(e.config), nil, nil)
		e.mu.Unlock()
		if err != nil || proof != nil || args != nil || called {
			t.Fatal("non-developer gained inspection/grant")
		}
	}
	p.run.Automation = nil
	e.mu.Lock()
	proof, _, err := e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), nil, nil)
	e.mu.Unlock()
	if err != nil || proof != nil || called {
		t.Fatal("manual gained inspection/grant")
	}
	p.run.Automation = &core.ManagedAutomationBinding{}
	a := e.config.Automation.Workspaces["test"]
	a.CodexGitMetadataWrites = false
	e.config.Automation.Workspaces["test"] = a
	e.mu.Lock()
	proof, _, err = e.prepareGitPermissionsLocked(p, active, e.stopEpoch, snapshotConfig(e.config), nil, nil)
	e.mu.Unlock()
	if err != nil || proof != nil || called {
		t.Fatal("default-off gained inspection/grant")
	}
}
