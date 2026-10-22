package worker

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-board/internal/core"
)

type fakeProcess struct {
	output io.ReadCloser
	done   chan struct{}
	once   sync.Once
	forced bool
}

func (p *fakeProcess) Output() io.ReadCloser { return p.output }
func (p *fakeProcess) Wait() error           { <-p.done; return nil }
func (p *fakeProcess) Stop() error           { p.once.Do(func() { p.output.Close(); close(p.done) }); return nil }
func (p *fakeProcess) Forced() bool          { return p.forced }

func testEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, _ = filepath.EvalSymlinks(executable)
	token := filepath.Join(dir, "token")
	if err = os.WriteFile(token, []byte("synthetic-private-token"), 0600); err != nil {
		t.Fatal(err)
	}
	config := Config{Server: "http://127.0.0.1:12345", Project: "p_00000000000000000000000000000001", TokenFile: token, StateDir: filepath.Join(dir, "state"), Name: "synthetic", Machine: "test-machine", Executables: map[string]string{"codex": executable}, Workspaces: map[string]Workspace{"test": {Path: dir, Providers: []string{"codex"}}}}
	e, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	e.discover = func(string, string) ([]string, error) { return []string{"accord_gui", "accord_managed"}, nil }
	e.workspaceLockDir = filepath.Join(dir, "workspace-locks")
	e.probe = func(Config) error { return nil }
	e.journal.OwnerPrincipalID = "principal_test"
	if err = e.SetLease(1, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func testDispatch(kind string) (core.ManagedCommand, core.ManagedRun) {
	run := core.ManagedRun{ID: "run_test", Provider: "codex", Workspace: "test", Prompt: "synthetic user instruction", Generation: 1, Profile: &core.ProfileSnapshot{ID: "profile_test", Version: 1, ProfileSettings: core.ProfileSettings{Provider: "openai", Role: "developer", Instructions: "synthetic profile"}}}
	command := core.ManagedCommand{ID: "command_" + kind, RunID: run.ID, Generation: 1, DispatchID: "dispatch_" + kind, Kind: kind, CreatedBy: "principal_test"}
	return command, run
}

func successfulFake() *fakeProcess {
	output := `{"type":"thread.started","thread_id":"` + syntheticSession + `"}` + "\n" + `{"type":"turn.started"}` + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":"synthetic done"}}` + "\n" + `{"type":"turn.completed"}` + "\n"
	done := make(chan struct{})
	close(done)
	return &fakeProcess{output: io.NopCloser(strings.NewReader(output)), done: done}
}

func TestAcceptedACKPrecedesLaunchAndReplayNeverRespawns(t *testing.T) {
	e := testEngine(t)
	calls := 0
	e.start = func(spec launchSpec) (process, error) {
		calls++
		if !strings.Contains(spec.prompt, "profile_id=\"profile_test\"") || !strings.Contains(spec.prompt, "profile_version=1") {
			t.Error("pinned profile not supplied")
		}
		args := strings.Join(spec.args, " ")
		if strings.Contains(args, "synthetic user instruction") || strings.Contains(args, "bypass") || !strings.Contains(args, `mcp_servers."accord_gui".enabled=false`) {
			t.Error("unsafe native invocation")
		}
		return successfulFake(), nil
	}
	command, run := testDispatch("start")
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("launched before accepted HTTP acknowledgement")
	}
	if err := e.Ack(*e.Pending()); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	if calls != 1 {
		t.Fatal("native invocation count mismatch")
	}
	for _, phase := range []string{"started", "completed"} {
		pending := e.Pending()
		if pending == nil || pending.Phase != phase {
			t.Fatalf("expected %s receipt", phase)
		}
		if err := e.Ack(*pending); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("same dispatch relaunched")
	}
	command.DispatchID = "different_dispatch"
	if err := e.Accept(command, run); err == nil {
		t.Fatal("accepted changed dispatch identity")
	}
}

func TestStopDispatchedBeforeStartACKCancelsSpawn(t *testing.T) {
	e := testEngine(t)
	e.start = func(launchSpec) (process, error) {
		t.Error("spawn despite pending stop")
		return nil, errors.New("unexpected")
	}
	start, run := testDispatch("start")
	stop, _ := testDispatch("stop")
	if err := e.Accept(start, run); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(stop, run); err != nil {
		t.Fatal(err)
	}
	for e.Pending() != nil {
		if err := e.Ack(*e.Pending()); err != nil {
			t.Fatal(err)
		}
	}
	if e.journal.Runs[run.ID].State != "stopped" {
		t.Fatal("run did not stop before launch")
	}
}

func TestFailedProviderOutputIsNotCompleted(t *testing.T) {
	e := testEngine(t)
	e.start = func(launchSpec) (process, error) {
		p := successfulFake()
		p.output = io.NopCloser(strings.NewReader(`{"type":"error","message":"synthetic auth failure"}` + "\n"))
		return p, nil
	}
	command, run := testDispatch("start")
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if err := e.Ack(*e.Pending()); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	if got := e.journal.Runs[run.ID].State; got != "failed" {
		t.Fatalf("reported %s", got)
	}
	if pending := e.Pending(); pending == nil || pending.Phase != "failed" {
		t.Fatal("missing provider failure report")
	}
}

func TestResumeRefusesForeignOrMissingNativeContext(t *testing.T) {
	e := testEngine(t)
	command, run := testDispatch("resume")
	e.journal.Runs[run.ID] = &RunState{ID: run.ID, Provider: "codex", Workspace: "test", State: "stopped", Generation: 1}
	if err := e.Accept(command, run); err != nil {
		t.Fatal(err)
	}
	if err := e.Ack(*e.Pending()); err != nil {
		t.Fatal(err)
	}
	if pending := e.Pending(); pending == nil || pending.Phase != "failed" || !strings.Contains(pending.Error, "resume_unavailable") {
		t.Fatal("missing exact-session refusal")
	}
}
