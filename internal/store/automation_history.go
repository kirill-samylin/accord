package store

import (
	"agent-board/internal/core"
	"fmt"
	"sort"
	"time"
)

// Admission caps count only live work: non-terminal workflows, active managed
// runs and the run budget other workflows have reserved. History is bounded
// separately: every transaction loads (and automation ticks clone) the whole
// live state, and a terminal run alone may hold 64KiB output plus a 16KiB
// prompt. When a NEW workflow or run needs admission and terminal history in
// the live state reaches its bound, the oldest terminal entries are moved, in
// the same transaction, to archive rows of the same tables (key prefix
// "archive/"). Nothing is deleted; archive rows are excluded from live loads
// and are paged by AutomationArchive. Each rotation emits an audit event.
const (
	workflowLiveLimit        = 32
	managedRunLiveLimit      = 128
	workflowTerminalBound    = 128
	workflowTerminalTarget   = 96
	managedTerminalBound     = 512
	managedTerminalTarget    = 448
	automationArchivePrefix  = "archive/"
	automationArchiveAuditEv = "automation_history_archived"
)

// archiveMove renames one live row to its archive key when the board is saved
// and stores the entity's final in-memory value with it. It is unexported so a
// discarded candidate board drops it with the rest of its changes.
type archiveMove struct {
	table, key string
	value      any
}

func managedLiveRuns(b *board) int {
	n := 0
	for _, r := range b.ManagedRuns {
		if managedActive(r.State) {
			n++
		}
	}
	return n
}
func workflowLive(b *board) int {
	n := 0
	for _, w := range b.Automation.Workflows {
		if !workflowTerminal(w.State) {
			n++
		}
	}
	return n
}
func automationTerminalHistory(b *board) (runs, flows int) {
	for _, r := range b.ManagedRuns {
		if !managedActive(r.State) {
			runs++
		}
	}
	if b.Automation != nil {
		flows = len(b.Automation.Workflows) - workflowLive(b)
	}
	return runs, flows
}

// automationRotateHistory makes room for one new workflow or run. It refuses
// with capacity only when the bound is reached and no terminal entry is
// archivable (every terminal run belongs to a live workflow or still holds a
// command/session/lease).
func automationRotateHistory(b *board, now time.Time, session string) error {
	runs, flows := automationTerminalHistory(b)
	if runs < managedTerminalBound && flows < workflowTerminalBound {
		return nil
	}
	archivedFlows, archivedRuns := 0, 0
	if b.Automation != nil && flows >= workflowTerminalBound {
		for _, w := range archivableWorkflows(b, now) {
			if flows-archivedFlows <= workflowTerminalTarget {
				break
			}
			archivedRuns += archiveWorkflow(b, w)
			archivedFlows++
		}
	}
	if runs-archivedRuns >= managedTerminalBound {
		for _, r := range archivableManualRuns(b, now) {
			if runs-archivedRuns <= managedTerminalTarget {
				break
			}
			archiveRun(b, r)
			archivedRuns++
		}
	}
	if b.Automation != nil && runs-archivedRuns > managedTerminalTarget {
		for _, w := range archivableWorkflows(b, now) {
			if runs-archivedRuns <= managedTerminalTarget {
				break
			}
			archivedRuns += archiveWorkflow(b, w)
			archivedFlows++
		}
	}
	if archivedFlows+archivedRuns > 0 {
		emit(b, now, automationArchiveAuditEv, "", session)
		b.Events[len(b.Events)-1].Reason = fmt.Sprintf("archived_workflows=%d archived_runs=%d", archivedFlows, archivedRuns)
	}
	if runs-archivedRuns >= managedTerminalBound || flows-archivedFlows >= workflowTerminalBound {
		return core.Fail("capacity", fmt.Sprintf("История автоматизации заполнена (%d завершённых запусков, %d циклов) и архивная ротация невозможна: старые записи ещё связаны с активными процессами", runs-archivedRuns, flows-archivedFlows))
	}
	return nil
}
func archivableWorkflows(b *board, now time.Time) []*core.Workflow {
	out := []*core.Workflow{}
	for _, w := range b.Automation.Workflows {
		if workflowTerminal(w.State) && b.Automation.Reservations[w.ID] == nil && !workflowOwnedActive(b, w) && workflowTasksIdle(b, w) && workflowRunsIdle(b, w, now) {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.Before(out[j].UpdatedAt)
	})
	return out
}

