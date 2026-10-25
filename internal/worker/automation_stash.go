package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-board/internal/core"
)

// Interrupted developer work. When a developer step is stopped (cancel, pause,
// local deadline, shutdown) and its native exit is proven, the worker records a
// fingerprint of the uncommitted tree: HEAD plus every porcelain status entry
// and the exact bytes of each changed file. A later launch preserves that work
// with `git stash push --include-untracked` only when:
//   - the current tree matches a recorded fingerprint byte for byte (so the
//     files provably are that step's work and nobody touched them since), and
//   - the launching step differs from the interrupted one. Accord keeps a
//     workspace reserved for a paused workflow, so another workflow can only be
//     dispatched there after the interrupted one completed or was cancelled; a
//     resume of the same step keeps the agent's own edits untouched.
// Symlinks, directories (submodules, nested repositories) and unmerged entries
// make ownership unprovable; the worker then touches nothing and fails with
// workspace_dirty. Nothing is ever reset or deleted: the stash keeps the work.

type interruptedWork struct {
	WorkflowID  string `json:"workflow_id"`
	StepID      string `json:"step_id"`
	Head        string `json:"head"`
	Fingerprint string `json:"fingerprint"`
	Files       int    `json:"files"`
	Stash       string `json:"stash,omitempty"`
}
type dirtyTree struct {
	head, fingerprint string
	files, tracked    int // tracked counts entries other than untracked "??"
	provable          bool
	paths             []string // slash paths of every changed entry
}
type interruptedRef struct {
	runID string
	work  interruptedWork
}
type stashResult struct {
	runID, ref, message string
	files               int
}

// gitRunnerFor returns the fixed Git runner. A test seam that replaces the
// whole source proof (automationGit) without a workspace runner observes a
// clean tree, so synthetic proofs never spawn the real supervised helper.
func (e *Engine) gitRunnerFor() gitRunner {
	if e.workspaceGit != nil {
		return e.workspaceGit
	}
	if e.automationGit != nil {
		return cleanTreeRunner
	}
	return runGit
}

func cleanTreeRunner(_ context.Context, _ AutomationWorkspace, _ string, args ...string) ([]byte, error) {
	if len(args) > 0 && args[0] == "rev-parse" {
		return []byte("0000000000000000000000000000000000000000"), nil
	}
	return nil, nil
}

func readDirtyTree(ctx context.Context, a AutomationWorkspace, directory string, read gitRunner) (dirtyTree, error) {
	out := dirtyTree{provable: true}
	head, err := read(ctx, a, directory, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return out, err
	}
	out.head = strings.TrimSpace(string(head))
	status, err := read(ctx, a, directory, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames", "--ignore-submodules=none")
	if err != nil {
		return out, err
	}
	var records []string
	var total int64
	for _, raw := range bytes.Split(status, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		out.files++
		if out.files > 10000 || len(raw) < 4 || raw[2] != ' ' {
			out.provable = false
			out.tracked++
			continue
		}
		code, path := string(raw[:2]), string(raw[3:])
		out.paths = append(out.paths, path)
		if code != "??" {
			out.tracked++
		}
		hash, size, ok := dirtyEntryHash(directory, code, path)
		total += size
		if !ok || total > 1<<30 {
			out.provable = false
			continue
		}
		records = append(records, code+"\x00"+path+"\x00"+hash)
	}
	sort.Strings(records)
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s", out.head, strings.Join(records, "\n"))
	out.fingerprint = hex.EncodeToString(h.Sum(nil))
	return out, nil
}

// dirtyEntryHash hashes one changed regular file (or records its deletion).
func dirtyEntryHash(directory, code, slashPath string) (string, int64, bool) {
	if strings.ContainsRune(code, 'U') || code == "AA" || code == "DD" || strings.HasSuffix(slashPath, "/") {
		return "", 0, false
	}
	relative := filepath.FromSlash(slashPath)
	if !consentRelative(relative) {
		return "", 0, false
	}
	if parent := filepath.Dir(relative); parent != "." {
		if _, err := safeRelativeFile(directory, parent); err != nil {
			return "", 0, false
		}
	}
	path := filepath.Join(directory, relative)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "deleted", 0, true
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 256<<20 {
		return "", 0, false
	}
	file, err := os.Open(path)
	if err != nil {
		return "", 0, false
	}
	defer file.Close()
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(file, 256<<20+1)); err != nil {
		return "", 0, false
	}
	return fmt.Sprintf("%o:%s", info.Mode().Perm(), hex.EncodeToString(h.Sum(nil))), info.Size(), true
}

// interruptedCandidatesLocked lists unstashed interrupted work recorded for the
// same checkout by runs without a live handle.
func (e *Engine) interruptedCandidatesLocked(run *RunState) []interruptedRef {
	var out []interruptedRef
	if run.Automation != nil && run.Automation.Interrupted != nil && run.Automation.Interrupted.Stash == "" {
		out = append(out, interruptedRef{run.ID, *run.Automation.Interrupted})
	}
	for id, other := range e.journal.Runs {
		if id == run.ID || other.Automation == nil || other.Automation.Interrupted == nil || other.Automation.Interrupted.Stash != "" || e.active[id] != nil {
			continue
		}
		if e.workspacePath(other.Workspace) == e.workspacePath(run.Workspace) && (other.State == "stopped" || other.State == "completed" || other.State == "failed") {
			out = append(out, interruptedRef{id, *other.Automation.Interrupted})
		}
	}
	return out
}

