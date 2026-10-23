package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

// These contexts are populated only inside the registry/legacy authority guards.
// Never decode them from a transport request or persist credential material.
type AutomationPrincipal struct {
	Principal     core.Principal
	Epoch         int64
	LegacyTokenID string
	ExpiresAt     time.Time
}
type AutomationAuthority struct {
	ProjectID      string
	Writable       bool
	Principals     map[string]AutomationPrincipal
	ReadPrincipals map[string]AutomationPrincipal
}
type AutomationPolicyApproval struct {
	PrincipalID    string
	SessionID      string
	HostID         string
	ConsentVersion int64
	Epoch          int64
	Source         string // web (issued action + CSRF), or local (verified host proof)
	ActionID       string
	HostProof      string
}
type automationAuthorityKey struct{}
type automationApprovalKey struct{}
type executionIdentityKey struct{}

func WithAutomationAuthority(ctx context.Context, authority AutomationAuthority) context.Context {
	return context.WithValue(ctx, automationAuthorityKey{}, authority)
}
func WithAutomationPolicyApproval(ctx context.Context, approval AutomationPolicyApproval) context.Context {
	return context.WithValue(ctx, automationApprovalKey{}, approval)
}
func WithExecutionCapability(ctx context.Context, identity core.ExecutionIdentity) context.Context {
	return context.WithValue(ctx, executionIdentityKey{}, identity)
}

func ExecutionIdentityFromContext(ctx context.Context) (core.ExecutionIdentity, bool) {
	identity, ok := ctx.Value(executionIdentityKey{}).(core.ExecutionIdentity)
	return identity, ok
}
func AutomationAuthorityFromContext(ctx context.Context) (AutomationAuthority, bool) {
	authority, ok := ctx.Value(automationAuthorityKey{}).(AutomationAuthority)
	return authority, ok
}

// AutomationIntakeApproval is a trusted browser issued-action proof. Native and
// public REST adapters must never manufacture it from request JSON.
type AutomationIntakeApproval struct {
	PrincipalID string
	SessionID   string
	ActionID    string
}
type automationIntakeKey struct{}

func WithAutomationIntakeApproval(ctx context.Context, approval AutomationIntakeApproval) context.Context {
	return context.WithValue(ctx, automationIntakeKey{}, approval)
}
