package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"database/sql"
)

// managedCall keeps the registry guard through the selected board transaction.
// Host grants augment membership; a project editor cannot control arbitrary PCs.
func managedCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, host, run string, write bool, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	return transaction(s.registry, ctx, func(tx *sql.Tx) (O, error) {
		var zero O
		who, err := accountFrom(ctx)
		if err != nil {
			return zero, err
		}
		access, err := s.registry.access(ctx, tx, who, s.projectID, write)
		if err != nil {
			return zero, err
		}
		if a.Principal != access.principal {
			return zero, denied()
		}
		if err = s.registry.sessionAllowed(ctx, tx, access, a.SessionID); err != nil {
			return zero, err
		}
		type issuer struct {
			local, identity, source, token, owner, machine string
			epoch                                          int64
		}
		issuers := []issuer{}
		rows, err := tx.QueryContext(ctx, `SELECT m.local_id,m.identity_id,m.access_epoch,i.source,i.token_id,i.owner,i.machine FROM memberships m JOIN identities i ON i.id=m.identity_id WHERE m.project_id=? AND m.active=1 AND m.role IN ('owner','editor')`, s.projectID)
		if err != nil {
			return zero, err
		}
		for rows.Next() {
			var v issuer
			if err = rows.Scan(&v.local, &v.identity, &v.epoch, &v.source, &v.token, &v.owner, &v.machine); err != nil {
				rows.Close()
				return zero, err
			}
			issuers = append(issuers, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return zero, err
		}
		grants := map[string]store.ManagedPrincipalGrant{}
		legacy := []core.Principal{}
		for _, v := range issuers {
			if v.source == "legacy" {
				legacy = append(legacy, core.Principal{ID: v.token, Owner: v.owner, Machine: v.machine})
			} else if _, e := s.registry.identity(ctx, tx, v.identity); e == nil {
				grants[v.local] = store.ManagedPrincipalGrant{Epoch: v.epoch}
			}
		}
		var out O
		invoke := func(valid map[string]bool) error {
			scoped := map[string]store.ManagedPrincipalGrant{}
			for id, v := range grants {
				scoped[id] = v
			}
			for _, v := range issuers {
				if v.source == "legacy" && (valid == nil || valid[v.token]) {
					g := store.ManagedPrincipalGrant{Epoch: v.epoch}
					if valid == nil {
						g.LegacyTokenID = v.token
					}
					scoped[v.local] = g
				}
			}
			inner := store.WithManagedEpoch(store.WithManagedPrincipals(ctx, scoped), access.member.Epoch)
			if host != "" || run != "" {
				h, e := s.board.ManagedAuthority(inner, a, host, run)
				if e != nil {
					return e
				}
				grant, ok := scoped[h.OwnerPrincipalID]
				if !ok || grant.Epoch != h.OwnerAccessEpoch {
					return core.Fail("unauthorized", "Владелец компьютера должен повторно подключить worker")
				}
			}
			out, err = call(inner, a, in)
			return err
		}
		if s.projectID == s.registry.defaultID || len(legacy) == 0 {
			err = invoke(nil)
		} else {
			required := ""
			if access.identity.Source == "legacy" {
				required = access.identity.TokenID
			}
			err = s.registry.options.Default.GuardManagedCredentials(ctx, required, legacy, invoke)
		}
		return out, err
	})
}
func (s *service) ManagedList(ctx context.Context, a core.Actor, in core.ManagedListInput) (core.ManagedPage, error) {
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
	return actorCall(s, ctx, a, in, false, s.board.ManagedRun)
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
	return managedCall(s, ctx, a, in, in.HostID, "", true, s.board.ManagedPoll)
}
func (s *service) ManagedReport(ctx context.Context, a core.Actor, in core.ManagedReportInput) (core.ManagedResult, error) {
	return managedCall(s, ctx, a, in, in.HostID, "", true, s.board.ManagedReport)
}
func (s *service) ManagedPermission(ctx context.Context, a core.Actor, in core.ManagedPermissionInput) (core.ManagedHost, error) {
	return managedCall(s, ctx, a, in, in.HostID, in.RunID, true, s.board.ManagedPermission)
}
