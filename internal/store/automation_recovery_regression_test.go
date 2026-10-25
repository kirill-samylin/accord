package store

import (
	"agent-board/internal/core"
	"strings"
	"testing"
	"time"
)

func (m *automationFixture) report(a automationAttempt, worker, phase, text string) core.ManagedReportInput {
	return core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: worker, WorkerEpoch: a.poll.WorkerEpoch}, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, Phase: phase, Error: text}
}

// T022-R2: cancel requested while a run is needs_recovery is remembered, resume
// and other dispatch stay refused while ownership is uncertain, and cancel
// completes (reservation released) after the worker reports recovered_stopped,
// from the live worker or from a restarted incarnation after lease expiry.
func TestAutomationCancelDuringNeedsRecoveryCompletesAfterRecoveredStopped(t *testing.T) {
	for _, restarted := range []bool{false, true} {
		name := "live_worker"
		if restarted {
			name = "restarted_worker"
		}
		t.Run(name, func(t *testing.T) {
			m := automationSetup(t)
			id := m.submit(t).Workflow.ID
			m.tick(t, id)
			a := m.attempt(t, id)
			if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, "fixture-worker", "delivery_unknown", "worker restarted after a possible launch")); e != nil {
				t.Fatal(e)
			}
			page := m.tick(t, id)
			if page.Workflow.State != "needs_recovery" || !strings.Contains(page.Workflow.ReasonMessage, "possible launch") {
				t.Fatal("uncertain run did not explain recovery", page.Workflow.State, page.Workflow.ReasonMessage)
			}
			cancel, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "cancel"})
			if err != nil || !cancel.Workflow.CancelRequested || cancel.Workflow.State == "cancelled" {
				t.Fatal("cancel of uncertain run", err, cancel.Workflow.State)
			}
			page = m.tick(t, id)
			if page.Workflow.State != "needs_recovery" || !page.Workflow.CancelRequested {
				t.Fatal("cancel intent lost while ownership uncertain", page.Workflow.State)
			}
			if _, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "resume"}); err == nil {
				t.Fatal("resume allowed while ownership uncertain")
			}
			second := m.submit(t)
			queued := m.tick(t, second.Workflow.ID)
			if queued.Workflow.State != "queued" {
				t.Fatal("uncertain workspace admitted another workflow", queued.Workflow.State)
			}
			if again := m.tick(t, id); again.Workflow.Version != page.Workflow.Version {
				t.Fatal("uncertain cancel scan churned version")
			}
			worker := "fixture-worker"
			if restarted {
				worker = "worker_restarted"
				early := m.report(a, worker, "recovered_stopped", "")
				if _, e := m.f.s.ManagedReport(m.ctx, m.a, early); e == nil {
					t.Fatal("restarted worker took over a live lease")
				}
				m.f.clock.advance(31 * time.Second)
			}
			if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, worker, "recovered_stopped", "")); e != nil {
				t.Fatal("recovered_stopped for needs_recovery run rejected", e)
			}
			done := m.tick(t, id)
			if done.Workflow.State != "cancelled" || done.Workflow.ReservedRuns != 0 || done.Workflow.CancelRequested {
				t.Fatal("cancel did not complete after proven local stop", done.Workflow.State)
			}
			if next := m.tick(t, second.Workflow.ID); next.Workflow.State != "running" {
				t.Fatal("reservation not released after cancel", next.Workflow.State, next.Workflow.ReasonCode)
			}
		})
	}
}

// T022-R2 (pause variant): after recovery a paused-for-recovery workflow is
// paused, not stuck, and resumes only explicitly.
func TestAutomationNeedsRecoveryBecomesPausedAfterRecoveredStopped(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, "fixture-worker", "delivery_unknown", "lost")); e != nil {
		t.Fatal(e)
	}
	m.tick(t, id)
	if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, "fixture-worker", "recovered_stopped", "")); e != nil {
		t.Fatal(e)
	}
	page := m.tick(t, id)
	if page.Workflow.State != "paused" || page.Workflow.ReasonCode != "native_recovered" {
		t.Fatal("recovered workflow not paused", page.Workflow.State, page.Workflow.ReasonCode)
	}
	if _, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "resume"}); err != nil {
		t.Fatal("explicit resume after recovery", err)
	}
}

