package projects

import (
	"agent-board/internal/core"
	"context"
)

// Every entry point keeps current registry and legacy credential authority
// through the board transaction. Browser approval is a separate typed context.
func (s *service) AutomationPolicyCreate(ctx context.Context, a core.Actor, in core.AutomationPolicyCreateInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_create", s.board.AutomationPolicyCreate)
}
func (s *service) AutomationPolicyUpdate(ctx context.Context, a core.Actor, in core.AutomationPolicyUpdateInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_update", s.board.AutomationPolicyUpdate)
}
func (s *service) AutomationPolicyRevoke(ctx context.Context, a core.Actor, in core.AutomationPolicyRevokeInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_revoke", s.board.AutomationPolicyRevoke)
}
func (s *service) AutomationPolicyGet(ctx context.Context, a core.Actor, in core.AutomationPolicyInput) (core.AutomationPolicyPage, error) {
	return automationMethodCall(s, ctx, a, in, false, "automation_policy_get", s.board.AutomationPolicyGet)
}
func (s *service) WorkflowSubmit(ctx context.Context, a core.Actor, in core.WorkflowSubmitInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, true, "workflow_submit", s.board.WorkflowSubmit)
}
func (s *service) WorkflowList(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowListPage, error) {
	return automationMethodCall(s, ctx, a, in, false, "workflow_list", s.board.WorkflowList)
}

// AutomationArchive pages audited history rotation rows (read-only) for the
// people who manage the project: owners and global admins with a membership.
// REST POST /api/v1/p/{id}/automation/archive and MCP resource
// accord://automation-archive bind it in internal/transport.
func (s *service) AutomationArchive(ctx context.Context, a core.Actor, in core.AutomationArchiveInput) (core.AutomationArchivePage, error) {
	if err := automationOperationAllowed(ctx, "automation_archive"); err != nil {
		return core.AutomationArchivePage{}, err
	}
	return automationAccessCall(s, ctx, a, in, false, true, s.board.AutomationArchive)
}
func (s *service) WorkflowGet(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, false, "workflow_get", s.board.WorkflowGet)
}
func (s *service) WorkflowReport(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, false, "workflow_report", s.board.WorkflowReport)
}
func (s *service) WorkflowCommand(ctx context.Context, a core.Actor, in core.WorkflowCommandInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, true, "workflow_command", s.board.WorkflowCommand)
}
func (s *service) WorkflowPlan(ctx context.Context, a core.Actor, in core.WorkflowPlanInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, true, "workflow_plan", s.board.WorkflowPlan)
}
func (s *service) WorkflowQuestion(ctx context.Context, a core.Actor, in core.WorkflowQuestionInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, true, "workflow_question", s.board.WorkflowQuestion)
}
func (s *service) WorkflowAnswer(ctx context.Context, a core.Actor, in core.WorkflowAnswerInput) (core.WorkflowPage, error) {
	return automationMethodCall(s, ctx, a, in, true, "workflow_answer", s.board.WorkflowAnswer)
}
func (s *service) WorkflowAttest(ctx context.Context, a core.Actor, in core.WorkflowAttestInput) (core.WorkflowAttestResult, error) {
	write := in.Kind != "run_exit" && in.Kind != "check_finished" && in.Kind != "check_unknown"
	return automationMethodCall(s, ctx, a, in, write, "workflow_attest", s.board.WorkflowAttest)
}
func (s *service) ExecutionCapabilityRegister(ctx context.Context, a core.Actor, in core.ExecutionCapabilityRegisterInput) (core.ExecutionCapability, error) {
	return automationMethodCall(s, ctx, a, in, true, "execution_capability_register", s.board.ExecutionCapabilityRegister)
}

func (s *service) AutomationPermission(ctx context.Context, a core.Actor, in core.AutomationPermissionInput) error {
	_, err := automationCall(s, ctx, a, in, false, func(inner context.Context, actor core.Actor, input core.AutomationPermissionInput) (struct{}, error) {
		return struct{}{}, s.board.AutomationPermission(inner, actor, input)
	})
	return err
}

func (s *service) AutomationAvailable() bool { return s.registry != nil }
