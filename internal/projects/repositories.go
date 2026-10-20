package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"net/url"
	"strings"
)

// Validate before binding durable action payloads: credentials must never enter
// the registry, audit log, error response or saved form draft.
func repositoryFields(raw, branch string) (string, string, error) {
	if len(raw) > 2048 || len(branch) > 200 || strings.ContainsAny(raw+branch, "\r\n\x00") || strings.TrimSpace(branch) != branch {
		return "", "", invalid()
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path == "" || strings.Contains(u.Path, "\\") {
		return "", "", invalid()
	}
	switch u.Scheme {
	case "https":
		if u.User != nil {
			return "", "", invalid()
		}
	case "ssh":
		if u.User == nil || u.User.Username() != "git" {
			return "", "", invalid()
		}
		if _, ok := u.User.Password(); ok {
			return "", "", invalid()
		}
	default:
		return "", "", invalid()
	}
	if strings.Contains(branch, "..") || strings.ContainsAny(branch, " ~^:?*[\\") || strings.HasPrefix(branch, "-") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") {
		return "", "", invalid()
	}
	return u.String(), branch, nil
}
func (r *Registry) repository(ctx context.Context, who Identity, pid, rid, operation, url, branch string, version int64) error {
	_, err := transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		a, e := r.access(ctx, tx, who, pid, false)
		if e != nil {
			return false, e
		}
		if e = requireOwner(a); e != nil {
			return false, e
		}
		switch operation {
		case "add":
			var count int
			if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM repositories WHERE project_id=?`, pid).Scan(&count); e != nil {
				return false, e
			}
			if count >= MaxRepositories {
				return false, core.Fail("conflict", "repository capacity reached")
			}
			rid, e = random("r_", 16)
			if e != nil {
				return false, e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO repositories VALUES(?,?,?,?,1)`, rid, pid, url, branch)
		case "update", "remove":
			var current int64
			e = tx.QueryRowContext(ctx, `SELECT version FROM repositories WHERE id=? AND project_id=?`, rid, pid).Scan(&current)
			if missing(e) {
				return false, notFound()
			}
			if e != nil {
				return false, e
			}
			if version != current {
				return false, core.Fail("conflict", "repository changed")
			}
			if operation == "remove" {
				_, e = tx.ExecContext(ctx, `DELETE FROM repositories WHERE id=? AND project_id=?`, rid, pid)
			} else {
				_, e = tx.ExecContext(ctx, `UPDATE repositories SET url=?,branch=?,version=version+1 WHERE id=? AND project_id=?`, url, branch, rid, pid)
			}
		default:
			return false, invalid()
		}
		if e != nil {
			return false, e
		}
		if e = audit(ctx, tx, "repository_"+operation, who.ID, pid); e != nil {
			return false, e
		}
		return true, finishCatalog(ctx, tx)
	})
	return err
}