// Tasks of an archived workflow become ordinary terminal board history; the
// workflow can never run again, so only idle tasks may lose the workflow guard.
func workflowTasksIdle(b *board, w *core.Workflow) bool {
	for _, t := range b.Tasks {
		if w.EpicKey != "" && (t.EpicKey == w.EpicKey || t.Key == w.EpicKey) && (t.Lease != nil || t.State == "in_progress") {
			return false
		}
	}
	for _, st := range b.Automation.Steps {
		if st.WorkflowID == w.ID {
			if t := b.Tasks[st.TaskKey]; t != nil && (t.Lease != nil || t.State == "in_progress") {
				return false
			}
		}
	}
	return true
}
func workflowRunsIdle(b *board, w *core.Workflow, now time.Time) bool {
	for _, r := range b.ManagedRuns {
		if r.Automation != nil && r.Automation.WorkflowID == w.ID && !managedRunIdle(b, r, now) {
			return false
		}
	}
	return true
}

// managedRunIdle: terminal, no undelivered/accepted command and no live session.
func managedRunIdle(b *board, r *core.ManagedRun, now time.Time) bool {
	if managedActive(r.State) {
		return false
	}
	for _, c := range b.ManagedCommands {
		if c.Command.RunID == r.ID && (c.Command.Status == "queued" || c.Command.Status == "accepted") {
			return false
		}
	}
	for _, sess := range b.Sessions {
		if sess.ManagedRunID == r.ID && sess.ExpiresAt.After(now) {
			return false
		}
	}
	return true
}
func archivableManualRuns(b *board, now time.Time) []*core.ManagedRun {
	out := []*core.ManagedRun{}
	for _, r := range b.ManagedRuns {
		if r.Automation == nil && managedRunIdle(b, r, now) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.Before(out[j].UpdatedAt)
	})
	return out
}
func archiveRun(b *board, r *core.ManagedRun) {
	for id, c := range b.ManagedCommands {
		if c.Command.RunID == r.ID {
			b.archive = append(b.archive, archiveMove{"managed_commands", id, c})
			delete(b.ManagedCommands, id)
		}
	}
	b.archive = append(b.archive, archiveMove{"managed_runs", r.ID, r})
	delete(b.ManagedRuns, r.ID)
}

// archiveWorkflow moves the workflow with its steps, plans, questions,
// attestations, capabilities and (terminal) runs; it returns the run count.
func archiveWorkflow(b *board, w *core.Workflow) int {
	a := b.Automation
	for id, v := range a.Steps {
		if v.WorkflowID == w.ID {
			b.archive = append(b.archive, archiveMove{"workflow_steps", id, v})
			delete(a.Steps, id)
		}
	}
	for id, v := range a.Plans {
		if v.WorkflowID == w.ID {
			b.archive = append(b.archive, archiveMove{"workflow_plans", id, v})
			delete(a.Plans, id)
		}
	}
	for id, v := range a.Questions {
		if v.WorkflowID == w.ID {
			b.archive = append(b.archive, archiveMove{"workflow_questions", id, v})
			delete(a.Questions, id)
		}
	}
	for id, v := range a.Attestations {
		if v.WorkflowID == w.ID {
			b.archive = append(b.archive, archiveMove{"workflow_attestations", id, v})
			delete(a.Attestations, id)
		}
	}
	for id, v := range a.Capabilities {
		if v.WorkflowID == w.ID {
			// An archived workflow never runs again: its execution authority
			// is revoked in the archive row itself, which also satisfies the
			// restore verifier (every capability row revoked after reset).
			v.State = "revoked"
			b.archive = append(b.archive, archiveMove{"execution_capabilities", id, v})
			delete(a.Capabilities, id)
		}
	}
	runs := 0
	for _, r := range b.ManagedRuns {
		if r.Automation != nil && r.Automation.WorkflowID == w.ID {
			archiveRun(b, r)
			runs++
		}
	}
	b.archive = append(b.archive, archiveMove{"workflows", w.ID, w})
	delete(a.Workflows, w.ID)
	return runs
}