// prepareCleanCheckout runs without Engine.mu under the attempt's checkout lock.
func (e *Engine) prepareCleanCheckout(ctx context.Context, a AutomationWorkspace, directory, alias string, binding *core.ManagedAutomationBinding, candidates []interruptedRef) (*stashResult, error) {
	read := e.gitRunnerFor()
	tree, err := readDirtyTree(ctx, a, directory, read)
	if err != nil {
		if errors.Is(err, errOwnership) {
			return nil, err
		}
		return nil, errSource
	}
	if tree.files == 0 {
		return nil, nil
	}
	var match *interruptedRef
	for i := range candidates {
		if tree.provable && candidates[i].work.Fingerprint == tree.fingerprint {
			match = &candidates[i]
		}
	}
	// Untracked-only trees stay as before: the source proof covers them and
	// nothing unattributed is touched. Tracked changes would fail the proof.
	if match == nil || match.work.StepID == binding.StepID {
		if tree.tracked == 0 {
			return nil, nil
		}
		return nil, dirtyWorkspaceCause(alias, tree.files, match != nil)
	}
	// A stash runs clean filters on the dirty files and smudge filters when it
	// restores the index; both are owner-configured programs. Such files are
	// never stashed by the worker.
	filtered, err := filteredPaths(ctx, a, directory, read, tree.paths)
	if err != nil {
		if errors.Is(err, errOwnership) {
			return nil, err
		}
		return nil, errSource
	}
	if filtered > 0 {
		return nil, filteredWorkspaceCause(alias, filtered)
	}
	// Hooks resolve to an empty private directory and fsmonitor is off for
	// this one invocation, so the stash executes no repository-configured
	// program.
	hooks, err := os.MkdirTemp("", "accord-nohooks-")
	if err != nil {
		return nil, cause("stash_failed", "could not prepare an isolated Git invocation; nothing was changed", "не удалось подготовить изолированный вызов Git; ничего не изменено")
	}
	defer os.RemoveAll(hooks)
	message := "accord: interrupted " + match.work.WorkflowID + "/" + match.work.StepID
	if _, err = read(ctx, a, directory, "-c", "core.hooksPath="+hooks, "-c", "core.fsmonitor=false", "-c", "commit.gpgSign=false", "-c", "user.name=Accord worker", "-c", "user.email=accord-worker@localhost", "stash", "push", "--include-untracked", "-m", message); err != nil {
		if errors.Is(err, errOwnership) {
			return nil, err
		}
		return nil, cause("stash_failed", "could not preserve interrupted work in a Git stash; nothing was deleted", "не удалось сохранить прерванную работу в git stash; ничего не удалено")
	}
	result := &stashResult{runID: match.runID, message: message, files: tree.files}
	ref, refErr := read(ctx, a, directory, "rev-parse", "--verify", "refs/stash")
	if refErr == nil && fullCommit(strings.TrimSpace(string(ref))) {
		result.ref = strings.TrimSpace(string(ref))
	}
	after, err := readDirtyTree(ctx, a, directory, read)
	if errors.Is(err, errOwnership) {
		return result, err
	}
	if err != nil || after.files != 0 || after.head != tree.head || result.ref == "" {
		return result, cause("stash_incomplete", "interrupted work was stashed but the checkout is still not clean; inspect `git stash list`", "прерванная работа сохранена в stash, но checkout всё ещё не чистый; проверьте `git stash list`")
	}
	return result, nil
}

// recordStashLocked records where interrupted work went, on both runs, so the
// owner can restore it; the notice also reaches this run's terminal report.
func (e *Engine) recordStashLocked(run *RunState, result *stashResult) {
	if result == nil {
		return
	}
	ref := result.ref
	if ref == "" {
		ref = "stash@{0}"
	}
	if source := e.journal.Runs[result.runID]; source != nil && source.Automation != nil && source.Automation.Interrupted != nil {
		source.Automation.Interrupted.Stash = ref
		source.LocalDiagnostic = "interrupted_work_stashed"
	}
	if run.Automation != nil && result.runID == run.ID && run.Automation.Interrupted != nil {
		run.Automation.Interrupted.Stash = ref
	}
	run.LocalDiagnostic = "interrupted_work_stashed"
	run.Notice = fmt.Sprintf("accord: %d changed files of %q were preserved in git stash %s; restore them with `git stash apply --index %s`", result.files, result.message, ref, ref)
	if e.logger != nil {
		e.logger(run.Notice)
	}
	_ = e.save()
}

// filteredPaths counts changed paths with a Git filter attribute (any value
// other than unspecified/unset). check-attr only reads attributes; it never
// runs a filter.
func filteredPaths(ctx context.Context, a AutomationWorkspace, directory string, read gitRunner, paths []string) (int, error) {
	count := 0
	for start := 0; start < len(paths); start += 100 {
		end := start + 100
		if end > len(paths) {
			end = len(paths)
		}
		args := append([]string{"check-attr", "-z", "filter", "--"}, paths[start:end]...)
		out, err := read(ctx, a, directory, args...)
		if err != nil {
			return 0, err
		}
		fields := bytes.Split(out, []byte{0})
		for i := 0; i+2 < len(fields); i += 3 {
			if value := string(fields[i+2]); value != "unspecified" && value != "unset" {
				count++
			}
		}
	}
	return count, nil
}
