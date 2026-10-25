package worker

import (
	"fmt"
	"sort"
	"time"

	"agent-board/internal/core"
)

// Local workflow budget: an owner-consented ceiling the server cannot raise.
//   - Invocations counts only native processes that actually started; a failed
//     preparation consumes nothing.
//   - The window starts at the first preparation and lasts MaxWorkflowSeconds.
//     When a human explicitly refreshes the workflow delegation the server
//     issues a later deadline; the window then follows it, still capped at
//     now+MaxWorkflowSeconds. The server-side invocation counter is not reset
//     by a refresh, so the local count is kept as well.
//   - Entries are pruned only well after they can matter (expired for longer
//     than the longest delegation lifetime) and the map is bounded.
const (
	budgetRetention = 8 * 24 * time.Hour
	budgetCapacity  = 512
)

func (e *Engine) automationBudgetLocked(binding *core.ManagedAutomationBinding, consent AutomationWorkspace) (*automationBudget, error) {
	now := time.Now()
	if e.journal.AutomationBudgets == nil {
		e.journal.AutomationBudgets = map[string]*automationBudget{}
	}
	pruneBudgets(e.journal.AutomationBudgets, binding.WorkflowID, now)
	window := time.Duration(consent.MaxWorkflowSeconds) * time.Second
	budget := e.journal.AutomationBudgets[binding.WorkflowID]
	if budget == nil {
		budget = &automationBudget{FirstStartedAt: now, DeadlineAt: now.Add(window)}
		e.journal.AutomationBudgets[binding.WorkflowID] = budget
	}
	if now.Before(budget.FirstStartedAt) {
		return nil, budgetCause("local clock moved backwards")
	}
	if server := binding.DeadlineAt; server.After(budget.DeadlineAt) {
		budget.DeadlineAt = minDeadline(server, now.Add(window))
	}
	return budget, budgetRemaining(budget, consent)
}

func budgetRemaining(budget *automationBudget, consent AutomationWorkspace) error {
	if budget.Invocations >= consent.MaxNativeInvocations {
		return budgetCause(fmt.Sprintf("native invocations %d/%d", budget.Invocations, consent.MaxNativeInvocations))
	}
	if !time.Now().Before(budget.DeadlineAt) {
		return budgetCause("local workflow deadline passed")
	}
	return nil
}

func pruneBudgets(budgets map[string]*automationBudget, keep string, now time.Time) {
	for id, budget := range budgets {
		if id != keep && now.Sub(budget.DeadlineAt) > budgetRetention {
			delete(budgets, id)
		}
	}
	if len(budgets) < budgetCapacity {
		return
	}
	ids := make([]string, 0, len(budgets))
	for id := range budgets {
		if id != keep {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return budgets[ids[i]].DeadlineAt.Before(budgets[ids[j]].DeadlineAt) })
	for _, id := range ids[:len(budgets)-budgetCapacity+1] {
		delete(budgets, id)
	}
}
