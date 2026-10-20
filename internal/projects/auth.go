package projects

import (
	"agent-board/internal/core"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

func random(prefix string, bytes int) (string, error) {
	b := make([]byte, bytes)
	_, err := rand.Read(b)
	return prefix + hex.EncodeToString(b), err
}
func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func validID(value, prefix string) bool {
	return regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `[0-9a-f]{32}$`).MatchString(value)
}
func (r *Registry) Authenticate(ctx context.Context, secret string) (Identity, error) {
	if strings.HasPrefix(secret, "ac_") {
		var id string
		err := r.db.QueryRowContext(ctx, `SELECT identity_id FROM credentials WHERE hash=?`, hash(secret)).Scan(&id)
		if err != nil {
			return Identity{}, core.Fail("unauthorized", "invalid or expired credential")
		}
		return transaction(r, ctx, func(tx *sql.Tx) (Identity, error) { return r.identity(ctx, tx, id) })
	}
	p, err := r.options.Default.Authenticate(ctx, secret)
	if err != nil {
		return Identity{}, core.Fail("unauthorized", "invalid or expired credential")
	}
	_, until, err := r.options.Default.AuthenticateID(ctx, p.ID)
	if err != nil {
		return Identity{}, err
	}
	return r.importLegacy(ctx, p, until)
}
func (r *Registry) importLegacy(ctx context.Context, p core.Principal, until time.Time) (Identity, error) {
	who := Identity{ID: "legacy_" + p.ID, Source: "legacy", TokenID: p.ID, Owner: p.Owner, Machine: p.Machine, Expires: until}
	return transaction(r, ctx, func(tx *sql.Tx) (Identity, error) {
		_, err := tx.ExecContext(ctx, `INSERT INTO identities(id,source,token_id,owner,machine) VALUES(?,'legacy',?,?,?) ON CONFLICT(id) DO NOTHING`, who.ID, p.ID, p.Owner, p.Machine)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO memberships(project_id,identity_id,role,local_id,active,version,access_epoch) VALUES(?,?,'editor',?,1,1,0) ON CONFLICT(project_id,identity_id) DO NOTHING`, r.defaultID, who.ID, p.ID)
		}
		return who, err
	})
}

// CreateCredential is an offline administration operation. Only the caller's
// private output writer may receive Secret; its value is never database state.
func (r *Registry) CreateCredential(ctx context.Context, owner, machine string, ttl time.Duration) (core.TokenView, Identity, error) {
	var token core.TokenView
	var who Identity
	if strings.TrimSpace(owner) == "" || len(owner) > 128 || strings.TrimSpace(machine) == "" || len(machine) > 128 || ttl < time.Second || ttl > 365*24*time.Hour {
		return token, who, invalid()
	}
	id, err := random("i_", 16)
	if err != nil {
		return token, who, err
	}
	secret, err := random("ac_", 32)
	if err != nil {
		return token, who, err
	}
	who = Identity{ID: id, Source: "catalog", Owner: owner, Machine: machine, Expires: time.Now().Add(ttl)}
	_, err = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		_, e := tx.ExecContext(ctx, `INSERT INTO identities(id,source,token_id,owner,machine) VALUES(?,'catalog','',?,?)`, id, owner, machine)
		if e == nil {
			_, e = tx.ExecContext(ctx, `INSERT INTO credentials(identity_id,hash,expires) VALUES(?,?,?)`, id, hash(secret), who.Expires.UnixNano())
		}
		return true, e
	})
	token = core.TokenView{ID: id, Secret: secret, ExpiresAt: who.Expires}
	return token, who, err
}
func (r *Registry) BootstrapDefaultOwner(ctx context.Context, legacyID string) error {
	p, until, err := r.options.Default.AuthenticateID(ctx, legacyID)
	if err != nil {
		return err
	}
	who, err := r.importLegacy(ctx, p, until)
	if err != nil {
		return err
	}
	_, err = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		var count int
		if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memberships WHERE project_id=? AND role='owner' AND active=1`, r.defaultID).Scan(&count); e != nil {
			return false, e
		}
		if count != 0 {
			var same int
			if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memberships WHERE project_id=? AND identity_id=? AND active=1 AND role='owner'`, r.defaultID, who.ID).Scan(&same); e != nil {
				return false, e
			}
			if same == 1 {
				return true, nil
			}
			return false, core.Fail("conflict", "default project already has an owner")
		}
		_, e := tx.ExecContext(ctx, `UPDATE memberships SET role='owner',version=version+1 WHERE project_id=? AND identity_id=? AND active=1`, r.defaultID, who.ID)
		return true, e
	})
	return err
}
func missing(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// RevokeCredential is local operator administration; every subsequent registry
// and project check observes the revoked source credential before data/replay.
func (r *Registry) RevokeCredential(ctx context.Context, id string) error {
	_, err := transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		result, e := tx.ExecContext(ctx, `UPDATE credentials SET revoked=1 WHERE identity_id=?`, id)
		if e != nil {
			return false, e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return false, e
		}
		if n != 1 {
			return false, notFound()
		}
		return true, audit(ctx, tx, "credential_revoked", id, "")
	})
	return err
}
