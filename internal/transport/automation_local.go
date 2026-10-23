package transport

import (
	"agent-board/internal/core"
	"context"
)

// HostProof is transported only by the owner CLI, never a model tool. Registry
// wrappers place it into typed approval context; it is not an idempotency payload.
type AutomationPolicyLocalCreateInput struct {
	core.AutomationPolicyCreateInput
	HostProof string `json:"host_proof"`
}
type AutomationPolicyLocalUpdateInput struct {
	core.AutomationPolicyUpdateInput
	HostProof string `json:"host_proof"`
}
type AutomationPolicyLocalRevokeInput struct {
	core.AutomationPolicyRevokeInput
	HostProof          string `json:"host_proof"`
	HostConsentVersion int64  `json:"host_consent_version"`
}
type LocalAutomationService interface {
	AutomationPolicyLocalRevoke(context.Context, core.Actor, AutomationPolicyLocalRevokeInput) (core.AutomationPolicy, error)
	AutomationPolicyLocalCreate(context.Context, core.Actor, AutomationPolicyLocalCreateInput) (core.AutomationPolicy, error)
	AutomationPolicyLocalUpdate(context.Context, core.Actor, AutomationPolicyLocalUpdateInput) (core.AutomationPolicy, error)
	AutomationPolicyGet(context.Context, core.Actor, core.AutomationPolicyInput) (core.AutomationPolicyPage, error)
}

func localAutomationOperations(s Service) []operation {
	a, ok := s.(LocalAutomationService)
	if !ok {
		return nil
	}
	return []operation{
		bind("automation_policy_create", "Approve a bounded policy using current local owner proof.", false, a.AutomationPolicyLocalCreate),
		bind("automation_policy_update", "Replace a policy using exact version and current owner proof.", false, a.AutomationPolicyLocalUpdate),
		bind("automation_policy_revoke", "Revoke a policy using current local owner proof.", false, a.AutomationPolicyLocalRevoke),
		bind("automation_policy_get", "Read a policy and its immutable revisions.", true, a.AutomationPolicyGet),
	}
}