// T022-R3: the worker's own deadline watchdog stops the native turn and reports
// recovered_stopped as the live lease holder; its exit proof must be accepted.
func TestAutomationLocalDeadlineRecoveredStoppedKeepsLiveWorkerLeaseForExitProof(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	lease := func() time.Time {
		page, e := m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{HostID: m.h.ID})
		if e != nil || len(page.Hosts) != 1 {
			t.Fatal("list", e)
		}
		return page.Hosts[0].LeaseExpiresAt
	}
	before := lease()
	m.f.clock.advance(10 * time.Second)
	if _, e := m.f.s.ManagedReport(m.ctx, m.a, m.report(a, "fixture-worker", "recovered_stopped", "local deadline")); e != nil {
		t.Fatal(e)
	}
	// T022-R2-9: a local stop keeps, but never extends, the live lease.
	if after := lease(); !after.Equal(before) {
		t.Fatal("recovered_stopped extended the lease", before, after)
	}
	if _, err := m.f.s.WorkflowAttest(m.ctx, m.a, m.exitInput(a)); err != nil {
		t.Fatal("run_exit after same-worker recovered_stopped rejected", err)
	}
	poll, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch, WorkerCapabilities: map[string]int{"automation": 1}})
	if e != nil || poll.WorkerEpoch != a.poll.WorkerEpoch {
		t.Fatal("live worker lost its epoch", e, poll.WorkerEpoch, a.poll.WorkerEpoch)
	}
}

// T022-R5: a human pause during startup only adds a stop; the dispatched start
// keeps its status and the worker's own failed report for it is accepted.
func TestAutomationPauseDuringStartupKeepsDispatchedStartAndAcceptsItsFailedReport(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	m.tick(t, id)
	pollIn := core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerCapabilities: map[string]int{"automation": 1}}
	poll, e := m.f.s.ManagedPoll(m.ctx, m.a, pollIn)
	if e != nil {
		t.Fatal(e)
	}
	start := automationAttempt{poll: poll}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(start, "fixture-worker", "accepted", "")); e != nil {
		t.Fatal(e)
	}
	page := m.page(t, id)
	if _, e = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"}); e != nil {
		t.Fatal(e)
	}
	pollIn.RequestID, pollIn.WorkerEpoch = m.f.id(), poll.WorkerEpoch
	stop, e := m.f.s.ManagedPoll(m.ctx, m.a, pollIn)
	if e != nil || stop.Command == nil || stop.Command.Kind != "stop" {
		t.Fatal("stop", e)
	}
	stopAttempt := automationAttempt{poll: stop}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(stopAttempt, "fixture-worker", "accepted", "")); e != nil {
		t.Fatal(e)
	}
	pollIn.RequestID = m.f.id()
	if _, e = m.f.s.ManagedPoll(m.ctx, m.a, pollIn); e != nil {
		t.Fatal(e)
	}
	run, _ := m.f.s.ManagedRun(m.ctx, m.a, core.ManagedRunInput{RunID: poll.Run.ID})
	for _, c := range run.Commands {
		if c.Kind == "start" && c.Status != "accepted" {
			t.Fatal("dispatched start rewritten after routine pause", c.Status)
		}
	}
	if strings.Contains(run.Run.LastError, "потерял доступ") {
		t.Fatal("pause reported as lost initiator access", run.Run.LastError)
	}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(start, "fixture-worker", "failed", "stopped before native turn was observed")); e != nil {
		t.Fatal("worker's own failed(start) report rejected", e)
	}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(stopAttempt, "fixture-worker", "stopped", "")); e != nil {
		t.Fatal("owned stop proof rejected", e)
	}
	if after := m.tick(t, id); after.Workflow.State != "paused" {
		t.Fatal("pause did not settle", after.Workflow.State)
	}
}

// T022-R1: a tester task finished before a pause is completed on resume with the
// same fields as normal observation, so the workflow can finish.
func TestAutomationTesterDoneBeforeInterruptionCompletesOnResume(t *testing.T) {
	m := automationSetup(t)
	id := automationToReview(t, m)
	reviewer := m.attempt(t, id)
	m.finish(t, reviewer, core.Evidence{VerifiedCommit: commitB, Findings: "Reviewed exact source and tests independently"}, "approve")
	m.exit(t, reviewer, commitB, commitB)
	m.tick(t, id)
	tester := m.attempt(t, id)
	m.attest(t, tester, "check_reserved", "check-one", core.WorkflowObservations{})
	m.attest(t, tester, "check_started", "check-one", core.WorkflowObservations{})
	clean, exit := true, 0
	m.attest(t, tester, "check_finished", "check-one", core.WorkflowObservations{ActualHead: commitB, TargetSHA: commitB, SourceClean: &clean, ManifestSHA256: strings.Repeat("b", 64), CheckExitCode: &exit, ChecksPassed: 7, ReportKind: "junit", ReportSHA256: strings.Repeat("c", 64)})
	m.finish(t, tester, core.Evidence{VerifiedCommit: commitB, RunSummary: "7 tests actually passed"}, "passed")
	m.exit(t, tester, commitB, commitB)
	page := m.page(t, id)
	paused, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"})
	if err != nil || paused.Workflow.State != "paused" {
		t.Fatal("pause", err)
	}
	if _, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: paused.Workflow.Version, Kind: "resume"}); err != nil {
		t.Fatal(err)
	}
	after := m.tick(t, id)
	st := after.Steps[3]
	if after.Workflow.State != "completed" || st.Status != "completed" || st.EndedAt == nil || st.NativeSessionID != tester.uuid || len(st.EvidenceRefs) == 0 {
		t.Fatalf("verified tester result unrecoverable: %s/%s step=%+v", after.Workflow.State, after.Workflow.ReasonCode, st)
	}
}

