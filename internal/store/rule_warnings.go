package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"time"
)

func ruleWarning(b *board, t *core.Task, rule, message, session string) {
	w := core.RuleWarning{Rule: rule, Message: message, TaskKey: t.Key, SessionID: session}
	t.Warnings = append(t.Warnings, w)
}
func emitWarnings(b *board, t *core.Task, now time.Time, session string) {
	for _, w := range t.Warnings {
		emit(b, now, "rule_warning", t.Key, session)
		raw, _ := json.Marshal(w)
		b.Events[len(b.Events)-1].Reason = string(raw)
		copy := w
		b.Events[len(b.Events)-1].Warning = &copy
	}
}
func ruleHints(b *board) map[string]string {
	r := b.Config.Workflow.Rules
	return map[string]string{"evidence_required": r.EvidenceRequired.Mode, "independent_review": r.IndependentReview.Mode, "one_task_per_agent": r.OneTaskPerAgent.Mode, "cycle_check": r.CycleCheck.Mode, "singleton_role": r.SingletonRole.Mode, "epic_owner": r.EpicOwner.Mode, "area_overlap": r.AreaOverlap.Mode, "duplicate_check": r.DuplicateCheck.Mode}
}
