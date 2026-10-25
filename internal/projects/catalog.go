package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func (r *Registry) List(ctx context.Context, who Identity) ([]Project, error) {
	return transaction(r, ctx, func(tx *sql.Tx) ([]Project, error) {
		if _, err := r.identity(ctx, tx, who.ID); err != nil {
			return nil, err
		}
		rows, err := tx.QueryContext(ctx, `SELECT p.id FROM projects p JOIN memberships m ON m.project_id=p.id WHERE m.identity_id=? AND m.active=1 AND p.state<>'provisioning' ORDER BY p.created,p.id`, who.ID)
		if err != nil {
			return nil, err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		out := []Project{}
		for _, id := range ids {
			a, e := r.access(ctx, tx, who, id, false)
			if e != nil {
				return nil, e
			}
			out = append(out, a.project)
		}
		return out, nil
	})
}

// validName keeps project names printable: valid UTF-8 and no Unicode control
// characters, so a name cannot inject terminal escapes into CLI/log output.
func validName(name string) bool {
	return strings.TrimSpace(name) == name && name != "" && len(name) <= 120 && utf8.ValidString(name) && strings.IndexFunc(name, unicode.IsControl) < 0
}
func validRole(role string) bool { return role == "owner" || role == "editor" || role == "viewer" }
func requireOwner(a access) error {
	if !a.project.CanManage {
		return denied()
	}
	return nil
}

// requireProjectOwner is for project settings beyond membership: a global
// admin must hold an owner membership to rename, archive or edit repositories.
func requireProjectOwner(a access) error {
	if a.adminOnly {
		return denied()
	}
	return requireOwner(a)
}
func (r *Registry) createProject(ctx context.Context, who Identity, id, name, url, branch string) (Project, error) {
	if !validName(name) {
		return Project{}, invalid()
	}
	_, err := transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		if _, e := r.identity(ctx, tx, who.ID); e != nil {
			return false, e
		}
		var rejected int
		if e := tx.QueryRowContext(ctx, `SELECT 1 FROM rejected_provisions WHERE id=?`, id).Scan(&rejected); e == nil {
			return false, core.Fail("conflict", "project provisioning was rejected")
		} else if !missing(e) {
			return false, e
		}
		if e := r.quarantineRejected(id); e != nil {
			return false, e
		}
		var state, creator string
		e := tx.QueryRowContext(ctx, `SELECT state,creator FROM projects WHERE id=?`, id).Scan(&state, &creator)
		if e == nil {
			if creator != who.ID {
				return false, denied()
			}
			return true, nil
		}
		if !missing(e) {
			return false, e
		}
		var all, owned, pending int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(creator=?),0),COALESCE(SUM(state='provisioning'),0) FROM projects`, who.ID).Scan(&all, &owned, &pending); e != nil {
			return false, e
		}
		if all >= MaxProjects || owned >= MaxOwnedProjects || pending >= MaxProvisions {
			return false, core.Fail("conflict", "project capacity reached")
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO projects(id,name,state,version,creator,created) VALUES(?,?,'provisioning',1,?,?)`, id, name, who.ID, time.Now().UnixNano())
		return true, e
	})
	if err != nil {
		return Project{}, err
	}
	v, err := r.provision(id)
	if err != nil {
		return Project{}, err
	}
	if err = r.syncProvision(id); err != nil {
		return Project{}, err
	}
	return transaction(r, ctx, func(tx *sql.Tx) (Project, error) {
		if e := r.quarantineRejected(id); e != nil {
			return Project{}, e
		}
		current, e := r.identity(ctx, tx, who.ID)
		if e != nil {
			return Project{}, e
		}
		var state string
		if e = tx.QueryRowContext(ctx, `SELECT state FROM projects WHERE id=? AND creator=?`, id, who.ID).Scan(&state); e != nil {
			return Project{}, e
		}
		if state == "provisioning" {
			p, e := r.localPrincipal(ctx, v, current)
			if e != nil {
				return Project{}, e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO memberships(project_id,identity_id,role,local_id,active,version,access_epoch) VALUES(?,?,'owner',?,1,1,1)`, id, who.ID, p.ID)
			if e != nil {
				return Project{}, e
			}
			if url != "" {
				rid, _ := random("r_", 16)
				if _, e = tx.ExecContext(ctx, `INSERT INTO repositories VALUES(?,?,?,?,1)`, rid, id, url, branch); e != nil {
					return Project{}, e
				}
			}
			if _, e = tx.ExecContext(ctx, `UPDATE projects SET state='active' WHERE id=?`, id); e != nil {
				return Project{}, e
			}
			if e = audit(ctx, tx, "project_created", who.ID, id); e != nil {
				return Project{}, e
			}
		}
		a, e := r.access(ctx, tx, who, id, false)
		if e == nil {
			e = finishCatalog(ctx, tx)
		}
		return a.project, e
	})
}
func (r *Registry) updateProject(ctx context.Context, who Identity, id, name, state string, version int64) (Project, error) {
	if state != "active" && state != "archived" || id == r.defaultID && state == "archived" {
		return Project{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (Project, error) {
		a, e := r.manageAccess(ctx, tx, who, id)
		if e != nil {
			return Project{}, e
		}
		// Only a new name must satisfy the current rules: a project with a
		// legacy name can still be archived or restored unchanged.
		if name != a.project.Name && !validName(name) {
			return Project{}, invalid()
		}
		if e = requireProjectOwner(a); e != nil {
			return Project{}, e
		}
		if a.project.Version != version {
			return Project{}, core.Fail("conflict", "project changed")
		}
		if state == "archived" && a.project.State != "archived" {
			v, e := r.runtime(id)
			if e != nil {
				return Project{}, e
			}
			if e = v.store.CheckNoLiveLeases(ctx); e != nil {
				return Project{}, e
			}
		}
		_, e = tx.ExecContext(ctx, `UPDATE projects SET name=?,state=?,version=version+1 WHERE id=?`, name, state, id)
		if e != nil {
			return Project{}, e
		}
		a.project.Name = name
		a.project.State = state
		a.project.Version++
		a.project.CanWrite = state == "active"
		a.project.CanArchive = state == "active" && id != r.defaultID
		e = audit(ctx, tx, "project_updated", who.ID, id)
		if e == nil {
			e = finishCatalog(ctx, tx)
		}
		return a.project, e
	})
}

// A rename is durable before the rejection SQL commit. Treat its recovery set
// as rejection evidence too, so reboot cannot revive the old pending action.
func (r *Registry) quarantineRejected(id string) error {
	_, err := os.Lstat(filepath.Join(r.options.DataDir, "quarantine", id))
	if err == nil {
		return core.Fail("conflict", "project provisioning was rejected")
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
