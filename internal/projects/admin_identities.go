package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"
)

// IssuedCredential is the live answer to credential issuance. Secret exists
// only in that first response; the stored replay result carries
// secret_delivered instead (docs/admin.md).
type IssuedCredential struct {
	IdentityID      string    `json:"identity_id"`
	Owner           string    `json:"owner"`
	Machine         string    `json:"machine"`
	ExpiresAt       time.Time `json:"expires_at"`
	Secret          string    `json:"secret,omitempty"`
	SecretDelivered bool      `json:"secret_delivered,omitempty"`
}

func (r *Registry) IssuePersonCredential(ctx context.Context, actor Identity, personID, machine string, days int) (IssuedCredential, error) {
	if !validID(personID, "u_") || !validPersonName(machine) || days < 1 || days > 365 {
		return IssuedCredential{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (IssuedCredential, error) {
		admin, err := r.requireAdmin(ctx, tx, actor)
		if err != nil {
			return IssuedCredential{}, err
		}
		p, err := r.person(ctx, tx, personID)
		if err != nil {
			return IssuedCredential{}, err
		}
		token, who, err := r.issueCredential(ctx, tx, admin.ID, personID, p.Name, machine, time.Duration(days)*24*time.Hour)
		if err != nil {
			return IssuedCredential{}, err
		}
		stored := IssuedCredential{IdentityID: who.ID, Owner: who.Owner, Machine: who.Machine, ExpiresAt: token.ExpiresAt.UTC(), SecretDelivered: true}
		raw, err := json.Marshal(map[string]any{"ok": true, "issued": stored})
		if err != nil {
			return IssuedCredential{}, err
		}
		if err = finishCatalogResult(ctx, tx, string(raw)); err != nil {
			return IssuedCredential{}, err
		}
		stored.SecretDelivered = false
		stored.Secret = token.Secret
		return stored, nil
	})
}

// adminIdentity reads one credential view. Legacy expiry and revocation come
// from the default board token table; a vanished token reads as revoked.
func (r *Registry) adminIdentity(ctx context.Context, tx *sql.Tx, id string, tokens map[string]core.TokenInfo) (AdminIdentity, error) {
	var out AdminIdentity
	var expires sql.NullInt64
	var revoked sql.NullBool
	err := tx.QueryRowContext(ctx, `SELECT i.id,i.person_id,i.owner,i.machine,i.source,i.token_id,c.expires,c.revoked FROM identities i LEFT JOIN credentials c ON c.identity_id=i.id WHERE i.id=?`, id).Scan(&out.ID, &out.PersonID, &out.Owner, &out.Machine, &out.Source, new(string), &expires, &revoked)
	if missing(err) {
		return out, notFound()
	}
	if err != nil {
		return out, err
	}
	if out.Source == "legacy" {
		return r.legacyView(ctx, out, tokens), nil
	}
	if expires.Valid {
		at := time.Unix(0, expires.Int64).UTC()
		out.ExpiresAt = &at
	}
	out.Revoked = !revoked.Valid || revoked.Bool
	return out, nil
}
func (r *Registry) legacyView(ctx context.Context, out AdminIdentity, tokens map[string]core.TokenInfo) AdminIdentity {
	if tokens == nil {
		tokens = r.legacyTokens(ctx)
	}
	token, ok := tokens[out.ID]
	if !ok {
		out.Revoked = true
		return out
	}
	at := token.ExpiresAt.UTC()
	out.ExpiresAt = &at
	out.Revoked = token.Revoked
	return out
}
func (r *Registry) legacyTokens(ctx context.Context) map[string]core.TokenInfo {
	tokens := map[string]core.TokenInfo{}
	list, err := r.options.Default.ListTokens(ctx)
	if err != nil {
		return tokens
	}
	for _, token := range list {
		tokens["legacy_"+token.ID] = token
	}
	return tokens
}
func identityResult(ctx context.Context, tx *sql.Tx, view AdminIdentity) error {
	raw, err := json.Marshal(map[string]any{"ok": true, "identity": view})
	if err != nil {
		return err
	}
	return finishCatalogResult(ctx, tx, string(raw))
}
func (r *Registry) LinkIdentity(ctx context.Context, actor Identity, identityID, personID string) (AdminIdentity, error) {
	if !validIdentityID(identityID) || !validID(personID, "u_") {
		return AdminIdentity{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (AdminIdentity, error) {
		admin, err := r.requireAdmin(ctx, tx, actor)
		if err != nil {
			return AdminIdentity{}, err
		}
		if _, err = r.person(ctx, tx, personID); err != nil {
			return AdminIdentity{}, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE identities SET person_id=? WHERE id=?`, personID, identityID)
		if err != nil {
			return AdminIdentity{}, err
		}
		if n, e := result.RowsAffected(); e != nil || n != 1 {
			return AdminIdentity{}, notFound()
		}
		view, err := r.adminIdentity(ctx, tx, identityID, nil)
		if err != nil {
			return view, err
		}
		if err = adminAudit(ctx, tx, "identity_linked", admin.ID, personID, identityID, ""); err != nil {
			return view, err
		}
		return view, identityResult(ctx, tx, view)
	})
}

// RevokeIdentity revokes a catalog credential in the registry or a legacy
// token on the default board. The board write is not part of the registry
// transaction; a failed commit leaves the token revoked and the replay retries.
func (r *Registry) RevokeIdentity(ctx context.Context, actor Identity, identityID string) (AdminIdentity, error) {
	if !validIdentityID(identityID) {
		return AdminIdentity{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (AdminIdentity, error) {
		admin, err := r.requireAdmin(ctx, tx, actor)
		if err != nil {
			return AdminIdentity{}, err
		}
		var source, tokenID, personID string
		err = tx.QueryRowContext(ctx, `SELECT source,token_id,person_id FROM identities WHERE id=?`, identityID).Scan(&source, &tokenID, &personID)
		if missing(err) {
			return AdminIdentity{}, notFound()
		}
		if err != nil {
			return AdminIdentity{}, err
		}
		if source == "legacy" {
			if err = r.options.Default.RevokeToken(ctx, tokenID); err != nil {
				return AdminIdentity{}, err
			}
			if err = audit(ctx, tx, "credential_revoked", identityID, ""); err != nil {
				return AdminIdentity{}, err
			}
			if err = adminAudit(ctx, tx, "credential_revoked", admin.ID, personID, identityID, "legacy"); err != nil {
				return AdminIdentity{}, err
			}
		} else if err = r.revokeCredential(ctx, tx, admin.ID, identityID); err != nil {
			return AdminIdentity{}, err
		}
		view, err := r.adminIdentity(ctx, tx, identityID, nil)
		if err != nil {
			return view, err
		}
		return view, identityResult(ctx, tx, view)
	})
}
func parseDays(value string) int {
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return days
}
