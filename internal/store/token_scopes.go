package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

func tokenRoles(b *board, in core.TokenInput) ([]string, error) {
	roles := append([]string(nil), in.Roles...)
	if in.Roles == nil {
		for name, r := range b.Config.Workflow.Roles {
			if r.Kind == "agent" {
				roles = append(roles, name)
			}
		}
		slices.Sort(roles)
	}
	if len(roles) == 0 {
		return nil, core.Fail("VALIDATION", "at least one role scope is required")
	}
	seen := map[string]bool{}
	for _, name := range roles {
		r, ok := b.Config.Workflow.Roles[name]
		if !ok || seen[name] {
			return nil, core.Fail("VALIDATION", "unknown or duplicate role scope")
		}
		seen[name] = true
		if r.Kind == "human" && !slices.Contains(b.Config.Workflow.Auth.Local.Users, in.Owner) {
			return nil, core.Fail("FORBIDDEN_ROLE", "human scope owner must be allowlisted")
		}
	}
	slices.Sort(roles)
	return roles, nil
}

// SetTokenScopes is local-only CAS administration; remote credentials cannot
// invoke this method. Active session scopes are retained until their expiry.
func (s *Store) SetTokenScopes(ctx context.Context, in core.ScopesInput) (core.TokenInfo, error) {
	var out core.TokenInfo
	if !bounded(in.Actor, 128) || in.ExpectedVersion < 1 || in.Roles == nil {
		return out, core.Fail("VALIDATION", "actor, roles and expected-version required")
	}
	err := s.adminTransaction(ctx, func(b *board, now time.Time) error {
		var raw []byte
		var expires int64
		if err := b.tx.QueryRowContext(ctx, `SELECT id,owner,machine,expires,revoked,roles,scopes_version FROM tokens WHERE id=?`, in.ID).Scan(&out.ID, &out.Owner, &out.Machine, &expires, &out.Revoked, &raw, &out.ScopesVersion); err != nil {
			if err == sql.ErrNoRows {
				return core.Fail("NOT_FOUND", "token does not exist")
			}
			return err
		}
		if out.ScopesVersion != in.ExpectedVersion {
			return core.Fail("CONFIG_CONFLICT", "token scopes version differs")
		}
		roles, err := tokenRoles(b, core.TokenInput{Owner: out.Owner, Roles: in.Roles})
		if err != nil {
			return err
		}
		for _, session := range b.Sessions {
			if session.PrincipalID == in.ID && sessionLive(b, session, now) && !slices.Contains(roles, session.Role) {
				return core.Fail("CONFIG_IN_USE", "session:"+session.SessionID)
			}
		}
		encoded, err := json.Marshal(roles)
		if err != nil {
			return err
		}
		if _, err = b.tx.ExecContext(ctx, `UPDATE tokens SET roles=?,scopes_version=scopes_version+1 WHERE id=? AND scopes_version=?`, string(encoded), in.ID, in.ExpectedVersion); err != nil {
			return err
		}
		out.ScopesVersion++
		out.Roles = roles
		out.ExpiresAt = time.Unix(0, expires).UTC()
		emit(b, now, "token_scopes_changed", "", "")
		e := &b.Events[len(b.Events)-1]
		e.TokenID = in.ID
		e.Reason = fmt.Sprintf("actor=%s version=%d roles=%s", in.Actor, out.ScopesVersion, encoded)
		return nil
	})
	return out, err
}
