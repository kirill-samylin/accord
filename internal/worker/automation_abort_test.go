package worker

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"agent-board/internal/core"
)

// This double models a program that catches interruption and exits 0. It closes
// stdout normally, so truncation/read errors cannot accidentally hide the bug.
type gracefulZeroCheck struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	done   chan struct{}
	once   sync.Once
}

func (p *gracefulZeroCheck) Output() io.ReadCloser { return p.reader }
func (p *gracefulZeroCheck) Wait() error           { <-p.done; return nil }
func (p *gracefulZeroCheck) Stop() error {
	p.once.Do(func() { p.writer.Close(); close(p.done) })
	return nil
}
func (p *gracefulZeroCheck) Forced() bool          { return false }
func (p *gracefulZeroCheck) ExitCode() (int, bool) { return 0, true }
func TestAbortedZeroExitWithValidReportCarriesNegativeAuthenticatedFact(t *testing.T) {
	for _, kind := range []string{"deadline", "stop"} {
		t.Run(kind, func(t *testing.T) {
			e, request := automationTestEngine(t)
			started := make(chan struct{})
			post := e.automationTransport
			e.automationTransport = func(ctx context.Context, path string, input, output any) error {
				err := post(ctx, path, input, output)
				in := input.(core.WorkflowAttestInput)
				if in.Kind == "check_started" {
					close(started)
				}
				if in.Kind == "check_finished" {
					output.(*core.WorkflowAttestResult).ControlEpoch = in.ControlEpoch + 1
				}
				return err
			}
			if kind == "deadline" {
				e.active[request.RunID].automation.deadline = time.Now().Add(60 * time.Millisecond)
			}
			e.start = func(launchSpec) (process, error) {
				report := filepath.Join(e.config.Workspaces["test"].Path, ".artifacts/test.tap")
				if err := os.WriteFile(report, []byte("1..1\nok 1 - observed passing test\n"), 0600); err != nil {
					return nil, err
				}
				reader, writer := io.Pipe()
				return &gracefulZeroCheck{reader: reader, writer: writer, done: make(chan struct{})}, nil
			}
			if _, err := e.automationCheckRequest(context.Background(), request.RunID, request); err != nil {
				t.Fatal(err)
			}
			if kind == "stop" {
				select {
				case <-started:
				case <-time.After(time.Second):
					t.Fatal("check not started")
				}
				if err := e.StopAll("explicit stop"); err != nil {
					t.Fatal(err)
				}
			}
			e.wg.Wait()
			e.mu.Lock()
			check := e.journal.Runs[request.RunID].Automation.Checks[request.RequestID]
			o := check.Observations
			code := check.ErrorCode
			e.mu.Unlock()
			if !o.CheckAborted || o.CheckExitCode == nil || *o.CheckExitCode != 0 || o.SourceClean == nil || !*o.SourceClean || o.ChecksPassed != 1 || o.ChecksFailed != 0 || o.OutcomeBytesTruncated || o.ReportKind != "tap" || code != "check_deadline_or_stop" {
				t.Fatalf("abort either lost its negative fact or falsified observed exit/report/source: %+v %s", o, code)
			}
			if err := e.flushAutomation(context.Background()); err != nil {
				t.Fatal("terminal newer epoch rejected its original-bound ACK", err)
			}
		})
	}
}

func TestNewWorkflowEpochNeverAcknowledgesReservationOrDifferentBoundProof(t *testing.T) {
	for _, kind := range []string{"check_reserved", "check_started", "check_finished"} {
		t.Run(kind, func(t *testing.T) {
			e, _ := automationTestEngine(t)
			post := e.automationTransport
			e.automationTransport = func(ctx context.Context, path string, input, output any) error {
				err := post(ctx, path, input, output)
				out := output.(*core.WorkflowAttestResult)
				out.ControlEpoch++
				if kind == "check_finished" {
					out.Attestation.ControlEpoch++
				}
				return err
			}
			input := core.WorkflowAttestInput{Kind: kind, ControlEpoch: 1}
			if _, err := e.sendAttestation(context.Background(), input); err == nil {
				t.Fatal("a newer current epoch admitted launch authority or rebound terminal proof")
			}
		})
	}
}
