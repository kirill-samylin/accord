package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-board/internal/core"
)

// A terminal failure without an attestable session (ExitUnreportable) that was
// delivered and acknowledged must stay failed and unreported across restarts.
func TestReviewUnreportableExitAfterAckIsStableAcrossRestart(t *testing.T) {
	e, request := automationTestEngine(t)
	e.automationReadTransport = func(_ context.Context, _, _ string, _ any, out any) error {
		*out.(*core.Task) = core.Task{Key: "T-1", Role: "tester", Generation: 3}
		return nil
	}
	lock, err := acquireWorkspace(e.config.Workspaces["test"].Path, e.workspaceLockDir, "worker run_test")
	if err != nil {
		t.Fatal(err)
	}
	run := e.journal.Runs[request.RunID]
	run.SessionID = ""
	run.State = "launching"
	e.active[run.ID].workspaceLock = lock
	child := e.active[run.ID].process
	_ = child.Stop()
	e.wg.Add(1)
	go e.observe(run.ID, child)
	e.wg.Wait()
	for p := e.Pending(); p != nil; p = e.Pending() {
		if err := e.Ack(*p); err != nil {
			t.Fatal(err)
		}
	}
	if run.State != "failed" || !run.Automation.ExitUnreportable || !run.Automation.LockReleased || run.Automation.ExitAck {
		t.Fatal("fixture did not produce an acknowledged unreportable exit", run.State, run.Automation.ExitUnreportable, run.Automation.LockReleased)
	}
	_ = e.Close()
	for i := 0; i < 2; i++ {
		e2, err := Open(e.config)
		if err != nil {
			t.Fatal(err)
		}
		r2 := e2.journal.Runs[run.ID]
		if err = e2.prepareReconciliation(); err != nil {
			t.Fatal(err)
		}
		var phases []string
		for _, p := range e2.journal.Pending {
			phases = append(phases, p.Phase+"/"+p.CommandID)
		}
		if len(phases) > 0 || r2.State != "failed" || r2.LocalDiagnostic != "native_exit_before_session" {
			t.Errorf("restart %d re-reports or rewrites an acknowledged terminal failure: %v state=%s diag=%s", i, phases, r2.State, r2.LocalDiagnostic)
		}
		_ = e2.Close()
	}
}

