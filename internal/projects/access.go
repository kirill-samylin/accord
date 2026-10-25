package projects

import (
	"agent-board/internal/core"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

func (r *Registry) project(ctx context.Context, tx *sql.Tx, id string) (Project, error) {
	var p Project
	err := tx.QueryRowContext(ctx, `SELECT id,name,state,version FROM projects WHERE id=? AND state<>'provisioning'`, id).Scan(&p.ID, &p.Name, &p.State, &p.Version)
	if missing(err) {
		return p, notFound()
	}
	if err != nil {
		return p, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,url,branch,version FROM repositories WHERE project_id=? ORDER BY id`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	p.Repositories = []Repository{}
	for rows.Next() {
		var repo Repository
		if err = rows.Scan(&repo.ID, &repo.URL, &repo.Branch, &repo.Version); err != nil {
			return p, err
		}
		p.Repositories = append(p.Repositories, repo)
	}
	return p, rows.Err()
}
func (r *Registry) access(ctx context.Context, tx *sql.Tx, who Identity, pid string, write bool) (access, error) {
	var out access
	current, err := r.identity(ctx, tx, who.ID)
	if err != nil {
		return out, err
	}
	p, err := r.project(ctx, tx, pid)
	if err != nil {
		return out, err
	}
	m := member{}
	err = tx.QueryRowContext(ctx, `SELECT role,local_id,active,version,access_epoch FROM memberships WHERE project_id=? AND identity_id=?`, pid, who.ID).Scan(&m.Role, &m.LocalID, &m.Active, &m.Version, &m.Epoch)
	if missing(err) || err == nil && !m.Active {
		return out, notFound()
	}
	if err != nil {
		return out, err
	}
	p.Role = m.Role
	// Global admins manage every project they belong to; board writes still
	// require an editor/owner membership (docs/admin.md).
	p.CanManage = m.Role == "owner" || current.Admin
	p.CanArchive = p.CanManage && p.State == "active" && p.ID != r.defaultID
	p.CanWrite = p.State == "active" && (m.Role == "owner" || m.Role == "editor")
	if write && !p.CanWrite {
		return out, denied()
	}
	out = access{tx: tx, identity: current, project: p, member: m, principal: core.Principal{ID: m.LocalID, Owner: current.Owner, Machine: current.Machine}}
	return out, nil
}

// manageAccess is the catalog management view: an active member, or a global
// admin without membership. The admin variant carries no board principal and
// no session epoch, so it can never open, read or write the project board.
func (r *Registry) manageAccess(ctx context.Context, tx *sql.Tx, who Identity, pid string) (access, error) {
	a, err := r.access(ctx, tx, who, pid, false)
	var domain *core.Error
	if err == nil || !errors.As(err, &domain) || domain.Code != "NOT_FOUND" {
		return a, err
	}
	current, e := r.identity(ctx, tx, who.ID)
	if e != nil {
		return a, e
	}
	if !current.Admin {
		return a, err
	}
	p, e := r.project(ctx, tx, pid)
	if e != nil {
		return a, e
	}
	p.CanManage = true
	p.CanArchive = p.State == "active" && p.ID != r.defaultID
	return access{tx: tx, identity: current, project: p, principal: core.Principal{Owner: current.Owner, Machine: current.Machine}}, nil
}
func withAccess[T any](s *service, ctx context.Context, write bool, session string, fn func(context.Context, access) (T, error)) (T, error) {
	var zero T
	who, err := accountFrom(ctx)
	if err != nil {
		return zero, err
	}
	return transaction(s.registry, ctx, func(tx *sql.Tx) (T, error) {
		access, err := s.registry.access(ctx, tx, who, s.projectID, write)
		if err != nil {
			return zero, err
		}
		if session != "" {
			if err = s.registry.sessionAllowed(ctx, tx, access, session); err != nil {
				return zero, err
			}
		}
		invoke := func() (T, error) { return fn(context.WithValue(ctx, accessKey{}, access), access) }
		if access.identity.Source == "legacy" && s.projectID != s.registry.defaultID {
			var result T
			err = s.registry.options.Default.GuardCredential(ctx, core.Principal{ID: access.identity.TokenID, Owner: access.identity.Owner, Machine: access.identity.Machine}, func() error { var e error; result, e = invoke(); return e })
			return result, err
		}
		return invoke()
	})
}

type accessKey struct{}

func (r *Registry) sessionAllowed(ctx context.Context, tx *sql.Tx, a access, session string) error {
	var epoch int64
	var identity string
	err := tx.QueryRowContext(ctx, `SELECT identity_id,access_epoch FROM session_epochs WHERE project_id=? AND session_id=?`, a.project.ID, session).Scan(&identity, &epoch)
	if missing(err) && a.project.ID == r.defaultID && a.identity.Source == "legacy" && a.member.Epoch == 0 {
		return nil
	}
	if err != nil || identity != a.identity.ID || epoch != a.member.Epoch {
		return core.Fail("unauthorized", "project session must be reopened")
	}
	return nil
}
func (r *Registry) describeID(a access, id string) string {
	if a.member.Epoch == 0 && a.project.ID == r.defaultID && a.identity.Source == "legacy" {
		return id
	}
	mac := hmac.New(sha256.New, r.signingKey)
	fmt.Fprintf(mac, "describe\x00%s\x00%s\x00%d\x00%s", a.project.ID, a.identity.ID, a.member.Epoch, id)
	return hex.EncodeToString(mac.Sum(nil))
}
func audit(ctx context.Context, tx *sql.Tx, kind, identity, project string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit(event,identity_id,project_id,at) VALUES(?,?,?,?)`, kind, identity, project, time.Now().UnixNano())
	return err
}
