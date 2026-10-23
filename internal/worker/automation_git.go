package worker

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GitProof describes filesystem bytes, including ignored input outside declared
// artifact/dependency roots. Git's assume-unchanged flags cannot hide edits.
type gitProof struct {
	Head, Manifest, Unknown string
	Dependencies            map[string]string
	Artifacts               map[string]string
}
type gitEntry struct{ mode, oid, path string }

func runGit(ctx context.Context, a AutomationWorkspace, directory string, args ...string) ([]byte, error) {
	if verifyExecutable(a.GitExecutable, a.GitSHA256) != nil {
		return nil, consentError()
	}
	child, e := startSupervised(launchSpec{executable: a.GitExecutable, directory: directory, args: append([]string{"--no-replace-objects", "-c", "core.fsmonitor=false", "-c", "core.untrackedcache=false"}, args...), env: gitEnv()})
	if e != nil {
		return nil, e
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = child.Stop()
		case <-done:
		}
	}()
	data, exceeded, e := captureProcess(child, 4<<20, true)
	wait := child.Wait()
	close(done)
	if errors.Is(wait, errOwnership) {
		return nil, wait
	}
	if e != nil || wait != nil || exceeded || ctx.Err() != nil {
		return nil, errors.New("fixed Git observation failed")
	}
	if runtime.GOOS != "windows" {
		data = bytes.TrimSuffix(data, []byte{'\n'})
	}
	return data, nil
}
func gitEnv() []string {
	var env []string
	for _, value := range providerEnv() {
		key, _, _ := strings.Cut(value, "=")
		if !strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			env = append(env, value)
		}
	}
	return append(env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_NO_LAZY_FETCH=1")
}

type gitRunner func(context.Context, AutomationWorkspace, string, ...string) ([]byte, error)

func observeGit(ctx context.Context, a AutomationWorkspace, directory string) (gitProof, error) {
	return observeGitWithRunner(ctx, a, directory, runGit)
}
func observeGitWithRunner(ctx context.Context, a AutomationWorkspace, directory string, read gitRunner) (gitProof, error) {
	var out gitProof
	top, e := read(ctx, a, directory, "rev-parse", "--show-toplevel")
	if e != nil {
		return out, e
	}
	canonical, e := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if e != nil || canonical != directory {
		return out, errors.New("workspace must be the exact Git root")
	}
	head, e := read(ctx, a, directory, "rev-parse", "--verify", "HEAD^{commit}")
	if e != nil {
		return out, e
	}
	out.Head = strings.TrimSpace(string(head))
	if !fullCommit(out.Head) {
		return out, errors.New("full Git HEAD required")
	}
	tree, e := read(ctx, a, directory, "ls-tree", "-rz", "--full-tree", "HEAD")
	if e != nil {
		return out, e
	}
	entries, e := parseGitTree(tree)
	if e != nil {
		return out, e
	}
	index, e := read(ctx, a, directory, "ls-files", "--stage", "-z")
	if e != nil {
		return out, e
	}
	if !indexMatches(index, entries) {
		return out, errors.New("Git index differs from HEAD")
	}
	for _, entry := range entries {
		if e = verifyGitEntry(directory, entry); e != nil {
			return out, e
		}
	}
	out.Manifest, out.Unknown, out.Dependencies, e = sourceManifest(ctx, directory, a, entries)
	if e != nil {
		return out, e
	}
	out.Artifacts, e = artifactPatches(ctx, directory, a)
	if e != nil {
		return out, e
	}
	return out, nil
}
func fullCommit(s string) bool {
	v, e := hex.DecodeString(s)
	return e == nil && (len(v) == 20 || len(v) == 32) && strings.ToLower(s) == s
}
func parseGitTree(raw []byte) (map[string]gitEntry, error) {
	out := map[string]gitEntry{}
	for _, record := range bytes.Split(raw, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, consentError()
		}
		meta := strings.Fields(string(parts[0]))
		path := string(parts[1])
		if len(meta) != 3 || meta[1] != "blob" || (meta[0] != "100644" && meta[0] != "100755") || !consentRelative(filepath.FromSlash(path)) || !fullCommit(meta[2]) || out[path].path != "" {
			return nil, errors.New("unsupported Git tree boundary")
		}
		out[path] = gitEntry{meta[0], meta[2], path}
	}
	return out, nil
}
func indexMatches(raw []byte, tree map[string]gitEntry) bool {
	seen := map[string]bool{}
	for _, record := range bytes.Split(raw, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return false
		}
		meta := strings.Fields(string(parts[0]))
		entry, ok := tree[string(parts[1])]
		if len(meta) != 3 || meta[2] != "0" || !ok || seen[entry.path] || meta[0] != entry.mode || meta[1] != entry.oid {
			return false
		}
		seen[entry.path] = true
	}
	return len(seen) == len(tree)
}
func verifyGitEntry(directory string, entry gitEntry) error {
	path, pathErr := safeRelativeFile(directory, filepath.FromSlash(entry.path))
	if pathErr != nil {
		return pathErr
	}
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Size() > 256<<20 {
		return errors.New("tracked source is unavailable or unsupported")
	}
	if runtime.GOOS != "windows" && ((info.Mode().Perm()&0111 != 0) != (entry.mode == "100755")) {
		return errors.New("tracked executable mode differs from HEAD")
	}
	file, e := os.Open(path)
	if e != nil {
		return e
	}
	defer file.Close()
	opened, e := file.Stat()
	if e != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return errors.New("tracked file identity changed")
	}
	var h io.Writer
	short := sha1.New()
	long := sha256.New()
	if len(entry.oid) == 40 {
		h = short
	} else {
		h = long
	}
	fmt.Fprintf(h, "blob %d\x00", info.Size())
	if _, e = io.Copy(h, io.LimitReader(file, 256<<20+1)); e != nil {
		return e
	}
	got := hex.EncodeToString(long.Sum(nil))
	if len(entry.oid) == 40 {
		got = hex.EncodeToString(short.Sum(nil))
	}
	if got != entry.oid {
		return errors.New("tracked filesystem bytes differ from HEAD")
	}
	return nil
}
func proofContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 20*time.Second)
}
func digestRecords(records []string) string {
	sort.Strings(records)
	h := sha256.New()
	for _, r := range records {
		fmt.Fprintln(h, strconv.Quote(r))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (e *Engine) observeAutomationGit(ctx context.Context, a AutomationWorkspace, directory string) (gitProof, error) {
	if e.automationGit != nil {
		return e.automationGit(ctx, a, directory)
	}
	return observeGit(ctx, a, directory)
}
