package worker

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gitFixtureEntry(path, contents string) gitEntry {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00%s", len(contents), contents)
	return gitEntry{"100644", hex.EncodeToString(h.Sum(nil)), path}
}
func TestSourceProofDetectsAssumeUnchangedBytesIgnoredInputsAndExclusionAbuse(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "src/main.go")
	if err := os.WriteFile(path, []byte("reviewed"), 0644); err != nil {
		t.Fatal(err)
	}
	entry := gitFixtureEntry("src/main.go", "reviewed")
	tracked := map[string]gitEntry{entry.path: entry}
	a := AutomationWorkspace{SourceRoots: []string{"src"}, ArtifactRoots: []string{".artifacts"}}
	if err := verifyGitEntry(root, entry); err != nil {
		t.Fatal(err)
	}
	manifest, unknown, _, err := sourceManifest(context.Background(), root, a, tracked)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "ignored-loader.js"), []byte("new executable input"), 0644); err != nil {
		t.Fatal(err)
	}
	next, nextUnknown, _, err := sourceManifest(context.Background(), root, a, tracked)
	if err != nil {
		t.Fatal(err)
	}
	if manifest == next || unknown == nextUnknown {
		t.Fatal("ignored/untracked source vanished from proof")
	}
	if err = os.WriteFile(path, []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	if verifyGitEntry(root, entry) == nil {
		t.Fatal("filesystem edit hidden by index flags")
	}
	a.ArtifactRoots = []string{"src"}
	if _, _, _, err = sourceManifest(context.Background(), root, a, tracked); err == nil {
		t.Fatal("tracked code excluded as an artifact")
	}
}
func TestGitTreeAndIndexRejectUnmergedSubmoduleAndPathTraversal(t *testing.T) {
	oid := strings.Repeat("a", 40)
	tree, err := parseGitTree([]byte("100644 blob " + oid + "\tsrc/a.go\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if !indexMatches([]byte("100644 "+oid+" 0\tsrc/a.go\x00"), tree) {
		t.Fatal("clean index rejected")
	}
	if indexMatches([]byte("100644 "+oid+" 2\tsrc/a.go\x00"), tree) {
		t.Fatal("unmerged index accepted")
	}
	for _, raw := range []string{"160000 commit " + oid + "\tvendor/x\x00", "120000 blob " + oid + "\tlink\x00", "100644 blob " + oid + "\t../escape\x00"} {
		if _, err := parseGitTree([]byte(raw)); err == nil {
			t.Fatal("unsupported Git tree accepted")
		}
	}
}
func TestReportPreparationRefusesLinkAndRemovesStaleReport(t *testing.T) {
	root := t.TempDir()
	if err := prepareReport(root, ".artifacts/tests/report.tap"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".artifacts/tests/report.tap")
	if err := os.WriteFile(path, []byte("stale passing report"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareReport(root, ".artifacts/tests/report.tap"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stale report can be reused")
	}
	if err := os.Symlink(filepath.Join(root, "outside"), path); err != nil {
		t.Skip("links unavailable")
	}
	if prepareReport(root, ".artifacts/tests/report.tap") == nil {
		t.Fatal("linked report accepted")
	}
}

func TestApprovedDependencyDigestAllowsOnlyInternalPinnedLinks(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(root, "deps/.bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "deps/tool.js"), []byte("pinned dependency"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "deps/.bin/tool")
	if err = os.Symlink("../tool.js", link); err != nil {
		t.Skip("links unavailable")
	}
	digest, err := AutomationDependencyDigest(context.Background(), root, "deps")
	if err != nil {
		t.Fatal(err)
	}
	a := AutomationWorkspace{SourceRoots: []string{"src"}, DependencyRoots: []string{"deps"}, DependencySHA256: map[string]string{"deps": digest}}
	if _, _, _, err = sourceManifest(context.Background(), root, a, map[string]gitEntry{}); err != nil {
		t.Fatal("approved internal dependency link rejected", err)
	}
	if err = os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "external.js"), []byte("unreviewed external input"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("../../external.js", link); err != nil {
		t.Fatal(err)
	}
	if _, err = AutomationDependencyDigest(context.Background(), root, "deps"); err == nil {
		t.Fatal("dependency link escaped its approved boundary")
	}
}
