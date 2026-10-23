package worker

import (
	"agent-board/internal/automationipc"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeferredACKCannotSpawnAfterGlobalStopBarrier(t *testing.T) {
	for _, automated := range []bool{false, true} {
		name := "manual"
		if automated {
			name = "automated"
		}
		t.Run(name, func(t *testing.T) {
			command, run := testDispatch("start")
			var e *Engine
			if automated {
				var request automationipc.Request
				e, request = automationTestEngine(t)
				run = e.journal.Runs[request.RunID].Automation.Run
				_ = e.active[request.RunID].process.Stop()
				delete(e.active, request.RunID)
				delete(e.journal.Runs, request.RunID)
				delete(e.journal.Receipts, command.ID)
			} else {
				e = testEngine(t)
			}
			e.asyncApply = true
			entered, release := make(chan struct{}), make(chan struct{})
			e.beforeApply = func() { close(entered); <-release }
			var calls atomic.Int32
			e.start = func(launchSpec) (process, error) { calls.Add(1); return successfulFake(), nil }

			if err := e.Accept(command, run); err != nil {
				t.Fatal(err)
			}
			if err := e.Ack(*e.Pending()); err != nil {
				t.Fatal(err)
			}
			<-entered
			stopped := make(chan error, 1)
			go func() { stopped <- e.StopAll("global stop before deferred apply") }()
			deadline := time.Now().Add(time.Second)
			for {
				e.mu.Lock()
				epoch, stopping := e.stopEpoch, e.stopCalls
				e.mu.Unlock()
				if epoch > 0 && stopping > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("StopAll did not enter barrier")
				}
				time.Sleep(time.Millisecond)
			}
			other, next := testDispatch("start")
			other.ID = "another_command"
			other.RunID = "another_run"
			next.ID = "another_run"
			if e.Accept(other, next) == nil {
				t.Fatal("new admission while global StopAll was waiting")
			}
			close(release)
			if err := <-stopped; err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 0 {
				t.Fatal("deferred accepted plan launched after the global stop epoch")
			}
			if e.journal.Runs[run.ID].State != "failed" {
				t.Fatal("deferred no-child plan lacks an explicit cancelled-before-launch receipt")
			}
		})
	}
}
