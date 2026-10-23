package worker

import (
	"context"
	"strings"
	"testing"

	"agent-board/internal/core"
)

func TestAutomationPromptMatchesCheckRoleAuthority(t *testing.T) {
	for _, tc := range []struct {
		role       string
		ownsChecks bool
		want       []string
	}{
		{"analyst", false, []string{"structured plan through workflow_plan", "workflow_question", "Planning does not require starting local verification presets"}},
		{"developer", true, []string{"Developer presets require a clean committed HEAD first"}},
		{"reviewer", false, []string{"read-only review", "exact target SHA \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" and its diff", "available qualified verification evidence", "approve, changes_requested or blocked", "Do not start local verification presets as the reviewer"}},
		{"tester", true, []string{"Reviewer/tester must preserve every reviewed source byte and full HEAD", "test_design/NOT-RUN, never PASSED"}},
	} {
		t.Run(tc.role, func(t *testing.T) {
			command, run := testDispatch("start")
			run.Profile.Role = tc.role
			run.TaskKey = "T-1"
			run.Automation = &core.ManagedAutomationBinding{TargetSHA: strings.Repeat("a", 40)}
			e := &Engine{config: Config{Machine: "test-machine", Automation: &AutomationConfig{Workspaces: map[string]AutomationWorkspace{"test": {ArtifactRoots: []string{".artifacts"}}}}}}
			prompt := e.prompt(plan{command: command, run: run})
			if got := strings.Contains(prompt, "workflow_run_check"); got != tc.ownsChecks {
				t.Fatalf("role %s check instruction=%t, want %t", tc.role, got, tc.ownsChecks)
			}
			if tc.ownsChecks {
				tc.want = append(tc.want, "required owner-approved local verification preset IDs", "status resource to be completed and inspect its actual result before transitioning", "Queued/running, failed, NOT-RUN and N/A are never PASSED")
			}
			for _, want := range append(tc.want, "acquire work with next", "LEASE_UNCERTAIN or LEASE_LOST", "Requested task: T-1", "synthetic profile", "User request:\nsynthetic user instruction") {
				if !strings.Contains(prompt, want) {
					t.Errorf("role %s missing instruction %q", tc.role, want)
				}
			}
		})
	}
}

func TestAutomationPromptDoesNotGrantAnalystOrReviewerChecks(t *testing.T) {
	for _, role := range []string{"analyst", "reviewer"} {
		t.Run(role, func(t *testing.T) {
			e, request := automationTestEngine(t)
			a := e.journal.Runs[request.RunID].Automation
			a.Run.Profile.Role = role
			// A fully bound, consented preset still cannot bypass the role guard.
			if _, err := e.automationCheckRequest(context.Background(), request.RunID, request); err == nil {
				t.Fatal("non-executing role started a local check")
			}
			if len(a.Checks) != 0 || e.active[request.RunID].automation.checkCount != 0 {
				t.Fatal("refused role created check execution intent")
			}
		})
	}
}

func TestRoleCheckPromptPreservesManualAndContinuationRequests(t *testing.T) {
	command, run := testDispatch("clarify")
	command.Text = "synthetic clarification"
	e := &Engine{config: Config{Machine: "test-machine"}}
	prompt := e.prompt(plan{command: command, run: run})
	if strings.Contains(prompt, "workflow_run_check") || strings.Contains(prompt, "workflow_plan") {
		t.Fatal("manual managed run received automation instructions")
	}
	if !strings.HasSuffix(prompt, "User request:\nsynthetic clarification") || strings.Contains(prompt, run.Prompt) {
		t.Fatal("continuation did not preserve the supplied clarification")
	}
}
