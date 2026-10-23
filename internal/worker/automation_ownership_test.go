package worker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGitIndexObservationPreservesOwnershipAndRetainsCheckoutIntent(t *testing.T) {
	e, request := automationTestEngine(t)
	directory := e.config.Workspaces["test"].Path
	expected := ownershipFailureAt("synthetic_index_ownership")
	read := func(_ context.Context, _ AutomationWorkspace, _ string, args ...string) ([]byte, error) {
		switch args[0] {
		case "rev-parse":
			if args[1] == "--show-toplevel" {
				return []byte(directory + "\n"), nil
			}
			return []byte(strings.Repeat("a", 40) + "\n"), nil
		case "ls-tree":
			return []byte{}, nil
		case "ls-files":
			return nil, expected
		}
		t.Fatal("unexpected fixed Git command")
		return nil, nil
	}
	_, err := observeGitWithRunner(context.Background(), AutomationWorkspace{}, directory, read)
	if !errors.Is(err, errOwnership) {
		t.Fatal("ls-files ownership was replaced with an ordinary dirty-index error", err)
	}
	active := e.active[request.RunID]
	_ = active.process.Stop()
	active.process = nil
	lock, lockErr := acquireWorkspace(directory, e.workspaceLockDir, "synthetic ownership intent")
	if lockErr != nil {
		t.Fatal(lockErr)
	}
	active.workspaceLock = lock
	command, managed := testDispatch("start")
	e.mu.Lock()
	saveErr := e.failAutomationPreparationLocked(plan{command: command, run: managed}, active, err)
	e.mu.Unlock()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	if e.journal.Runs[request.RunID].State != "needs_recovery" {
		t.Fatal("ambiguous Git process became a clean preparation failure")
	}
	if next, err := acquireWorkspace(directory, e.workspaceLockDir, "must not acquire"); err == nil {
		next.release(true)
		t.Fatal("ambiguous Git proof cleared the durable checkout intent")
	}
}
func TestPostCheckGitOwnershipNeverProducesFinishedProofOrFreesCheckout(t *testing.T) {
	e, request := automationTestEngine(t)
	directory := e.config.Workspaces["test"].Path
	lock, err := acquireWorkspace(directory, e.workspaceLockDir, "post-check ownership intent")
	if err != nil {
		t.Fatal(err)
	}
	e.active[request.RunID].workspaceLock = lock
	baseline := e.journal.Runs[request.RunID].Automation.Baseline
	var reads atomic.Int32
	e.automationGit = func(context.Context, AutomationWorkspace, string) (gitProof, error) {
		if reads.Add(1) > 1 {
			return gitProof{}, ownershipFailureAt("synthetic_postcheck_ownership")
		}
		return baseline, nil
	}
	e.start = func(launchSpec) (process, error) {
		report := filepath.Join(directory, ".artifacts/test.tap")
		if err := os.WriteFile(report, []byte("1..1\nok 1\n"), 0600); err != nil {
			return nil, err
		}
		return &statusFake{successfulFake(), 0}, nil
	}
	if _, err = e.automationCheckRequest(context.Background(), request.RunID, request); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	e.mu.Lock()
	run := e.journal.Runs[request.RunID]
	uncertain := e.active[request.RunID].uncertain
	state := run.State
	pending := len(run.Automation.Pending)
	check := run.Automation.Checks[request.RequestID]
	checkState := check.Status
	e.mu.Unlock()
	if !uncertain || state != "needs_recovery" || checkState != "needs_recovery" || pending != 0 {
		t.Fatal("unproven Git child was turned into a negative finished check", state, checkState, pending)
	}
	if err = e.Close(); err != nil && !errors.Is(err, errOwnership) {
		t.Fatal(err)
	}
	if next, err := acquireWorkspace(directory, e.workspaceLockDir, "must not acquire"); err == nil {
		next.release(true)
		t.Fatal("post-check ownership uncertainty released checkout intent")
	}
}
