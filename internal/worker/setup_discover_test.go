package worker

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupFixture builds a fake installation tree: a real codex binary, a
// claude shell script (rejected), the npm launcher of gennadiy behind a PATH
// symlink with the platform binary beside it, and the ~/.gennadiy bash shim.
func setupFixture(t *testing.T) (root string, probe setupProbe) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	binary := []byte{0xCF, 0xFA, 0xED, 0xFE, 'n', 'a', 't', 'i', 'v', 'e'}
	write := func(path string, data []byte, mode os.FileMode) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	probe = setupProbe{home: filepath.Join(root, "home"), goos: runtime.GOOS, goarch: runtime.GOARCH}
	bin := filepath.Join(root, "prefix", "bin")
	write(filepath.Join(bin, probe.executableName("codex")), binary, 0755)
	write(filepath.Join(bin, probe.executableName("claude")), []byte("#!/bin/sh\nexec node claude.js\n"), 0755)
	launcher := filepath.Join(root, "prefix", "lib", "node_modules", "gennadiy", "bin", "gennadiy.cjs")
	write(launcher, []byte("#!/usr/bin/env node\n"), 0755)
	write(filepath.Join(root, "prefix", "lib", "node_modules", "gennadiy", "node_modules", probe.gennadiyPlatformPackage(), "bin", probe.executableName("gennadiy")), binary, 0755)
	if err := os.Symlink(launcher, filepath.Join(bin, probe.executableName("gennadiy"))); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(probe.home, ".gennadiy", "bin", "gennadiy")
	write(shim, []byte("#!/usr/bin/env bash\nexec gennadiy \"$@\"\n"), 0755)
	// A second PATH entry pointing at the same codex must not duplicate it.
	local := filepath.Join(probe.home, ".local", "bin")
	if err := os.MkdirAll(local, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(bin, probe.executableName("codex")), filepath.Join(local, probe.executableName("codex"))); err != nil {
		t.Fatal(err)
	}
	probe.pathDirs = []string{filepath.Dir(shim), local, bin}
	probe.version = func(provider, executable string) (string, error) {
		if provider == "gennadiy" {
			return "", os.ErrNotExist
		}
		return provider + " 9.9.9\nextra line", nil
	}
	return root, probe
}

func TestSetupDiscoverFindsNativeBinariesOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	root, probe := setupFixture(t)
	found := probe.discover()
	if got := found["codex"]; len(got) != 1 || got[0].Path != filepath.Join(root, "prefix", "bin", "codex") || got[0].Version != "codex 9.9.9" || got[0].Note != "" {
		t.Fatalf("codex discovery: %+v", got)
	}
	if got := found["claude"]; len(got) != 0 {
		t.Fatalf("claude shell script must be rejected: %+v", got)
	}
	want := filepath.Join(root, "prefix", "lib", "node_modules", "gennadiy", "node_modules", probe.gennadiyPlatformPackage(), "bin", "gennadiy")
	if got := found["gennadiy"]; len(got) != 1 || got[0].Path != want || got[0].Note == "" || got[0].Version != "версия не прочитана" {
		t.Fatalf("gennadiy launcher must resolve to the platform binary: %+v", got)
	}
	if text := found["gennadiy"][0].describe(); !strings.Contains(text, want) || !strings.Contains(text, probe.gennadiyPlatformPackage()) {
		t.Fatalf("describe: %s", text)
	}
}

func TestSetupResolveRejectsShimsAndSymlinkedDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	root, probe := setupFixture(t)
	if path, _ := probe.resolve("gennadiy", filepath.Join(probe.home, ".gennadiy", "bin", "gennadiy")); path != "" {
		t.Fatalf("bash shim resolved to %q", path)
	}
	if path, _ := probe.resolve("codex", filepath.Join(root, "missing", "codex")); path != "" {
		t.Fatal("missing file resolved")
	}
	if path, _ := probe.resolve("claude", filepath.Join(root, "prefix", "bin", "claude")); path != "" {
		t.Fatal("script accepted for claude")
	}
	if path, _ := probe.resolve("codex", "prefix/bin/codex"); path != "" {
		t.Fatal("relative path accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "prefix", "bin", "codex")); err != nil {
		t.Fatal(err)
	}
	if path, _ := probe.resolve("codex", filepath.Join(root, "home", ".local", "bin", "codex")); path != filepath.Join(root, "prefix", "bin", "codex") {
		t.Fatalf("symlinked candidate must resolve to its canonical path, got %q", path)
	}
	if got := (setupProbe{goos: "darwin", goarch: "amd64"}).gennadiyPlatformPackage(); got != "gennadiy-darwin-x64" {
		t.Fatal(got)
	}
	if got := (setupProbe{goos: "windows"}).executableName("claude"); got != "claude.exe" {
		t.Fatal(got)
	}
}
