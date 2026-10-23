package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
	"context"
)

func localApproval(ctx context.Context, a core.Actor, host string, version int64, proof string) (context.Context, error) {
	authority, ok := store.AutomationAuthorityFromContext(ctx)
	p, found := authority.Principals[a.Principal.ID]
	if !ok || !found || proof == "" {
		return ctx, denied()
	}
	return store.WithAutomationPolicyApproval(ctx, store.AutomationPolicyApproval{PrincipalID: a.Principal.ID, SessionID: a.SessionID, HostID: host, ConsentVersion: version, Epoch: p.Epoch, Source: "local", HostProof: proof}), nil
}
func (s *service) AutomationPolicyLocalCreate(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalCreateInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_create", func(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalCreateInput) (core.AutomationPolicy, error) {
		inner, err := localApproval(ctx, a, in.HostID, in.HostConsentVersion, in.HostProof)
		if err != nil {
			return core.AutomationPolicy{}, err
		}
		return s.board.AutomationPolicyCreate(inner, a, in.AutomationPolicyCreateInput)
	})
}
func (s *service) AutomationPolicyLocalUpdate(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalUpdateInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_update", func(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalUpdateInput) (core.AutomationPolicy, error) {
		inner, err := localApproval(ctx, a, in.HostID, in.HostConsentVersion, in.HostProof)
		if err != nil {
			return core.AutomationPolicy{}, err
		}
		return s.board.AutomationPolicyUpdate(inner, a, in.AutomationPolicyUpdateInput)
	})
}

func (s *service) AutomationPolicyLocalRevoke(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalRevokeInput) (core.AutomationPolicy, error) {
	return automationMethodCall(s, ctx, a, in, true, "automation_policy_revoke", func(ctx context.Context, a core.Actor, in transport.AutomationPolicyLocalRevokeInput) (core.AutomationPolicy, error) {
		page, err := s.board.AutomationPolicyGet(ctx, a, core.AutomationPolicyInput{PolicyID: in.PolicyID})
		if err != nil {
			return core.AutomationPolicy{}, err
		}
		inner, err := localApproval(ctx, a, page.Policy.HostID, in.HostConsentVersion, in.HostProof)
		if err != nil {
			return core.AutomationPolicy{}, err
		}
		return s.board.AutomationPolicyRevoke(inner, a, in.AutomationPolicyRevokeInput)
	})
}
