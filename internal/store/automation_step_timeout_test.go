package store

import (
	"agent-board/internal/core"
	"reflect"
	"testing"
	"time"
)

// A step that ran keeps its pinned deadline across human waiting. A step that
// was never dispatched is covered by TestAutomationNeverDispatchedExpiredStepGetsFreshStepOnResume.
func TestAutomationExpiredInterruptedStepBlocksWithoutRetryChurn(t *testing.T) {
	for _, started := range []bool{true} {
		name := "human_question"
		t.Run(name, func(t *testing.T) {
			m := automationSetup(t)
			id := m.submit(t).Workflow.ID
			page := m.tick(t, id)
			if started {
				a := m.attempt(t, id)
				page = m.page(t, id)
				asked, err := m.f.s.WorkflowQuestion(a.ctx, a.actor, core.WorkflowQuestionInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Question: "Confirm acceptance before proceeding."})
				if err != nil {
					t.Fatal(err)
				}
				timeoutStop(t, m, a)
				m.f.clock.advance(time.Duration(m.p.Limits.StepSeconds) * time.Second)
				page = m.page(t, id)
				q := asked.Questions[0]
				page, err = m.f.s.WorkflowAnswer(m.ctx, m.a, core.WorkflowAnswerInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, QuestionID: q.ID, QuestionVersion: q.Version, Answer: "Confirmed.", Resume: true})
				if err != nil || page.Questions[0].State != "answered" {
					t.Fatal("human answer was not durable", err)
				}
			} else {
				var err error
				page, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"})
				if err != nil {
					t.Fatal(err)
				}
				m.f.clock.advance(time.Duration(m.p.Limits.StepSeconds) * time.Second)
				page, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "resume"})
				if err != nil {
					t.Fatal(err)
				}
			}
			before := timeoutSnapshot(t, m)
			blocked := m.tick(t, id)
			if blocked.Workflow.State != "blocked" || blocked.Workflow.ReasonCode != "STEP_TIMEOUT" || blocked.Workflow.Version != page.Workflow.Version+1 {
				t.Fatal("expired step not explicitly blocked", blocked.Workflow)
			}
			if !reflect.DeepEqual(page.Steps, blocked.Steps) || !reflect.DeepEqual(page.Workflow.Counters, blocked.Workflow.Counters) || !page.Workflow.DeadlineAt.Equal(blocked.Workflow.DeadlineAt) || page.Workflow.ReservedRuns != blocked.Workflow.ReservedRuns || !reflect.DeepEqual(page.Workflow.Grant, blocked.Workflow.Grant) {
				t.Fatal("timeout changed step, budget or grant")
			}
			after := timeoutSnapshot(t, m)
			if !reflect.DeepEqual(before.ManagedRuns, after.ManagedRuns) || !reflect.DeepEqual(before.ManagedCommands, after.ManagedCommands) || !reflect.DeepEqual(before.Tasks, after.Tasks) {
				t.Fatal("timeout dispatched or rewrote business work")
			}
			for i := 0; i < 3; i++ {
				m.f.clock.advance(2 * time.Second)
				v, err := m.f.s.AutomationTick(m.ctx, core.WorkflowAdvanceInput{WorkflowID: id, ExpectedVersion: blocked.Workflow.Version, ControlEpoch: blocked.Workflow.ControlEpoch})
				if err != nil || v.Changed || v.Workflow.Version != blocked.Workflow.Version || !reflect.DeepEqual(m.page(t, id).Workflow, blocked.Workflow) {
					t.Fatal("blocked tick churned", err, v.Changed)
				}
			}
			// A fresh human command now retains its expected version. Replan is
			// explicit and retains both the workflow deadline and invocation ceiling.
			paused, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: blocked.Workflow.Version, Kind: "pause"})
			if err != nil || paused.Workflow.State != "paused" {
				t.Fatal("stable human pause failed", err)
			}
			replanned, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: paused.Workflow.Version, Kind: "replan", Text: "Retain prior history; analyze the clarified acceptance."})
			if err != nil || replanned.Workflow.CurrentStepID != "" {
				t.Fatal("explicit replan failed", err)
			}
			next := m.tick(t, id)
			if next.Workflow.State != "running" || len(next.Steps) != 2 || next.Steps[1].Kind != "analyst" || !next.Workflow.DeadlineAt.Equal(blocked.Workflow.DeadlineAt) || next.Workflow.Grant.Limits.NativeInvocations != blocked.Workflow.Grant.Limits.NativeInvocations || next.Workflow.Counters.Invocations != blocked.Workflow.Counters.Invocations {
				t.Fatal("replan did not preserve overall bounds", next.Workflow)
			}
		})
	}
}