// statusServer answers /managed/report with a fixed status and error envelope.
func statusServer(t *testing.T, status int, code, message string, phases *[]string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/session"):
			_ = json.NewEncoder(w).Encode(core.SessionView{SessionID: "human_test", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/managed/report"):
			var in core.ManagedReportInput
			_ = json.NewDecoder(r.Body).Decode(&in)
			mu.Lock()
			*phases = append(*phases, in.Phase)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// A recovery report for an attempt Accord no longer has (the run was resumed
// to a newer generation, or archived) is settled locally: dropped and logged,
// never a fatal startup error, and never re-sent after a restart.
func TestRecoveryReportRejectedForStaleOrArchivedRunIsSettled(t *testing.T) {
	for _, tc := range []struct {
		name, code, message string
		status              int
	}{
		{"resumed_generation", "conflict", "Устаревшая generation запуска", 409},
		{"foreign_dispatch", "conflict", "Чужой или устаревший dispatch", 409},
		{"archived_run", "not_found", "Запуск не найден", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testEngine(t)
			e.journal.HostID = "host_test"
			run := &RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1, State: "stopped", CommandID: "command_start", Automation: &automationAttempt{NativeExited: true, ExitDurable: true, ExitUnreportable: true, ProofAbandoned: true, LockReleased: true}}
			e.journal.Runs[run.ID] = run
			e.journal.Receipts[run.CommandID] = &Receipt{CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
			var phases []string
			server := statusServer(t, tc.status, tc.code, tc.message, &phases)
			config := e.config
			e.config.Server = server.URL
			var logs []string
			e.SetLogger(func(line string) { logs = append(logs, line) })
			r, err := newRemote(e.config)
			if err != nil {
				t.Fatal(err)
			}
			if err = e.reconcile(context.Background(), r); err != nil {
				t.Fatal("a definitive rejection of a recovery report ended startup", err)
			}
			if strings.Join(phases, ",") != "recovered_stopped" || len(e.journal.Pending) != 0 || !run.RecoveryNotified || run.State != "stopped" {
				t.Fatal("rejected recovery report was not settled once", phases, len(e.journal.Pending), run.RecoveryNotified, run.State)
			}
			if len(logs) == 0 || !strings.Contains(strings.Join(logs, "\n"), "dropped") {
				t.Fatal("dropped recovery report was not logged", logs)
			}
			_ = e.Close()
			e2, err := Open(config)
			if err != nil {
				t.Fatal(err)
			}
			defer e2.Close()
			if err = e2.prepareReconciliation(); err != nil || len(e2.journal.Pending) != 0 || e2.journal.Runs[run.ID].State != "stopped" {
				t.Fatal("settled run was reported again after restart", err, len(e2.journal.Pending))
			}
		})
	}
	for _, transient := range []statusError{{code: 409, server: "conflict", message: "Прежний worker ещё владеет lease"}, {code: 409}, {code: 500}, {code: 403, message: "generation"}} {
		if definitiveRecoveryRejection(transient) {
			t.Fatal("a transient or authority rejection was treated as settled", transient)
		}
	}
}

// `worker recover` of an uncertain automation exit stays recovered across
// restarts and never leads to delivery_unknown afterwards. On Unix a run
// without recorded process groups needs the explicit --unverified mode.
func TestReviewRecoverIsDurableForUncertainAutomationExit(t *testing.T) {
	e, request := automationTestEngine(t)
	run := e.journal.Runs[request.RunID]
	e.journal.HostID = "host_test"
	run.State = "needs_recovery"
	run.Automation.NativeExited = true
	run.Automation.ExitDurable = false
	delete(e.active, run.ID)
	e.groupCensus = func([]int) ([]int, error) { return nil, nil }
	scans := 0
	e.processScan = func(map[string]bool) ([]localProcess, error) { scans++; return nil, nil }
	if recoveryNeedsGroups {
		if err := e.Recover(context.Background(), run.ID); !errors.Is(err, errNoRecordedGroups) || run.State != "needs_recovery" {
			t.Fatal("verified recovery without recorded process groups was not refused", err, run.State)
		}
	}
	// Delivery fails (no server); the recovered_stopped report stays queued.
	_ = e.RecoverRun(context.Background(), run.ID, RecoverOptions{Unverified: true})
	if run.State != "stopped" || run.LocalDiagnostic != "owner_recovered_unverified" || scans != 1 {
		t.Fatal("unverified recovery did not scan and stop the run", run.State, run.LocalDiagnostic, scans)
	}
	config := e.config
	_ = e.Close()
	for i := 0; i < 2; i++ {
		e2, err := Open(config)
		if err != nil {
			t.Fatal(err)
		}
		r2 := e2.journal.Runs[run.ID]
		if r2.State != "stopped" {
			t.Fatalf("restart %d undid worker recover: state=%s err=%s", i, r2.State, r2.Error)
		}
		if err = e2.prepareReconciliation(); err != nil {
			t.Fatal(err)
		}
		for _, p := range e2.journal.Pending {
			if p.Phase != "recovered_stopped" {
				t.Fatalf("restart %d queued %s for a recovered run", i, p.Phase)
			}
		}
		if i == 0 && len(e2.journal.Pending) != 1 {
			t.Fatal("queued recovered_stopped was lost", len(e2.journal.Pending))
		}
		if i == 1 && len(e2.journal.Pending) != 0 {
			t.Fatal("acknowledged recovered_stopped was queued again", len(e2.journal.Pending))
		}
		for p := e2.Pending(); p != nil; p = e2.Pending() {
			if err = e2.Ack(*p); err != nil {
				t.Fatal(err)
			}
		}
		_ = e2.Close()
	}
}

func TestUnverifiedRecoverRefusesPossiblyLiveAgent(t *testing.T) {
	workspace := "/work/repo"
	session := "019a0000-0000-7000-8000-000000000001"
	names := providerProcessNames(Config{Executables: map[string]string{"codex": "/opt/bin/codex-cli"}})
	for _, tc := range []struct {
		name   string
		procs  []localProcess
		refuse bool
	}{
		{"provider_cwd_in_workspace", []localProcess{{PID: 10, Args: []string{"/opt/bin/codex-cli", "exec"}, Cwd: workspace + "/src", CwdKnown: true}}, true},
		{"script_provider_names_workspace", []localProcess{{PID: 11, Args: []string{"node", "/usr/lib/claude", "--add-dir", workspace}, Cwd: "/", CwdKnown: true}}, true},
		{"any_process_names_session", []localProcess{{PID: 12, Args: []string{"sh", "-c", "resume " + session}, Cwd: "/", CwdKnown: true}}, true},
		{"provider_cwd_unknown", []localProcess{{PID: 13, Args: []string{"codex"}}}, true},
		{"provider_elsewhere", []localProcess{{PID: 14, Args: []string{"claude"}, Cwd: "/work/repo-other", CwdKnown: true}}, false},
		{"editor_in_workspace", []localProcess{{PID: 15, Args: []string{"vim", "x.go"}, Cwd: workspace, CwdKnown: true}}, false},
	} {
		err := refuseRunProcesses(tc.procs, workspace, session, names)
		if (err != nil) != tc.refuse {
			t.Fatal(tc.name, "wrong verdict", err)
		}
	}
	e := testEngine(t)
	e.journal.HostID = "host_test"
	run := &RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1, State: "needs_recovery", CommandID: "command_start", SessionID: session}
	e.journal.Runs[run.ID] = run
	e.journal.Receipts[run.CommandID] = &Receipt{CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
	e.groupCensus = func([]int) ([]int, error) { return nil, nil }
	e.processScan = func(map[string]bool) ([]localProcess, error) {
		return []localProcess{{PID: 77, Args: []string{"codex", "resume", session}}}, nil
	}
	if err := e.RecoverRun(context.Background(), run.ID, RecoverOptions{Unverified: true}); err == nil || run.State != "needs_recovery" || len(e.journal.Pending) != 0 {
		t.Fatal("unverified recovery ignored a live native process of the run", err, run.State)
	}
	e.processScan = func(map[string]bool) ([]localProcess, error) { return nil, errors.New("scan failed") }
	if err := e.RecoverRun(context.Background(), run.ID, RecoverOptions{Unverified: true}); err == nil || run.State != "needs_recovery" {
		t.Fatal("unverified recovery proceeded without a process scan", err)
	}
}

// The real read-only scanner sees a live process of this user and its
// working directory; nothing is signalled.
func TestLocalProcessScanSeesWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Job Objects replace the scan on Windows")
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep unavailable")
	}
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	child := exec.Command(sleep, "30")
	child.Dir = dir
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	names := map[string]bool{"sleep": true}
	processes, err := scanLocalProcesses(names)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range processes {
		if p.PID == child.Process.Pid {
			found = true
			if !p.CwdKnown {
				t.Skip("working directory of own processes is not readable here")
			}
			if cwd, _ := filepath.EvalSymlinks(p.Cwd); cwd != dir {
				t.Fatal("scanner reported a wrong working directory", p.Cwd, dir)
			}
		}
	}
	if !found {
		t.Fatal("scanner missed a live process of this user")
	}
	if err = refuseRunProcesses(processes, dir, "", names); err == nil {
		t.Fatal("a live provider-like process in the workspace was not refused")
	}
}

