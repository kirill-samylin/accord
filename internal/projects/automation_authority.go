package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"database/sql"
	"time"
)

type automationIssuer struct {
	local, identity, source, token, owner, machine string
	epoch                                          int64
	expires                                        int64
}

// automationIssuers is read only while the registry writer guard is held. The
// legacy credential writer guard is acquired next, before the target board.
func (r *Registry) automationIssuers(ctx context.Context, tx *sql.Tx, project string) ([]automationIssuer, error) {
	rows, err := tx.QueryContext(ctx, `SELECT m.local_id,m.identity_id,m.access_epoch,i.source,i.token_id,i.owner,i.machine,COALESCE(c.expires,0)
FROM memberships m JOIN identities i ON i.id=m.identity_id
LEFT JOIN credentials c ON c.identity_id=i.id
WHERE m.project_id=? AND m.active=1 AND m.role IN ('owner','editor')
AND (i.source='legacy' OR (c.revoked=0 AND c.expires>?))`, project, time.Now().UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []automationIssuer
	for rows.Next() {
		var v automationIssuer
		if err = rows.Scan(&v.local, &v.identity, &v.epoch, &v.source, &v.token, &v.owner, &v.machine, &v.expires); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func automationGuard[T any](s *service, ctx context.Context, tx *sql.Tx, project Project, required core.Principal, reader *store.AutomationPrincipal, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	issuers, err := s.registry.automationIssuers(ctx, tx, s.projectID)
	if err != nil {
		return zero, err
	}
	var legacy []core.Principal
	seen := map[string]bool{}
	for _, v := range issuers {
		if v.source == "legacy" {
			legacy = append(legacy, core.Principal{ID: v.token, Owner: v.owner, Machine: v.machine})
			seen[v.token] = true
		}
	}
	if required.ID != "" && !seen[required.ID] {
		legacy = append(legacy, required)
	}
	var out T
	invoke := func(valid map[string]time.Time) error {
		principals := map[string]store.AutomationPrincipal{}
		managed := map[string]store.ManagedPrincipalGrant{}
		for _, v := range issuers {
			if v.source == "legacy" && valid != nil && valid[v.token].IsZero() {
				continue
			}
			p := store.AutomationPrincipal{Principal: core.Principal{ID: v.local, Owner: v.owner, Machine: v.machine}, Epoch: v.epoch}
			m := store.ManagedPrincipalGrant{Epoch: v.epoch}
			if v.source == "legacy" && valid == nil {
				p.LegacyTokenID = v.token
				m.LegacyTokenID = v.token
			} else if v.source == "legacy" {
				p.ExpiresAt = valid[v.token]
			} else {
				p.ExpiresAt = time.Unix(0, v.expires)
			}
			principals[v.local] = p
			managed[v.local] = m
		}
		inner := store.WithManagedPrincipals(ctx, managed)
		readers := map[string]store.AutomationPrincipal{}
		if reader != nil {
			readers[reader.Principal.ID] = *reader
		}
		inner = store.WithAutomationAuthority(inner, store.AutomationAuthority{ProjectID: s.projectID, Writable: project.State == "active", Principals: principals, ReadPrincipals: readers})
		out, err = fn(inner)
		return err
	}
	// A default-board mutation checks legacy tokens inside its own board write
	// transaction. Recursively acquiring that same DB lock would deadlock.
	if s.projectID == s.registry.defaultID || len(legacy) == 0 {
		err = invoke(nil)
	} else {
		err = s.registry.options.Default.GuardAutomationCredentials(ctx, required.ID, legacy, invoke)
	}
	return out, err
}

func automationCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, write bool, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	return automationAccessCall(s, ctx, a, in, write, false, call)
}

// automationAccessCall is automationCall that, with manage, also requires the
// caller to manage the project (owner membership or global admin member).
func automationAccessCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, write, manage bool, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
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
		if manage {
			if err = requireOwner(access); err != nil {
				return zero, err
			}
		}
		if err = s.registry.sessionAllowed(ctx, tx, access, a.SessionID); err != nil {
			return zero, err
		}
		required := core.Principal{}
		if access.identity.Source == "legacy" {
			required = core.Principal{ID: access.identity.TokenID, Owner: access.identity.Owner, Machine: access.identity.Machine}
		}
		reader := &store.AutomationPrincipal{Principal: access.principal, Epoch: access.member.Epoch, ExpiresAt: access.identity.Expires}
		if access.identity.Source == "legacy" && s.projectID == s.registry.defaultID {
			reader.LegacyTokenID = access.identity.TokenID
		}
		return automationGuard(s, ctx, tx, access.project, required, reader, func(inner context.Context) (O, error) {
			return call(store.WithManagedEpoch(inner, access.member.Epoch), a, in)
		})
	})
}
