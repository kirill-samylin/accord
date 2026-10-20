package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/web"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
)

// ResetOffline is for a fully staged restore with EVERY server stopped. It
// revokes all restored credentials and sessions, preserves board history and
// repository associations, and binds the restored registry to its new paths.
// Failure leaves the staging directory unsuitable for serving; retry offline.
func ResetOffline(ctx context.Context, root, defaultDB, defaultWeb string) error {
	absolute, err := privateRoot(root)
	if err != nil {
		return err
	}
	if info, e := os.Lstat(filepath.Join(absolute, "MULTIPROJECT_REQUIRES_CAPABILITY")); e != nil || !info.Mode().IsRegular() {
		return errors.New("existing capability marker required")
	}
	db, err := openRegistry(absolute)
	if err != nil {
		return err
	}
	defer db.Close()
	var defaultID string
	if err = db.QueryRowContext(ctx, `SELECT default_id FROM registry_meta WHERE id=1`).Scan(&defaultID); err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, `SELECT id,state FROM projects ORDER BY id`)
	if err != nil {
		return err
	}
	type item struct{ id, state string }
	items := []item{}
	for rows.Next() {
		var entry item
		if err = rows.Scan(&entry.id, &entry.state); err != nil {
			rows.Close()
			return err
		}
		items = append(items, entry)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, entry := range items {
		if !validID(entry.id, "p_") {
			return errors.New("invalid project inventory")
		}
		if entry.state == "provisioning" {
			if err = quarantinePending(absolute, entry.id); err != nil {
				return err
			}
			continue
		}
		if !validID(entry.id, "p_") {
			return errors.New("invalid project inventory")
		}
		boardPath := filepath.Join(absolute, "projects", entry.id, "board.db")
		webPath := filepath.Join(absolute, "projects", entry.id, "web")
		if entry.id == defaultID {
			boardPath = defaultDB
			webPath = defaultWeb
		}
		if _, e := os.Lstat(boardPath); os.IsNotExist(e) && entry.state == "provisioning" {
			continue
		}
		board, e := store.OpenExisting(boardPath, core.Config{Project: "restored"}, nil)
		if e != nil {
			return e
		}
		_, e = board.RecoverState(ctx, "restore")
		board.Close()
		if e != nil {
			return e
		}
		if _, e = os.Lstat(filepath.Join(webPath, "web.db")); os.IsNotExist(e) && entry.state == "provisioning" {
			continue
		}
		if e = web.ResetOffline(ctx, webPath); e != nil {
			return e
		}
	}
	dbPath, err := filepath.Abs(defaultDB)
	if err != nil {
		return err
	}
	webPath, err := filepath.Abs(defaultWeb)
	if err != nil {
		return err
	}
	key, err := random("", 32)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE credentials SET revoked=1; DELETE FROM sessions; DELETE FROM preauth; DELETE FROM actions; DELETE FROM session_epochs; UPDATE memberships SET access_epoch=access_epoch+1; INSERT INTO rejected_provisions(id,at) SELECT id,0 FROM projects WHERE state='provisioning' ON CONFLICT(id) DO NOTHING; DELETE FROM memberships WHERE project_id IN (SELECT id FROM projects WHERE state='provisioning'); DELETE FROM repositories WHERE project_id IN (SELECT id FROM projects WHERE state='provisioning'); DELETE FROM projects WHERE state='provisioning';`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE registry_meta SET default_db=?,default_web=?,signing_key=? WHERE id=1`, dbPath, webPath, key); err != nil {
		return err
	}
	return tx.Commit()
}

// RecoverOwner is an explicitly trusted local administrator operation, used
// after restore or loss of the last live owner credential. It grants only one
// named identity in one named project and never issues a universal bearer.
func (r *Registry) RecoverOwner(ctx context.Context, pid, identity string) error {
	v, err := r.runtime(pid)
	if err != nil {
		return err
	}
	_, err = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		if _, e := r.project(ctx, tx, pid); e != nil {
			return false, e
		}
		who, e := r.identity(ctx, tx, identity)
		if e != nil {
			return false, e
		}
		var local string
		e = tx.QueryRowContext(ctx, `SELECT local_id FROM memberships WHERE project_id=? AND identity_id=?`, pid, identity).Scan(&local)
		if missing(e) {
			p, e := r.localPrincipal(ctx, v, who)
			if e != nil {
				return false, e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO memberships VALUES(?,?,'owner',?,1,1,1)`, pid, identity, p.ID)
			if e != nil {
				return false, e
			}
		} else if e != nil {
			return false, e
		} else {
			_, e = tx.ExecContext(ctx, `UPDATE memberships SET role='owner',active=1,version=version+1,access_epoch=access_epoch+1 WHERE project_id=? AND identity_id=?`, pid, identity)
			if e != nil {
				return false, e
			}
		}
		return true, audit(ctx, tx, "offline_owner_recovery", identity, pid)
	})
	return err
}
