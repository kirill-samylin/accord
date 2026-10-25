package worker

import (
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestLocalBudgetFollowsServerDeadlineAndNamesExhaustion(t *testing.T) {
	e := testEngine(t)
	consent := AutomationWorkspace{MaxNativeInvocations: 2, MaxWorkflowSeconds: 60}
	binding := &core.ManagedAutomationBinding{WorkflowID: "wf_budget", DeadlineAt: time.Now().Add(30 * time.Second)}
	e.mu.Lock()
	defer e.mu.Unlock()
	budget, err := e.automationBudgetLocked(binding, consent)
	if err != nil || budget.Invocations != 0 {
		t.Fatal("preparation alone consumed an invocation", err)
	}
	budget.Invocations = 2
	if _, err = e.automationBudgetLocked(binding, consent); err == nil || !strings.HasPrefix(err.Error(), "local_budget_exhausted: ") || !strings.Contains(err.Error(), "2/2") {
		t.Fatal("exhaustion lost its specific cause", err)
	}
	// An explicit human refresh issues a later server deadline: the local
	// window follows it, still capped at now+MaxWorkflowSeconds.
	budget.Invocations = 1
	budget.DeadlineAt = time.Now().Add(-time.Minute)
	binding.DeadlineAt = time.Now().Add(10 * time.Minute)
	if _, err = e.automationBudgetLocked(binding, consent); err != nil {
		t.Fatal("refreshed server deadline was ignored", err)
	}
	if budget.DeadlineAt.After(time.Now().Add(61 * time.Second)) {
		t.Fatal("server deadline widened the local workflow window")
	}
	e.journal.AutomationBudgets["wf_old"] = &automationBudget{DeadlineAt: time.Now().Add(-9 * 24 * time.Hour)}
	if _, err = e.automationBudgetLocked(binding, consent); err != nil || e.journal.AutomationBudgets["wf_old"] != nil {
		t.Fatal("stale workflow budget was not pruned", err)
	}
}

func TestPreparationCausesAreSpecificAndPathFree(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{dirtyWorkspaceCause("repo", 2, false), "workspace_dirty"},
		{errTargetSHA, "target_sha_mismatch"},
		{budgetCause("native invocations 4/4"), "local_budget_exhausted"},
		{errProofOpen, "previous_proof_unresolved"},
		{registrationCause(statusError{code: 403, server: "FORBIDDEN_ROLE", message: "Workflow остановлен"}), "registration_rejected"},
		{errIPC, "ipc_unavailable"},
		{pinError{"provider", "codex"}, "provider_pin_changed"},
		{pinError{"workspace", "repo"}, "workspace_pin_changed"},
		{errPermissionProof, "native_git_permission_unproven"},
	} {
		code, message := classifyPreparation(tc.err)
		if code != tc.code || message == "" || strings.Contains(message, "/Users/") {
			t.Fatal("cause lost", tc.code, code, message)
		}
	}
	if _, message := classifyPreparation(pinError{"provider", "codex"}); !strings.Contains(message, "local workspace or provider identity changed") {
		t.Fatal("owner guidance pattern lost", message)
	}
	if !strings.Contains(dirtyWorkspaceCause("repo", 2, false).Error(), "2 uncommitted") {
		t.Fatal("dirty cause does not count files")
	}
}