// While Accord refuses the lease because a run of this host needs recovery,
// an idle worker keeps polling with backoff and resumes once the lease is
// granted again; it neither exits nor launches anything meanwhile.
func TestIdleWorkerWaitsThroughRecoveryLeaseConflict(t *testing.T) {
	e := testEngine(t)
	e.deadline, e.journal.HostID = time.Time{}, "host_test"
	launches := 0
	e.start = func(launchSpec) (process, error) { launches++; return successfulFake(), nil }
	var mu sync.Mutex
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/session"):
			_ = json.NewEncoder(w).Encode(core.SessionView{SessionID: "human_test", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/heartbeat"):
			_ = json.NewEncoder(w).Encode(core.SessionView{SessionID: "human_test", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/managed/poll"):
			mu.Lock()
			polls++
			n := polls
			mu.Unlock()
			if n == 1 || n >= 5 {
				_ = json.NewEncoder(w).Encode(core.ManagedPollResult{WorkerEpoch: 3, LeaseExpiresAt: time.Now().Add(30 * time.Second)})
				return
			}
			w.WriteHeader(409)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "conflict", "message": "Ownership lease истекла; нужна локальная сверка остановки"}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	e.config.Server = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err := e.run(ctx, 10*time.Millisecond, 0)
	mu.Lock()
	defer mu.Unlock()
	if !errors.Is(err, context.DeadlineExceeded) || polls < 6 || launches != 0 || e.journal.Epoch != 3 {
		t.Fatal("idle worker did not wait through a recovery lease conflict", err, polls, launches)
	}
}

// A 403 for the started report of a run that is being stopped is dropped; the
// stop continues and other reports are still delivered.
func TestForbiddenStartedWhileStoppingDoesNotEndService(t *testing.T) {
	e := testEngine(t)
	run := &RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1, State: "running", CommandID: "command_start"}
	e.journal.Runs[run.ID] = run
	e.journal.Receipts[run.CommandID] = &Receipt{CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}
	e.active[run.ID] = &activeRun{process: &fakeProcess{output: io.NopCloser(strings.NewReader("")), done: make(chan struct{})}, stop: true, stopCommand: "command_stop"}
	defer func() {
		e.mu.Lock()
		delete(e.active, run.ID)
		e.mu.Unlock()
	}()
	e.journal.Pending = []*Receipt{{RequestID: "r1", CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}, {RequestID: "r2", CommandID: "command_stop", RunID: run.ID, DispatchID: "dispatch_stop", Generation: 1, Phase: "accepted"}}
	var sent []string
	post := func(_ context.Context, _ string, input, _ any) error {
		in := input.(core.ManagedReportInput)
		sent = append(sent, in.Phase+"/"+in.CommandID)
		if in.CommandID == run.CommandID {
			return statusError{code: 403, server: "forbidden", message: "Инициатор команды потерял разрешение управления"}
		}
		return nil
	}
	e.asyncApply = true
	if err := e.sendPendingReports(context.Background(), post, 8); err != nil {
		t.Fatal("a 403 for a non-terminal report of a stopping run ended the worker", err)
	}
	if strings.Join(sent, ",") != "started/command_start,accepted/command_stop" || len(e.journal.Pending) != 0 {
		t.Fatal("stop reports were blocked behind the refused started report", sent)
	}
	// Not stopping: the same refusal is still fatal.
	active := e.active[run.ID]
	active.stop = false
	e.journal.Pending = []*Receipt{{RequestID: "r3", CommandID: run.CommandID, RunID: run.ID, DispatchID: "dispatch_start", Generation: 1, Phase: "started"}}
	if err := e.sendPendingReports(context.Background(), post, 8); err == nil {
		t.Fatal("a 403 for a run that is not being stopped was ignored")
	}
}

// Interrupted work is never stashed through a Git filter, and the stash itself
// runs with an empty private hooks directory and fsmonitor disabled.
func TestStashRefusesFilteredFilesAndIsolatesHooks(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	e := testEngine(t)
	repo := gitRepo(t, git)
	ctx := context.Background()
	base := gitTestRunner(git)
	var stashArgs []string
	hooksEmpty := false
	e.workspaceGit = func(ctx context.Context, a AutomationWorkspace, directory string, args ...string) ([]byte, error) {
		for i, arg := range args {
			if arg == "stash" && i > 0 {
				stashArgs = append([]string{}, args...)
				for _, a := range args {
					if hooks, ok := strings.CutPrefix(a, "core.hooksPath="); ok {
						entries, err := os.ReadDir(hooks)
						hooksEmpty = err == nil && len(entries) == 0
					}
				}
			}
		}
		return base(ctx, a, directory, args...)
	}
	write(t, repo, "src/math_ops.py", "def add(a, b):\n    return b + a\n")
	work, err := e.observeInterruptedWork(ctx, AutomationWorkspace{}, repo, &core.ManagedAutomationBinding{WorkflowID: "wf_old", StepID: "ws_dev"})
	if err != nil || work == nil {
		t.Fatal(err)
	}
	candidates := []interruptedRef{{"run_dev", *work}}
	other := &core.ManagedAutomationBinding{WorkflowID: "wf_next", StepID: "ws_next"}
	// An info/attributes filter on the dirty file: refused, nothing stashed.
	attributes := filepath.Join(repo, ".git", "info", "attributes")
	if err = os.MkdirAll(filepath.Dir(attributes), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(attributes, []byte("*.py filter=owner\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", other, candidates); err == nil || !strings.HasPrefix(err.Error(), "workspace_dirty: ") || !strings.Contains(err.Error(), "filter") || stashArgs != nil {
		t.Fatal("a filtered dirty file was stashed or misreported", err, stashArgs)
	}
	if err = os.Remove(attributes); err != nil {
		t.Fatal(err)
	}
	result, err := e.prepareCleanCheckout(ctx, AutomationWorkspace{}, repo, "repo", other, candidates)
	if err != nil || result == nil {
		t.Fatal("unfiltered interrupted work was not stashed", err)
	}
	joined := strings.Join(stashArgs, " ")
	if !hooksEmpty || !strings.Contains(joined, "core.fsmonitor=false") || !strings.Contains(joined, "core.hooksPath=") {
		t.Fatal("stash was not isolated from repository hooks and fsmonitor", stashArgs)
	}
	next := &RunState{ID: "run_next", Workspace: "repo"}
	e.mu.Lock()
	e.recordStashLocked(next, result)
	e.mu.Unlock()
	if !strings.Contains(next.Notice, "git stash apply --index "+result.ref) {
		t.Fatal("restore hint does not keep the staged state", next.Notice)
	}
}
