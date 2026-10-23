package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"slices"
)

// A capability cannot gain ordinary project read/admin rights by choosing a
// different REST method or MCP resource. Store also checks its current scope.
func automationOperationAllowed(ctx context.Context, operation string) error {
	identity, bound := store.ExecutionIdentityFromContext(ctx)
	if bound && !slices.Contains(identity.Capability.Operations, operation) {
		return core.Fail("forbidden", "Операция не разрешена дочерней попытке")
	}
	return nil
}

func nativeActorCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, write bool, operation string, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	var zero O
	if err := automationOperationAllowed(ctx, operation); err != nil {
		return zero, err
	}
	if _, bound := store.ExecutionIdentityFromContext(ctx); bound {
		return automationCall(s, ctx, a, in, write, call)
	}
	return actorCall(s, ctx, a, in, write, call)
}

func automationMethodCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, write bool, operation string, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	var zero O
	if err := automationOperationAllowed(ctx, operation); err != nil {
		return zero, err
	}
	return automationCall(s, ctx, a, in, write, call)
}

// Missing optional binding fields are supplied by the server. Explicitly
// conflicting fields fail; the model never chooses another run/profile/role.
func executionOpen(ctx context.Context, p core.Principal, in core.OpenSession) (core.OpenSession, error) {
	identity, bound := store.ExecutionIdentityFromContext(ctx)
	if !bound {
		return in, nil
	}
	c := identity.Capability
	for _, field := range []struct{ got, want string }{{in.ManagedRunID, c.RunID}, {in.RunID, c.RunID}, {in.Role, c.Role}, {in.ProfileID, c.ProfileID}, {in.Machine, p.Machine}} {
		if field.got != "" && field.got != field.want {
			return in, denied()
		}
	}
	if in.ManagedGeneration != 0 && in.ManagedGeneration != c.Generation || in.ProfileVersion != 0 && in.ProfileVersion != c.ProfileVersion {
		return in, denied()
	}
	in.ManagedRunID, in.RunID, in.Role, in.ProfileID, in.Machine = c.RunID, c.RunID, c.Role, c.ProfileID, p.Machine
	in.ManagedGeneration, in.ProfileVersion = c.Generation, c.ProfileVersion
	return in, nil
}
