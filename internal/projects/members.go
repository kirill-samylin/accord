package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
)

func (r *Registry) members(ctx context.Context, who Identity, pid string) ([]Member, error) {
	return transaction(r, ctx, func(tx *sql.Tx) ([]Member, error) {
		a, e := r.access(ctx, tx, who, pid, false)
		if e != nil {
			return nil, e
		}
		if e = requireOwner(a); e != nil {
			return nil, e
		}
		rows, e := tx.QueryContext(ctx, `SELECT m.identity_id,i.owner,i.machine,m.role,m.version FROM memberships m JOIN identities i ON i.id=m.identity_id WHERE m.project_id=? AND m.active=1 ORDER BY m.identity_id`, pid)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		out := []Member{}
		for rows.Next() {
			var m Member
			if e = rows.Scan(&m.IdentityID, &m.Owner, &m.Machine, &m.Role, &m.Version); e != nil {
				return nil, e
			}
			out = append(out, m)
		}
		return out, rows.Err()
	})
}
func (r *Registry) changeMember(ctx context.Context, who Identity, pid, target, role, operation string, version int64) error {
	if operation != "revoke" && !validRole(role) {
		return invalid()
	}
	_, err := transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		a, e := r.access(ctx, tx, who, pid, false)
		if e != nil {
			return false, e
		}
		if e = requireOwner(a); e != nil {
			return false, e
		}
		v, e := r.runtime(pid)
		if e != nil {
			return false, e
		}
		var oldRole, local string
		var active bool
		var oldVersion, epoch int64
		e = tx.QueryRowContext(ctx, `SELECT role,local_id,active,version,access_epoch FROM memberships WHERE project_id=? AND identity_id=?`, pid, target).Scan(&oldRole, &local, &active, &oldVersion, &epoch)
		absent := missing(e)
		if e != nil && !absent {
			return false, e
		}
		if operation != "grant" && (absent || !active) {
			return false, notFound()
		}
		if operation == "grant" && active {
			return false, core.Fail("conflict", "membership already exists")
		}
		if operation != "grant" && !absent && oldVersion != version {
			return false, core.Fail("conflict", "membership changed")
		}
		if active && oldRole == "owner" && (operation == "revoke" || role != "owner") {
			var owners int
			if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memberships WHERE project_id=? AND role='owner' AND active=1`, pid).Scan(&owners); e != nil {
				return false, e
			}
			if owners <= 1 {
				return false, core.Fail("conflict", "last owner must remain")
			}
		}
		if operation == "revoke" {
			_, e = tx.ExecContext(ctx, `UPDATE memberships SET active=0,version=version+1,access_epoch=access_epoch+1 WHERE project_id=? AND identity_id=?`, pid, target)
		} else if operation == "set" {
			_, e = tx.ExecContext(ctx, `UPDATE memberships SET role=?,version=version+1 WHERE project_id=? AND identity_id=?`, role, pid, target)
		} else if operation == "grant" {
			current, identityErr := r.identity(ctx, tx, target)
			if identityErr != nil {
				return false, notFound()
			}
			var count int
			if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memberships WHERE project_id=? AND active=1`, pid).Scan(&count); e != nil {
				return false, e
			}
			if count >= MaxMembers {
				return false, core.Fail("conflict", "member capacity reached")
			}
			if absent {
				p, principalErr := r.localPrincipal(ctx, v, current)
				if principalErr != nil {
					return false, principalErr
				}
				local = p.ID
				_, e = tx.ExecContext(ctx, `INSERT INTO memberships VALUES(?,?,?,?,1,1,1)`, pid, target, role, local)
			} else {
				_, e = tx.ExecContext(ctx, `UPDATE memberships SET active=1,role=?,version=version+1 WHERE project_id=? AND identity_id=?`, role, pid, target)
			}
		} else {
			return false, invalid()
		}
		if e != nil {
			return false, e
		}
		if e = audit(ctx, tx, "member_"+operation, who.ID, pid); e != nil {
			return false, e
		}
		return true, finishCatalog(ctx, tx)
	})
	return err
}
