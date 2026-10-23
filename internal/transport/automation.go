package transport

import (
	"agent-board/internal/core"
	"context"
)

// Registry authority is required; a standalone board cannot delegate execution.
type AutomationService interface {
	AutomationAvailable() bool
	WorkflowSubmit(context.Context, core.Actor, core.WorkflowSubmitInput) (core.WorkflowPage, error)
	WorkflowCommand(context.Context, core.Actor, core.WorkflowCommandInput) (core.WorkflowPage, error)
	WorkflowPlan(context.Context, core.Actor, core.WorkflowPlanInput) (core.WorkflowPage, error)
	WorkflowQuestion(context.Context, core.Actor, core.WorkflowQuestionInput) (core.WorkflowPage, error)
	WorkflowAnswer(context.Context, core.Actor, core.WorkflowAnswerInput) (core.WorkflowPage, error)
	WorkflowGet(context.Context, core.Actor, core.WorkflowReadInput) (core.WorkflowPage, error)
	WorkflowReport(context.Context, core.Actor, core.WorkflowReadInput) (core.WorkflowPage, error)
	WorkflowList(context.Context, core.Actor, core.WorkflowReadInput) (core.WorkflowListPage, error)
	WorkflowAttest(context.Context, core.Actor, core.WorkflowAttestInput) (core.WorkflowAttestResult, error)
	ExecutionCapabilityRegister(context.Context, core.Actor, core.ExecutionCapabilityRegisterInput) (core.ExecutionCapability, error)
}

func automationTools(s Service) []operation {
	a, ok := s.(AutomationService)
	if !ok || !a.AutomationAvailable() {
		return nil
	}
	return []operation{
		bind("workflow_submit", "Submit a requirement to an existing owner-approved policy. Without a policy only a draft is created; no agents start.", false, a.WorkflowSubmit),
		bind("workflow_command", "Pause, resume, stop, clarify or replan an authorized workflow using its current version. Queued control is not completed execution.", false, a.WorkflowCommand),
		bind("workflow_plan", "Publish the analyst plan for this exact leased workflow step.", false, a.WorkflowPlan),
		bind("workflow_question", "Ask a blocking question for this exact workflow step.", false, a.WorkflowQuestion),
	}
}
func automationOperations(s Service) []operation {
	a, ok := s.(AutomationService)
	if !ok || !a.AutomationAvailable() {
		return nil
	}
	return []operation{
		bind("workflow_list", "Read a bounded workflow page.", true, a.WorkflowList),
		bind("workflow_get", "Read the workflow, steps, questions and evidence.", true, a.WorkflowGet),
		bind("workflow_report", "Read a workflow report with actual evidence.", true, a.WorkflowReport),
		bind("workflow_answer", "Answer an open workflow question as an authorized human.", false, a.WorkflowAnswer),
		bind("workflow_attest", "Record an authenticated worker observation.", false, a.WorkflowAttest),
		bind("execution_capability_register", "Register an exact dispatched attempt capability using local host proof.", false, a.ExecutionCapabilityRegister),
	}
}

var automationRoutes = []route{
	{"POST", "/api/v1/automation/policies/revoke", "automation_policy_revoke", true},
	{"POST", "/api/v1/automation/policies/create", "automation_policy_create", true},
	{"POST", "/api/v1/automation/policies/update", "automation_policy_update", true},
	{"POST", "/api/v1/automation/policies/get", "automation_policy_get", true},
	{"POST", "/api/v1/automation/submit", "workflow_submit", true},
	{"POST", "/api/v1/automation/command", "workflow_command", true},
	{"POST", "/api/v1/automation/plan", "workflow_plan", true},
	{"POST", "/api/v1/automation/question", "workflow_question", true},
	{"POST", "/api/v1/automation/answer", "workflow_answer", true},
	{"POST", "/api/v1/automation/list", "workflow_list", true},
	{"POST", "/api/v1/automation/get", "workflow_get", true},
	{"POST", "/api/v1/automation/report", "workflow_report", true},
	{"POST", "/api/v1/automation/attest", "workflow_attest", true},
	{"POST", "/api/v1/automation/capabilities/register", "execution_capability_register", true},
}
