package worker

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stopFailure struct{ *fakeProcess }

func (p stopFailure) Stop() error { return errors.New("synthetic owned tree termination failure") }

func TestStopErrorIsImmediatelyUnknownAndNeverStopped(t *testing.T) {
	e := testEngine(t)
	command, run := testDispatch("stop")
	e.journal.Runs[run.ID] = &RunState{ID: run.ID, State: "running", CommandID: "command_start"}
	e.journal.Receipts[command.ID] = &Receipt{CommandID: command.ID, RunID: run.ID, DispatchID: command.DispatchID, Generation: 1, Phase: "accepted"}
	active := &activeRun{process: stopFailure{successfulFake()}}
	e.active[run.ID] = active
	e.stopOwned(run.ID, active, command.ID)
	if !active.uncertain || e.journal.Runs[run.ID].State != "needs_recovery" {
		t.Fatal("stop error was silently ignored")
	}
	if pending := e.Pending(); pending == nil || pending.Phase != "delivery_unknown" {
		t.Fatal("unknown process ownership was not reported")
	}
	delete(e.active, run.ID)
}

func TestRevokedLocalControllerCannotLaunchAcceptedCommand(t *testing.T) {
	e := testEngine(t)
	command, run := testDispatch("start")
	command.CreatedBy = "principal_colleague"
	if err := e.Accept(command, run); err == nil {
		t.Fatal("unconsented local controller admitted")
	}
	e.config.Controllers = []string{"principal_colleague"}
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	e.config.Controllers = nil
	e.start = func(launchSpec) (process, error) { t.Fatal("launch after local revocation"); return nil, nil }
	if err := e.Ack(*e.Pending()); err != nil {
		t.Fatal(err)
	}
	if pending := e.Pending(); pending == nil || pending.Phase != "failed" {
		t.Fatal("missing local consent failure")
	}
}

func TestUnconfiguredProfileUsesExplicitNativeSelection(t *testing.T) {
	e := testEngine(t)
	command, run := testDispatch("start")
	run.Profile.Provider = "unconfigured"
	calls := 0
	e.start = func(launchSpec) (process, error) { calls++; return successfulFake(), nil }
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if err := e.Ack(*e.Pending()); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	if calls != 1 || e.journal.Runs[run.ID].State != "completed" {
		t.Fatal("explicit locally allowed provider was rejected")
	}
}

func TestPathsRejectSymlinksAndRecheckPinnedIdentity(t *testing.T) {
	e := testEngine(t)
	root := e.config.Workspaces["test"].Path
	workspace := filepath.Join(root, "checkout")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "linked-checkout")
	if err := os.Symlink(workspace, alias); err != nil {
		t.Skip("OS account cannot create symlinks")
	}
	config := e.config
	config.Workspaces = map[string]Workspace{"test": {Path: alias, Providers: []string{"codex"}}}
	if err := config.Validate(); err == nil {
		t.Fatal("symlink workspace admitted")
	}
	e.config.Workspaces = map[string]Workspace{"test": {Path: workspace, Providers: []string{"codex"}}}
	pins, err := pinPaths(e.config)
	if err != nil {
		t.Fatal(err)
	}
	e.journal.PathPins = pins
	if err = e.checkPins("codex", "test"); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(workspace, workspace+"-old"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	if err = e.checkPins("codex", "test"); err == nil {
		t.Fatal("retargeted workspace identity admitted")
	}
}

func TestJournalQuotaRejectsBeforeReplacingReadableState(t *testing.T) {
	d, j, err := openDisk(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	j.Runs["synthetic"] = &RunState{ID: "synthetic", Output: strings.Repeat("x", 8<<20)}
	if err = d.save(j); err == nil {
		t.Fatal("oversized journal saved")
	}
	file, err := privateRead(filepath.Join(d.dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var original journal
	if err = json.NewDecoder(file).Decode(&original); err != nil {
		t.Fatal("previous journal became unreadable")
	}
	if len(original.Runs) != 0 {
		t.Fatal("quota failure changed durable state")
	}
}