func TestAutomationNeverDispatchedExpiredStepGetsFreshStepOnResume(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	page := m.tick(t, id)
	page, err := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"})
	if err != nil || page.Workflow.State != "paused" {
		t.Fatal("pause", err)
	}
	m.f.clock.advance(time.Duration(m.p.Limits.StepSeconds) * time.Second)
	page, err = m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	fresh := m.tick(t, id)
	if fresh.Workflow.State != "running" || len(fresh.Steps) != 2 || fresh.Steps[0].Status != "expired" || fresh.Steps[1].Kind != "analyst" || fresh.Steps[1].Status != "queued" || fresh.Steps[1].TaskKey != fresh.Steps[0].TaskKey || !fresh.Steps[1].DeadlineAt.After(fresh.Steps[0].DeadlineAt) {
		t.Fatal("never-dispatched expired step did not get a fresh step", fresh.Workflow, fresh.Steps)
	}
	if fresh.Workflow.Counters.Invocations != 0 || !fresh.Workflow.DeadlineAt.Equal(page.Workflow.DeadlineAt) {
		t.Fatal("fresh step spent budget or renewed the workflow deadline", fresh.Workflow.Counters)
	}
	a := m.attempt(t, id)
	if a.poll.Run.ID != fresh.Steps[1].ManagedRunID || m.page(t, id).Workflow.Counters.Invocations != 1 {
		t.Fatal("fresh step is not the dispatched one or start was not counted")
	}
	// Once the workflow deadline itself is gone the step stays a timeout.
	m2 := automationSetup(t)
	id2 := m2.submit(t).Workflow.ID
	p2 := m2.tick(t, id2)
	p2, _ = m2.f.s.WorkflowCommand(m2.ctx, m2.a, core.WorkflowCommandInput{RequestID: m2.f.id(), WorkflowID: id2, Version: p2.Workflow.Version, Kind: "pause"})
	m2.f.clock.advance(time.Duration(m2.p.Limits.WorkflowSeconds) * time.Second)
	if _, err = m2.f.s.WorkflowCommand(m2.ctx, m2.a, core.WorkflowCommandInput{RequestID: m2.f.id(), WorkflowID: id2, Version: p2.Workflow.Version, Kind: "resume"}); err == nil {
		if after := m2.tick(t, id2); after.Workflow.State == "running" {
			t.Fatal("expired workflow deadline produced a fresh step", after.Workflow)
		}
	}
}

func TestAutomationCompletedUnfinishedTurnCannotContinueAtDeadline(t *testing.T) {
	m := automationSetup(t)
	id := m.submit(t).Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	m.exit(t, a, commitA, "")
	before := m.page(t, id)
	m.f.clock.advance(time.Duration(m.p.Limits.StepSeconds) * time.Second)
	after := m.tick(t, id)
	if after.Workflow.State != "blocked" || after.Workflow.ReasonCode != "STEP_TIMEOUT" || after.Workflow.Counters.Invocations != before.Workflow.Counters.Invocations || after.Steps[0].ManagedGeneration != before.Steps[0].ManagedGeneration || !after.Steps[0].DeadlineAt.Equal(before.Steps[0].DeadlineAt) {
		t.Fatal("expired native continuation queued", after.Workflow)
	}
}

func timeoutSnapshot(t *testing.T, m *automationFixture) *board {
	t.Helper()
	b, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (*board, error) { return automationClone(b) })
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func timeoutStop(t *testing.T, m *automationFixture, a automationAttempt) {
	t.Helper()
	stop, err := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch, WorkerCapabilities: map[string]int{"automation": 1}})
	if err != nil || stop.Command == nil || stop.Command.Kind != "stop" {
		t.Fatal("stop not dispatched", err)
	}
	report := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: stop.WorkerEpoch}, CommandID: stop.Command.ID, DispatchID: stop.Command.DispatchID, RunID: stop.Run.ID, Generation: stop.Run.Generation, Phase: "accepted"}
	if _, err = m.f.s.ManagedReport(m.ctx, m.a, report); err != nil {
		t.Fatal(err)
	}
	report.RequestID, report.Phase = m.f.id(), "stopped"
	if _, err = m.f.s.ManagedReport(m.ctx, m.a, report); err != nil {
		t.Fatal(err)
	}
}
