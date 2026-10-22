package worker

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type watchdogStop struct {
	entered chan struct{}
	once    sync.Once
}

func (p *watchdogStop) Output() io.ReadCloser { return io.NopCloser(strings.NewReader("")) }
func (p *watchdogStop) Wait() error           { return nil }
func (p *watchdogStop) Forced() bool          { return false }
func (p *watchdogStop) Stop() error           { p.once.Do(func() { close(p.entered) }); return nil }

func TestBlockedConfigurationCannotStarveAnotherRunWatchdog(t *testing.T) {
	e := testEngine(t)
	directory := filepath.Join(e.config.Workspaces["test"].Path, "other-checkout")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	e.config.Workspaces["other"] = Workspace{Path: directory, Providers: []string{"codex"}}
	child := &watchdogStop{entered: make(chan struct{})}
	e.active["already_running"] = &activeRun{process: child}
	e.journal.Runs["already_running"] = &RunState{ID: "already_running", Provider: "codex", Workspace: "other", State: "running"}
	verificationEntered, releaseVerification := make(chan struct{}), make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(releaseVerification) })
	e.verifyMCP = func(string, string, []string, string, []string) error {
		close(verificationEntered)
		<-releaseVerification
		return nil
	}
	var launches atomic.Int32
	e.start = func(launchSpec) (process, error) { launches.Add(1); return successfulFake(), nil }
	command, run := testDispatch("start")
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if err := e.SetLease(1, time.Now().Add(500*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- e.Ack(*e.Pending()) }()
	select {
	case <-verificationEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("configuration verification did not enter")
	}
	ctx, cancel := context.WithCancel(context.Background())
	watchDone := make(chan struct{})
	go func() { defer close(watchDone); e.watchdog(ctx) }()
	defer func() { cancel(); <-watchDone }()
	select {
	case <-child.entered:
	case <-time.After(2 * time.Second):
		release.Do(func() { close(releaseVerification) })
		<-done
		t.Fatal("blocked verification starved the expired ownership watchdog")
	}
	if launches.Load() != 0 {
		t.Fatal("new launch occurred while verification was still blocked")
	}
	release.Do(func() { close(releaseVerification) })
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if launches.Load() != 0 {
		t.Fatal("expired launch was not rechecked after configuration verification")
	}
}

func TestConfigurationSnapshotDoesNotShareMutableAllowlists(t *testing.T) {
	e := testEngine(t)
	e.config.Models = map[string][]string{"codex": {"synthetic-model"}}
	e.config.Controllers = []string{"synthetic-controller"}
	snapshot := snapshotConfig(e.config)
	e.config.Controllers[0] = "changed"
	e.config.Models["codex"][0] = "changed"
	e.config.Executables["codex"] = "changed"
	workspace := e.config.Workspaces["test"]
	workspace.Providers[0] = "changed"
	if snapshot.Controllers[0] != "synthetic-controller" || snapshot.Models["codex"][0] != "synthetic-model" || snapshot.Executables["codex"] == "changed" || snapshot.Workspaces["test"].Providers[0] != "codex" {
		t.Fatal("native preparation shared a mutable configuration allowlist")
	}
}
