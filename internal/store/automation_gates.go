package store

import (
	"agent-board/internal/core"
	"context"
	"strings"
	"time"
)

func workflowGitGate(t *core.Task, a *core.WorkflowAttestation) bool {
	o := a.Observations
	if o.SourceClean == nil || !*o.SourceClean || !validCommit(o.ActualHead) || !digestValid(o.ManifestSHA256) || o.OutcomeBytesTruncated {
		return false
	}
	if t.Role == "developer" {
		return o.ActualHead == t.Evidence.Commit
	}
	if t.Review != nil {
		return o.ActualHead == t.Review.TargetCommit && o.TargetSHA == t.Review.TargetCommit && t.Review.TargetGeneration > 0
	}
	return true
}
func workflowTestsGate(b *board, w *core.Workflow, st *core.WorkflowStep, t *core.Task) bool {
	checks := workflowTaskChecks(b, w, t)
	if len(checks) == 0 {
		return false
	}
	hasTest := false
	for _, preset := range checks {
		var result *core.WorkflowAttestation
		for _, a := range b.Automation.Attestations {
			if a.StepID == st.ID && a.RunID == st.ManagedRunID && a.Generation == st.ManagedGeneration && a.TaskGeneration == t.Generation && a.Kind == "check" && a.Status == "completed" && a.PresetID == preset.PresetID && a.PresetVersion == preset.LocalVersion && a.PresetDigest == preset.LocalDigest {
				if result == nil || a.Sequence > result.Sequence || (a.Sequence == result.Sequence && a.ID > result.ID) {
					result = a
				}
			}
		}
		if result == nil {
			return false
		}
		o := result.Observations
		if o.CheckAborted || o.CheckExitCode == nil || *o.CheckExitCode != 0 || o.ActualHead != st.TargetSHA || o.TargetSHA != st.TargetSHA || o.SourceClean == nil || !*o.SourceClean || !digestValid(o.ManifestSHA256) || o.OutcomeBytesTruncated {
			return false
		}
		if preset.Kind == "test" {
			if (o.ReportKind != "junit" && o.ReportKind != "tap") || !digestValid(o.ReportSHA256) || o.ChecksPassed < 1 || o.ChecksFailed != 0 {
				return false
			}
			hasTest = true
		}
	}
	return hasTest
}
func (s *Store) workflowAdvanceAfter(ctx context.Context, b *board, w *core.Workflow, st *core.WorkflowStep, now time.Time) error {
	t := b.Tasks[st.TaskKey]
	switch st.Kind {
	case "analyst":
		p := currentPlan(b, w)
		if p == nil || p.SubmittedSessionID != t.AuthorSession || p.TaskGeneration != t.Generation {
			return core.Fail("evidence_missing", "Нет плана от текущего native аналитика")
		}
		found := false
		for _, ref := range t.Evidence.Artifact {
			if ref == "workflow-plan:"+p.Hash {
				found = true
			}
		}
		if !found {
			return core.Fail("evidence_missing", "Analyst evidence не ссылается на immutable plan hash")
		}
		if err := s.materializePlan(b, w, now); err != nil {
			return err
		}
		return s.workflowNextDeveloper(ctx, b, w, now)
	case "developer":
		review, err := s.workflowNewTask(b, w, "reviewer", "Независимое ревью "+t.Key, "Проверить точный commit и generation результата разработчика.", t, now)
		if err != nil {
			return err
		}
		if err = s.workflowEnqueue(ctx, b, w, review, now); err != nil {
			return err
		}
	case "reviewer":
		if t.Review == nil {
			return core.Fail("evidence_missing", "Нет review outcome")
		}
		switch t.Review.Outcome {
		case "approve":
			target := b.Tasks[t.Review.TargetKey]
			qa, err := s.workflowNewTask(b, w, "tester", "Независимая проверка "+target.Key, "Запустить обязательные policy checks и подтвердить реальный test outcome.", target, now)
			if err != nil {
				return err
			}
			if err = s.workflowEnqueue(ctx, b, w, qa, now); err != nil {
				return err
			}
		case "changes_requested":
			return s.workflowFix(ctx, b, w, t, now)
		default:
			return core.Fail("human_required", "Ревью заблокировано")
		}
	case "tester":
		if t.Review == nil {
			return core.Fail("evidence_missing", "Нет test outcome")
		}
		switch t.Review.Outcome {
		case "passed":
			if !workflowTestsGate(b, w, st, t) {
				return core.Fail("test_evidence_missing", "Нет реально исполненной обязательной проверки")
			}
			return s.workflowNextDeveloper(ctx, b, w, now)
		case "failed":
			return s.workflowFix(ctx, b, w, t, now)
		case "not_run":
			if exit := workflowFindExit(b, st); exit != nil && exit.Observations.TestDesign && len(exit.Observations.ArtifactRefs) > 0 {
				return s.workflowFix(ctx, b, w, t, now)
			}
			return core.Fail("test_not_run", "Тесты не выполнены")
		default:
			return core.Fail("test_not_run", "QA требует реальный passed/failed outcome")
		}
	default:
		return core.Fail("invalid_state", "Неизвестная роль этапа")
	}
	workflowTouch(w, now)
	return nil
}
func (s *Store) workflowFix(ctx context.Context, b *board, w *core.Workflow, defect *core.Task, now time.Time) error {
	target := b.Tasks[defect.Review.TargetKey]
	if target == nil || target.State != "done" || target.Generation != defect.Review.TargetGeneration || target.Evidence.Commit != defect.Review.TargetCommit || defect.State != "done" || !hasEvidence(defect.Evidence) {
		return core.Fail("evidence_stale", "Дефект не относится к текущему результату")
	}
	if w.Counters.FixTotal >= w.Grant.Limits.FixTotal || w.Counters.FixPerTask[target.Key] >= w.Grant.Limits.FixPerTask {
		return core.Fail("budget_exhausted", "Исчерпаны fix iterations")
	}
	if err := reopenWithEvidence(b, "coordinator", target, defect.Key, "Automation fix loop", now, ""); err != nil {
		return err
	}
	w.Counters.FixTotal++
	w.Counters.FixPerTask[target.Key]++
	findings := workflowFindingsMeta(defect)
	workflowAppendMessage(w, findings, workflowFixPrefix+defect.Key+": "+defect.Evidence.Findings+" "+defect.Evidence.RunSummary)
	for _, step := range b.Automation.Steps {
		if step.TaskKey == defect.Key {
			if exit := workflowFindExit(b, step); exit != nil && exit.Observations.TestDesign {
				workflowAppendMessage(w, core.WorkflowMessageMeta{Source: core.WorkflowMessageTestDesign, TaskKey: findings.TaskKey, DefectKey: defect.Key}, workflowTestDesignPrefix+"применить предложенный artifact-only patch, затем новый commit, independent review и execution. Artifact hashes: "+strings.Join(exit.Observations.ArtifactRefs, ", "))
			}
		}
	}
	if err := s.workflowEnqueue(ctx, b, w, target, now); err != nil {
		return err
	}
	workflowTouch(w, now)
	return nil
}
func (s *Store) workflowNextDeveloper(ctx context.Context, b *board, w *core.Workflow, now time.Time) error {
	p := currentPlan(b, w)
	if p == nil || p.Status != "materialized" {
		return core.Fail("evidence_missing", "Нет materialized plan")
	}
	for _, item := range p.Tasks {
		t := b.Tasks[p.TaskKeys[item.ClientKey]]
		if t == nil {
			return core.Fail("evidence_missing", "Задача плана отсутствует")
		}
		verified := false
		for _, st := range b.Automation.Steps {
			if st.WorkflowID != w.ID || st.Kind != "tester" || st.Status != "completed" {
				continue
			}
			check := b.Tasks[st.TaskKey]
			if check != nil && check.Review != nil && check.Review.TargetKey == t.Key && check.Review.TargetGeneration == t.Generation && check.Review.Outcome == "passed" && workflowTestsGate(b, w, st, check) {
				verified = true
			}
		}
		if verified {
			continue
		}
		if t.State != "ready" || t.Lease != nil || blocked(b, t) {
			return core.Fail("dependency_blocked", "Следующая задача не готова")
		}
		if err := s.workflowEnqueue(ctx, b, w, t, now); err != nil {
			return err
		}
		workflowTouch(w, now)
		return nil
	}
	if err := epicClosable(b, w.EpicKey); err != nil {
		return err
	}
	epic := b.Tasks[w.EpicKey]
	epic.State = "done"
	epic.Version++
	if epic.FirstDoneAt == nil {
		epic.FirstDoneAt = &now
	}
	w.State = "completed"
	workflowReason(w, "", "")
	w.ReservedRuns = 0
	delete(b.Automation.Reservations, w.ID)
	revokeWorkflowCapabilities(b, w.ID)
	workflowTouch(w, now)
	emit(b, now, "workflow_completed", w.EpicKey, "")
	return nil
}

// Used by both native transitions and delegated fix loops; this is administration,
// never completion, and retains the existing defect evidence and role gates.
func reopenWithEvidence(b *board, role string, t *core.Task, ref, reason string, now time.Time, session string) error {
	if err := reopenTaskWithEvidence(b, role, t, ref, reason, now); err != nil {
		return err
	}
	t.Version++
	reopenEpic(b, t.EpicKey, now, session)
	emit(b, now, "task_ready", t.Key, session)
	return nil
}

func reopenTaskWithEvidence(b *board, role string, t *core.Task, ref, reason string, now time.Time) error {
	if !reopenAllowed(role, t.Role) {
		return core.Fail("forbidden", "Роль не может переоткрыть задачу")
	}
	evidence := b.Tasks[ref]
	if !bounded(reason, 4096) || evidence == nil || evidence.State != "done" || !hasEvidence(evidence.Evidence) {
		return core.Fail("evidence_missing", "Нужны причина и существующее evidence")
	}
	t.ReturnsCount++
	t.State = "ready"
	t.ReadyAt = now
	return nil
}