// T022-R2-4: a worker's late receipt or start of a command dispatched before a
// routine pause is recorded (native session, invocation budget) instead of
// being refused; the run stays stopping, the owned stop stays pending and the
// step is not resumed. Another worker still has no authority to report it.
func TestAutomationLateReceiptAfterPauseIsRecordedWithoutResumingWork(t *testing.T) {
	for _, acceptedBeforePause := range []bool{false, true} {
		name := "accepted_after_pause"
		if acceptedBeforePause {
			name = "started_after_pause"
		}
		t.Run(name, func(t *testing.T) {
			m := automationSetup(t)
			id := m.submit(t).Workflow.ID
			m.tick(t, id)
			pollIn := core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerCapabilities: map[string]int{"automation": 1}}
			poll, e := m.f.s.ManagedPoll(m.ctx, m.a, pollIn)
			if e != nil || poll.Command == nil || poll.Command.Kind != "start" {
				t.Fatal("start", e)
			}
			start := automationAttempt{poll: poll}
			if acceptedBeforePause {
				if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(start, "fixture-worker", "accepted", "")); e != nil {
					t.Fatal(e)
				}
			}
			page := m.page(t, id)
			if _, e = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"}); e != nil {
				t.Fatal(e)
			}
			pollIn.RequestID, pollIn.WorkerEpoch = m.f.id(), poll.WorkerEpoch
			stop, e := m.f.s.ManagedPoll(m.ctx, m.a, pollIn)
			if e != nil || stop.Command == nil || stop.Command.Kind != "stop" {
				t.Fatal("stop", e)
			}
			invocations := m.page(t, id).Workflow.Counters.Invocations
			if !acceptedBeforePause {
				if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(start, "fixture-worker", "accepted", "")); e != nil {
					t.Fatal("late accepted refused", e)
				}
			}
			intruder := m.report(start, "other-worker", "started", "")
			intruder.ProviderSessionID = "00000001-1234-1234-1234-000000000001"
			if _, e = m.f.s.ManagedReport(m.ctx, m.a, intruder); e == nil {
				t.Fatal("non-owner late start accepted")
			}
			late := m.report(start, "fixture-worker", "started", "")
			late.ProviderSessionID = "00000002-1234-1234-1234-000000000002"
			got, e := m.f.s.ManagedReport(m.ctx, m.a, late)
			if e != nil {
				t.Fatal("late started refused", e)
			}
			if got.Run.State != "stopping" || got.Run.ProviderSessionID != late.ProviderSessionID || got.Command.Status != "applied" {
				t.Fatal("late start resumed work or lost its session", got.Run.State, got.Run.ProviderSessionID, got.Command.Status)
			}
			after := m.page(t, id)
			if after.Workflow.Counters.Invocations != invocations+1 {
				t.Fatal("late native start not counted", invocations, after.Workflow.Counters.Invocations)
			}
			for _, st := range after.Steps {
				if st.Status == "running" {
					t.Fatal("paused step resumed by late start", st.ID)
				}
			}
			detail, e := m.f.s.ManagedRun(m.ctx, m.a, core.ManagedRunInput{RunID: poll.Run.ID})
			if e != nil {
				t.Fatal(e)
			}
			for _, c := range detail.Commands {
				if c.Kind == "stop" && c.ID == stop.Command.ID && c.Status != "queued" {
					t.Fatal("owned stop no longer pending", c.Status)
				}
			}
			pollIn.RequestID = m.f.id()
			again, e := m.f.s.ManagedPoll(m.ctx, m.a, pollIn)
			if e != nil || again.Command == nil || again.Command.ID != stop.Command.ID {
				t.Fatal("pending stop not re-delivered", e)
			}
			stopAttempt := automationAttempt{poll: stop}
			if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(stopAttempt, "fixture-worker", "accepted", "")); e != nil {
				t.Fatal(e)
			}
			if _, e = m.f.s.ManagedReport(m.ctx, m.a, m.report(stopAttempt, "fixture-worker", "stopped", "")); e != nil {
				t.Fatal("owned stop proof rejected", e)
			}
			if settled := m.tick(t, id); settled.Workflow.State != "paused" {
				t.Fatal("pause did not settle", settled.Workflow.State)
			}
		})
	}
}
