package worker

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"agent-board/internal/core"
)

// gitTestRunner runs the real Git binary directly with the same fixed flags as
// runGit; tests cannot start the supervised helper from the test binary.
func gitTestRunner(git string) gitRunner {
	return func(ctx context.Context, _ AutomationWorkspace, directory string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, git, append([]string{"--no-replace-objects", "-c", "core.fsmonitor=false", "-c", "core.untrackedcache=false"}, args...)...)
		cmd.Dir, cmd.Env = directory, gitEnv()
		out, err := cmd.Output()
		return bytes.TrimSuffix(out, []byte{'\n'}), err
	}
}

func gitRepo(t *testing.T, git string) string {
	t.Helper()
	repo, _ := filepath.EvalSymlinks(t.TempDir())
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.invalid"}, {"config", "user.name", "Test"}} {
		if out, err := exec.Command(git, append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	write(t, repo, "src/math_ops.py", "def add(a, b):\n    return a + b\n")
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "base"}} {
		if out, err := exec.Command(git, append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	return repo
}

func write(t *testing.T, root, relative, text string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInterruptedDeveloperWorkIsStashedOnlyForADifferentStep(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	e := testEngine(t)
	repo := gitRepo(t, git)
	e.config.Workspaces["repo"] = Workspace{Path: repo, Providers: []string{"codex"}}
	e.workspaceGit = gitTestRunner(git)
	ctx := context.Background()
	// The cancelled developer step left one modified and one new file.
	write(t, repo, "src/math_ops.py", "def add(a, b):\n    return a + b\n\ndef rem(a, b):\n    return a % b\n")
	write(t, repo, "tests/test_remainder.py", "def test_rem():\n    pass\n")
	dev := &core.ManagedAutomationBinding{WorkflowID: "wf_cancelled", StepID: "ws_dev"}
	work, err := e.observeInterruptedWork(ctx, AutomationWorkspace{}, repo, dev)
	if err != nil || work == nil || work.Files != 2 {
		t.Fatal("interrupted work was not fingerprinted", work, err)
	}
	e.journal.Runs["run_dev"] = &RunState{ID: "run_dev", Workspace: "repo", State: "stopped", Automation: &automationAttempt{Interrupted: work}}
	next := &RunState{ID: "run_next", Workspace: "repo"}
	e.mu.Lock()
	candidates := e.interruptedCandidatesLocked(next)
	e.mu.Unlock()
	// A resume of the same step keeps the agent's own edits untouched.
	if _, err = e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", dev, candidates); err == nil || !strings.HasPrefix(err.Error(), "workspace_dirty: ") || !strings.Contains(err.Error(), "same interrupted step") {
		t.Fatal("same-step resume touched or misreported its own edits", err)
	}
	// Files changed after the interruption are not provably that step's work.
	write(t, repo, "notes.txt", "human scratch\n")
	other := &core.ManagedAutomationBinding{WorkflowID: "wf_next", StepID: "ws_analyst"}
	if _, err = e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", other, candidates); err == nil || !strings.Contains(err.Error(), "3 uncommitted or untracked files") {
		t.Fatal("unattributable dirty files were not refused with a count", err)
	}
	if err = os.Remove(filepath.Join(repo, "notes.txt")); err != nil {
		t.Fatal(err)
	}
	result, err := e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", other, candidates)
	if err != nil || result == nil || !fullCommit(result.ref) {
		t.Fatal("interrupted work of another step was not preserved", err)
	}
	if out, _ := gitTestRunner(git)(ctx, AutomationWorkspace{}, repo, "status", "--porcelain"); len(out) != 0 {
		t.Fatal("checkout is still dirty after stash", string(out))
	}
	write(t, repo, "scratch.txt", "untracked only\n")
	if again, err := e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", other, nil); err != nil || again != nil {
		t.Fatal("untracked-only unattributed files were refused or touched", err)
	}
	if _, err = os.Stat(filepath.Join(repo, "scratch.txt")); err != nil {
		t.Fatal("untracked file was touched")
	}
	list, _ := gitTestRunner(git)(ctx, AutomationWorkspace{}, repo, "stash", "list")
	if !strings.Contains(string(list), "accord: interrupted wf_cancelled/ws_dev") {
		t.Fatal("stash message does not name the interrupted step", string(list))
	}
	e.mu.Lock()
	e.recordStashLocked(next, result)
	e.mu.Unlock()
	if e.journal.Runs["run_dev"].Automation.Interrupted.Stash != result.ref || !strings.Contains(next.Notice, result.ref) || next.LocalDiagnostic != "interrupted_work_stashed" {
		t.Fatal("stash ref was not recorded for the owner")
	}
}

func TestDirtySymlinkMakesOwnershipUnprovable(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	e := testEngine(t)
	repo := gitRepo(t, git)
	e.workspaceGit = gitTestRunner(git)
	if err = os.Symlink("/etc/hosts", filepath.Join(repo, "link")); err != nil {
		t.Skip("symlinks unavailable")
	}
	tree, err := readDirtyTree(context.Background(), AutomationWorkspace{}, repo, e.workspaceGit)
	if err != nil || tree.files != 1 || tree.provable {
		t.Fatal("symlinked dirty entry was treated as provable work", tree, err)
	}
	if work, _ := e.observeInterruptedWork(context.Background(), AutomationWorkspace{}, repo, &core.ManagedAutomationBinding{WorkflowID: "wf", StepID: "ws"}); work != nil {
		t.Fatal("unprovable tree was recorded for a later stash")
	}
}
