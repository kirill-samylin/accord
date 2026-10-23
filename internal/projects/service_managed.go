package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
)

// managedCall keeps the registry guard through the selected board transaction.
// Host grants augment membership; a project editor cannot control arbitrary PCs.
func managedCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, host, run string, write bool, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	return automationCall(s, ctx, a, in, write, func(inner context.Context, actor core.Actor, input I) (O, error) {
		var zero O
		authority, current := store.AutomationAuthorityFromContext(inner)
		if principal, ok := authority.Principals[actor.Principal.ID]; !current || !ok || principal.Principal != actor.Principal {
			return zero, denied()
		}
		if host != "" || run != "" {
			h, err := s.board.ManagedAuthority(inner, actor, host, run)
			if err != nil {
				return zero, err
			}
			grant, exists := authority.Principals[h.OwnerPrincipalID]
			if !current || !exists || grant.Epoch != h.OwnerAccessEpoch {
				return zero, core.Fail("unauthorized", "Владелец компьютера должен повторно подключить worker")
			}
		}
		return call(inner, actor, input)
	})
}
func (s *service) ManagedList(ctx context.Context, a core.Actor, in core.ManagedListInput) (core.ManagedPage, error) {
	if err := automationOperationAllowed(ctx, "managed_list"); err != nil {
		return core.ManagedPage{}, err
	}
	return withAccess(s, ctx, false, a.SessionID, func(ctx context.Context, access access) (core.ManagedPage, error) {
		if a.Principal != access.principal {
			return core.ManagedPage{}, denied()
		}
		out, err := s.board.ManagedList(ctx, a, in)
		if err != nil {
			return out, err
		}
		for i := range out.Hosts {
			h := &out.Hosts[i]
			var epoch int64
			var active bool
			var role string
			e := access.tx.QueryRowContext(ctx, `SELECT access_epoch,active,role FROM memberships WHERE project_id=? AND local_id=?`, s.projectID, h.OwnerPrincipalID).Scan(&epoch, &active, &role)
			h.CanControl = h.CanControl && access.project.CanWrite && e == nil && active && role != "viewer" && epoch == h.OwnerAccessEpoch
		}
		return out, nil
	})
}
func (s *service) ManagedRun(ctx context.Context, a core.Actor, in core.ManagedRunInput) (core.ManagedRunPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "managed_run", s.board.ManagedRun)
}
func (s *service) ManagedHostEnroll(ctx context.Context, a core.Actor, in core.ManagedHostEnrollInput) (core.ManagedHost, error) {
	return managedCall(s, ctx, a, in, "", "", true, s.board.ManagedHostEnroll)
}
func (s *service) ManagedHostUpdate(ctx context.Context, a core.Actor, in core.ManagedHostUpdateInput) (core.ManagedHost, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", true, s.board.ManagedHostUpdate)
}
func (s *service) ManagedHostDisable(ctx context.Context, a core.Actor, in core.ManagedHostDisableInput) (core.ManagedHost, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", true, s.board.ManagedHostDisable)
}
func (s *service) ManagedStart(ctx context.Context, a core.Actor, in core.ManagedStartInput) (core.ManagedResult, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", true, s.board.ManagedStart)
}
func (s *service) ManagedCommand(ctx context.Context, a core.Actor, in core.ManagedCommandInput) (core.ManagedResult, error) {
	return managedCall(s, ctx, a, in, "", in.RunID, true, s.board.ManagedCommand)
}
func (s *service) ManagedPoll(ctx context.Context, a core.Actor, in core.ManagedPollInput) (core.ManagedPollResult, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", false, s.board.ManagedPoll)
}
func (s *service) ManagedReport(ctx context.Context, a core.Actor, in core.ManagedReportInput) (core.ManagedResult, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", false, s.board.ManagedReport)
}
func (s *service) ManagedPermission(ctx context.Context, a core.Actor, in core.ManagedPermissionInput) (core.ManagedHost, error) {
	return managedCall(s, ctx, a, in, in.HostID, in.RunID, true, s.board.ManagedPermission)
}
