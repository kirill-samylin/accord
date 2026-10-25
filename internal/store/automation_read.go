package store

import (
	"agent-board/internal/core"
	"context"
	"sort"
	"time"
)

func workflowPage(ctx context.Context, b *board, a core.Actor, w *core.Workflow, now time.Time) (core.WorkflowPage, error) {
	return workflowPageWindow(workflowFullPage(ctx, b, a, w, now), core.WorkflowReadInput{}, automationCursorNamespace(b))
}
func workflowFullPage(ctx context.Context, b *board, a core.Actor, w *core.Workflow, now time.Time) core.WorkflowPage {
	out := core.WorkflowPage{Workflow: *w, Steps: []core.WorkflowStep{}, Plans: []core.WorkflowPlanSnapshot{}, Questions: []core.WorkflowQuestionRecord{}, Attestations: []core.WorkflowAttestation{}, CanControl: workflowController(ctx, b, a, w, now) == nil}
	// Readers always see one label per message, including inferred legacy rows.
	out.Workflow.MessageMeta = workflowMessageMetas(b, w)
	for _, v := range b.Automation.Steps {
		if v.WorkflowID == w.ID {
			out.Steps = append(out.Steps, *v)
		}
	}
	sort.Slice(out.Steps, func(i, j int) bool { return out.Steps[i].Sequence < out.Steps[j].Sequence })
	for _, v := range b.Automation.Plans {
		if v.WorkflowID == w.ID {
			out.Plans = append(out.Plans, *v)
		}
	}
	sort.Slice(out.Plans, func(i, j int) bool { return out.Plans[i].Revision < out.Plans[j].Revision })
	for _, v := range b.Automation.Questions {
		if v.WorkflowID == w.ID {
			out.Questions = append(out.Questions, *v)
		}
	}
	sort.Slice(out.Questions, func(i, j int) bool {
		return numericKey(out.Questions[i].HumanTaskKey) < numericKey(out.Questions[j].HumanTaskKey)
	})
	for _, v := range b.Automation.Attestations {
		if v.WorkflowID == w.ID {
			out.Attestations = append(out.Attestations, *v)
		}
	}
	sort.Slice(out.Attestations, func(i, j int) bool {
		a, z := out.Attestations[i], out.Attestations[j]
		if a.Sequence == z.Sequence {
			return a.ID < z.ID
		}
		return a.Sequence < z.Sequence
	})
	return out
}
func (s *Store) WorkflowGet(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.WorkflowPage, error) {
		if _, _, err := automationActor(ctx, b, a, now, false); err != nil {
			return core.WorkflowPage{}, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return core.WorkflowPage{}, err
		}
		w, err := workflowFor(b, in.WorkflowID)
		if err != nil {
			return core.WorkflowPage{}, err
		}
		if ident, ok := ExecutionIdentityFromContext(ctx); ok && ident.Capability.WorkflowID != w.ID {
			return core.WorkflowPage{}, core.Fail("forbidden", "Дочерняя сессия ограничена своим workflow")
		}
		return workflowPageWindow(workflowFullPage(ctx, b, a, w, now), in, automationCursorNamespace(b))
	})
}
func (s *Store) WorkflowReport(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowPage, error) {
	return s.WorkflowGet(ctx, a, in)
}
