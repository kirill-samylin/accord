package store

import (
	"agent-board/internal/core"
	"context"
	"fmt"
	"time"
)

func workflowSteps(b *board, id string) int {
	n := 0
	for _, st := range b.Automation.Steps {
		if st.WorkflowID == id {
			n++
		}
	}
	return n
}
func workflowPrompt(b *board, w *core.Workflow, st *core.WorkflowStep) string {
	text := fmt.Sprintf("Workflow %s, step %s, role %s. Requirement:\n%s\nYour only task: %s. First describe the exact pinned profile and next this task; keep its lease. Complete using transition with genuine evidence; never infer success from CLI exit. Ask workflow_question for human blockers. Do not create recursive workflows or control hosts.\n", w.ID, st.ID, st.Kind, w.IntakeText, st.TaskKey)
	if st.Kind == "analyst" {
		text += "Analyze the checkout and submit workflow_plan with a topologically ordered plan (client_key/title/description/areas/acceptance/prerequisites/required_check_ids), then finish the analyst task with artifact workflow-plan hash and summary. Every task must include the required test presets.\n"
	}
	if st.Kind == "developer" {
		text += "Implement and run ordinary unit/integration tests, commit the actual result and complete the task with its exact full Git SHA and run summary. Do not claim independent review/testing.\n"
	}
	if st.Kind == "reviewer" {
		text += "Read-only independent source review of exact target SHA " + st.TargetSHA + " and current target generation. Complete with approve/changes_requested/blocked and concrete findings. Do not edit source.\n"
	}
	if st.Kind == "tester" {
		text += "Independent testing of reviewed exact SHA " + st.TargetSHA + ". Use workflow_run_check for every required policy check and wait for genuine completed results. Passed needs actual executed tests; NOT-RUN is not PASS. For missing process/E2E tests propose an artifact-only test patch and report test_design; never edit reviewed source directly.\n"
	}
	return text + workflowPromptMessages(b, w, st)
}
func (s *Store) workflowNewTask(b *board, w *core.Workflow, role, title, description string, target *core.Task, now time.Time) (*core.Task, error) {
	binding := w.Grant.RoleBindings[role]
	in := core.CreateInput{Type: "task", EpicKey: w.EpicKey, Role: role, Title: title, Description: description, ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion}
	if target != nil && (role == "reviewer" || role == "tester") {
		v, err := s.checkCreateAsRoleInBoard(b, "coordinator", core.Actor{Principal: w.Requester}, core.CheckCreateInput{TargetKey: target.Key, TargetVersion: target.Version, Role: role, ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion}, now)
		if err != nil {
			return nil, err
		}
		return b.Tasks[v.Key], nil
	}
	v, err := s.createAsRoleInBoard(b, "coordinator", core.Actor{Principal: w.Requester}, in, now)
	if err != nil {
		return nil, err
	}
	t := b.Tasks[v.Key]
	return t, nil
}
func (s *Store) workflowEnqueue(ctx context.Context, b *board, w *core.Workflow, t *core.Task, now time.Time) error {
	if w.Counters.Invocations >= w.Grant.Limits.NativeInvocations || w.ReservedRuns <= 0 {
		return core.Fail("budget_exhausted", "Исчерпан бюджет native invocations")
	}
	if workflowSteps(b, w.ID) >= 256 {
		return core.Fail("capacity", "Предел workflow steps")
	}
	h := b.ManagedHosts[w.HostID]
	if h.Host.WorkerCapabilities["automation"] < 1 {
		return core.Fail("worker_incompatible", "Worker не поддерживает automation:1")
	}
	active := 0
	for _, r := range b.ManagedRuns {
		if managedActive(r.State) {
			active++
			if r.HostID == w.HostID && r.Workspace == w.Workspace {
				return core.Fail("conflict", "Workspace ещё занят")
			}
		}
	}
	if active >= 8 {
		return core.Fail("capacity", "Нет свободного managed slot: уже 8 активных запусков")
	}
	if err := automationRotateHistory(b, now, ""); err != nil {
		return err
	}
	id, err := managedID("ws_")
	if err != nil {
		return err
	}
	runID, err := managedID("mr_")
	if err != nil {
		return err
	}
	kind := t.Role
	profile := w.Grant.Profiles[kind]
	st := &core.WorkflowStep{ID: id, WorkflowID: w.ID, Sequence: int64(workflowSteps(b, w.ID) + 1), Kind: kind, Status: "queued", AttemptNo: 1, TaskKey: t.Key, TaskVersion: t.Version, TaskGeneration: t.Generation, Profile: profile, Provider: w.Grant.RoleBindings[kind].Provider, ManagedRunID: runID, ManagedGeneration: 1, StartedAt: now, DeadlineAt: now.Add(time.Duration(w.Grant.Limits.StepSeconds) * time.Second), ContributorRunIDs: []string{runID}}
	for _, v := range []time.Time{w.DeadlineAt, w.Grant.ExpiresAt} {
		if v.Before(st.DeadlineAt) {
			st.DeadlineAt = v
		}
	}
	if t.Review != nil {
		st.TargetSHA = t.Review.TargetCommit
	}
	profileCopy := profile
	r := &core.ManagedRun{ID: runID, HostID: w.HostID, Version: 1, Provider: st.Provider, Workspace: w.Workspace, Profile: &profileCopy, TaskKey: t.Key, CreatedBy: w.Grant.RequesterID, State: "queued", Generation: 1, CreatedAt: now, UpdatedAt: now, Output: []string{}, Automation: &core.ManagedAutomationBinding{WorkflowID: w.ID, StepID: id, ControlEpoch: w.ControlEpoch, PolicyVersion: w.Grant.PolicyVersion, MessageCount: len(w.Messages), TaskGeneration: t.Generation + 1, DeadlineAt: st.DeadlineAt, TargetSHA: st.TargetSHA, Checks: workflowTaskChecks(b, w, t)}}
	r.Prompt = workflowPrompt(b, w, st)
	b.ManagedRuns[r.ID] = r
	b.Automation.Steps[id] = st
	w.CurrentStepID = id
	w.State = "running"
	// Invocations are counted when the worker reports started
	// (automationRunStarted); a launch that never started spends no budget.
	w.ReservedRuns--
	_, err = managedNewCommand(b, r, core.Actor{Principal: w.Requester}, "start", r.Prompt, now, w.Grant.RequesterEpoch)
	if err == nil {
		workflowTouch(w, now)
	}
	return err
}
func workflowTaskChecks(b *board, w *core.Workflow, t *core.Task) []core.AutomationCheckPresetRef {
	target := t.Key
	if t.Review != nil {
		target = t.Review.TargetKey
	}
	ids := map[string]bool{}
	p := currentPlan(b, w)
	if p != nil {
		for _, item := range p.Tasks {
			if p.TaskKeys[item.ClientKey] == target {
				for _, id := range item.RequiredCheckIDs {
					ids[id] = true
				}
			}
		}
	}
	out := []core.AutomationCheckPresetRef{}
	for _, c := range w.Grant.Checks {
		if ids[c.PresetID] {
			out = append(out, c)
		}
	}
	return out
}
func workflowFindExit(b *board, st *core.WorkflowStep) *core.WorkflowAttestation {
	var found *core.WorkflowAttestation
	for _, a := range b.Automation.Attestations {
		if a.StepID == st.ID && a.RunID == st.ManagedRunID && a.Generation == st.ManagedGeneration && a.Kind == "run_exit" && a.Status == "completed" && (found == nil || a.CreatedAt.After(found.CreatedAt)) {
			found = a
		}
	}
	return found
}
