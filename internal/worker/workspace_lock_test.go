package worker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckoutLockSpansStateDirectoriesAndAliases(t *testing.T) {
	workspace, locks := t.TempDir(), filepath.Join(t.TempDir(), "locks")
	first, err := acquireWorkspace(workspace, locks, "synthetic-first-worker run-first")
	if err != nil {
		t.Fatal(err)
	}
	if second, err := acquireWorkspace(filepath.Join(workspace, "."), locks, "synthetic-second-worker run-second"); err == nil {
		second.release(true)
		t.Fatal("same checkout admitted another worker")
	}
	if err = first.release(true); err != nil {
		t.Fatal(err)
	}
	second, err := acquireWorkspace(workspace, locks, "synthetic-second-worker run-second")
	if err != nil {
		t.Fatal("proven exit did not release checkout")
	}
	if err = second.release(false); err != nil {
		t.Fatal(err)
	}
	if third, err := acquireWorkspace(workspace, locks, "synthetic-third-worker run-third"); err == nil {
		third.release(true)
		t.Fatal("unresolved ownership silently cleared")
	}
}

func TestLocalPermissionPresetsRejectBypass(t *testing.T) {
	e := testEngine(t)
	for _, unsafe := range []string{"danger-full-access", "external-sandbox", "bypassPermissions"} {
		config := e.config
		config.CodexSandbox = unsafe
		if config.Validate() == nil {
			t.Fatal("unsafe Codex preset")
		}
	}
	for _, unsafe := range []string{"bypassPermissions", "dontAsk", "auto"} {
		config := e.config
		config.ClaudePermissionMode = unsafe
		if config.Validate() == nil {
			t.Fatal("unsafe Claude preset")
		}
	}
	config := e.config
	config.CodexSandbox = "workspace-write"
	config.ClaudePermissionMode = "acceptEdits"
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStateBindingCannotSendHostProofToAnotherOrigin(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	executable, _ := os.Executable()
	executable, _ = filepath.EvalSymlinks(executable)
	config := Config{Server: "https://example.invalid", Project: "p_00000000000000000000000000000001", StateDir: filepath.Join(dir, "state"), Name: "synthetic", Machine: "synthetic", Executables: map[string]string{"codex": executable}, Workspaces: map[string]Workspace{"test": {Path: dir, Providers: []string{"codex"}}}}
	engine, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Close(); err != nil {
		t.Fatal(err)
	}
	config.Server = "https://another.invalid"
	if other, err := Open(config); err == nil {
		other.Close()
		t.Fatal("local host proof could cross origins")
	}
}
