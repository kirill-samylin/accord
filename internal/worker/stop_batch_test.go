package worker

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type barrierStop struct {
	entered chan<- struct{}
	release <-chan struct{}
	once    sync.Once
	failure error
}

func (p *barrierStop) Output() io.ReadCloser { return io.NopCloser(strings.NewReader("")) }
func (p *barrierStop) Wait() error           { return nil }
func (p *barrierStop) Forced() bool          { return false }
func (p *barrierStop) Stop() error {
	p.once.Do(func() { p.entered <- struct{}{} })
	<-p.release
	return p.failure
}

func TestStopAllStartsAllGroupsBeforeWaitingForGrace(t *testing.T) {
	e := testEngine(t)
	entered, release := make(chan struct{}, 2), make(chan struct{})
	for _, id := range []string{"first", "second"} {
		code := "unix_stop_interrupt_errno_1"
		if id == "second" {
			code = "unix_reap_kill_errno_1"
		}
		child := &barrierStop{entered: entered, release: release, failure: ownershipFailureAt(code)}
		e.active[id] = &activeRun{process: child}
		e.journal.Runs[id] = &RunState{ID: id, State: "running"}
	}
	done := make(chan error, 1)
	go func() { done <- e.StopAll("synthetic deadline") }()
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			close(release)
			<-done
			t.Fatal("a later group waited behind another group's grace period")
		}
	}
	close(release)
	if err := <-done; !errors.Is(err, errOwnership) {
		t.Fatal("ownership failures were lost")
	}
	for id, run := range e.journal.Runs {
		if run.State != "needs_recovery" || run.LocalDiagnostic == "" || !e.active[id].uncertain {
			t.Fatal("one group's uncertainty was discarded")
		}
	}
}
