package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"database/sql"
	"encoding/hex"
	"strings"
)

func executionNativePath(path string) bool {
	if path == "/mcp" {
		return true
	}
	for _, allowed := range []string{"/api/v1/session", "/api/v1/heartbeat", "/api/v1/tasks/next", "/api/v1/automation/get", "/api/v1/automation/report", "/api/v1/automation/plan", "/api/v1/automation/question"} {
		if path == allowed {
			return true
		}
	}
	return strings.HasPrefix(path, "/api/v1/tasks/")
}

// Execution tokens are looked up only inside their explicit project. They are
// never accepted by browser/catalog or registry administrative authentication.
func (r *Registry) nativeIdentity(ctx context.Context, token, project, path string, scoped bool) (context.Context, Identity, error) {
	if !strings.HasPrefix(token, core.ExecutionTokenPrefix) {
		who, err := r.Authenticate(ctx, token)
		return accountContext(ctx, who), who, err
	}
	var zero Identity
	secret := strings.TrimPrefix(token, core.ExecutionTokenPrefix)
	decoded, decodeErr := hex.DecodeString(secret)
	if !scoped || !validID(project, "p_") || !executionNativePath(path) || decodeErr != nil || len(decoded) != 32 || secret != strings.ToLower(secret) {
		return ctx, zero, denied()
	}
	var state string
	if err := r.db.QueryRowContext(ctx, `SELECT state FROM projects WHERE id=?`, project).Scan(&state); err != nil || state != "active" {
		return ctx, zero, denied()
	}
	v, err := r.runtime(project)
	if err != nil {
		return ctx, zero, err
	}
	type result struct {
		who       Identity
		execution core.ExecutionIdentity
	}
	auth, err := transaction(r, ctx, func(tx *sql.Tx) (result, error) {
		p, err := r.project(ctx, tx, project)
		if err != nil || p.State != "active" {
			return result{}, denied()
		}
		identity, err := automationGuard(v.service, ctx, tx, p, core.Principal{}, nil, func(inner context.Context) (core.ExecutionIdentity, error) {
			return v.store.AuthenticateExecutionCapability(inner, token)
		})
		if err != nil {
			return result{}, err
		}
		var account string
		err = tx.QueryRowContext(ctx, `SELECT identity_id FROM memberships WHERE project_id=? AND local_id=? AND active=1 AND role IN ('owner','editor')`, project, identity.Principal.ID).Scan(&account)
		if err != nil {
			return result{}, denied()
		}
		who, err := r.identity(ctx, tx, account)
		if err != nil {
			return result{}, err
		}
		return result{who, identity}, nil
	})
	if err != nil {
		return ctx, zero, err
	}
	return store.WithExecutionCapability(accountContext(ctx, auth.who), auth.execution), auth.who, nil
}

func (s *service) authenticateExecution(ctx context.Context, token string) (core.Principal, error) {
	var zero core.Principal
	expected, bound := store.ExecutionIdentityFromContext(ctx)
	if !bound {
		return zero, denied()
	}
	return transaction(s.registry, ctx, func(tx *sql.Tx) (core.Principal, error) {
		who, err := accountFrom(ctx)
		if err != nil {
			return zero, err
		}
		access, err := s.registry.access(ctx, tx, who, s.projectID, true)
		if err != nil {
			return zero, err
		}
		required := core.Principal{}
		if who.Source == "legacy" {
			required = core.Principal{ID: who.TokenID, Owner: who.Owner, Machine: who.Machine}
		}
		return automationGuard(s, ctx, tx, access.project, required, nil, func(inner context.Context) (core.Principal, error) {
			actual, err := s.board.AuthenticateExecutionCapability(inner, token)
			if err != nil {
				return zero, err
			}
			if actual.Principal != access.principal || actual.Capability.ID != expected.Capability.ID {
				return zero, denied()
			}
			return actual.Principal, nil
		})
	})
}
