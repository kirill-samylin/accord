package projects

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// quarantinePending makes even a missing pending directory an explicit durable
// recovery set. Call only offline; no private bytes are deleted or made usable.
func quarantinePending(root, id string) error {
	if !validID(id, "p_") {
		return errors.New("invalid pending project ID")
	}
	source := filepath.Join(root, "projects", id)
	quarantine := filepath.Join(root, "quarantine")
	target := filepath.Join(quarantine, id)
	if _, err := privateRoot(quarantine); err != nil {
		return err
	}
	entries, err := os.ReadDir(quarantine)
	if err != nil {
		return err
	}
	exists := false
	for _, entry := range entries {
		if !validID(entry.Name(), "p_") || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("unsafe quarantine inventory")
		}
		if entry.Name() == id {
			exists = true
		}
	}
	if !exists && len(entries) >= 128 {
		return errors.New("quarantine capacity reached")
	}
	info, err := os.Lstat(source)
	switch {
	case err == nil:
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 || exists {
			return errors.New("unsafe or conflicting pending storage")
		}
		if err = os.Rename(source, target); err != nil {
			return err
		}
	case os.IsNotExist(err):
		if _, err = privateRoot(target); err != nil {
			return err
		}
	default:
		return err
	}
	for _, dir := range []string{target, quarantine, filepath.Join(root, "projects"), root} {
		file, e := os.Open(dir)
		if os.IsNotExist(e) && dir == filepath.Join(root, "projects") {
			continue
		}
		if e != nil {
			return e
		}
		e = file.Sync()
		file.Close()
		if e != nil {
			return e
		}
	}
	return nil
}

// RejectProvisioning is an explicit offline operator recovery. A durable ID
// tombstone defeats old still-live HMAC create actions after the row is removed.
func RejectProvisioning(ctx context.Context, root, id string) error {
	if !validID(id, "p_") {
		return invalid()
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	for _, name := range []string{"registry.sqlite", "MULTIPROJECT_REQUIRES_CAPABILITY"} {
		info, e := os.Lstat(filepath.Join(absolute, name))
		if e != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			return errors.New("existing private registry and capability marker required")
		}
	}
	absolute, err = privateRoot(root)
	if err != nil {
		return err
	}
	db, err := openRegistry(absolute)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE registry_meta SET default_id=default_id WHERE id=1`); err != nil {
		return err
	}
	var defaultID string
	if err = tx.QueryRowContext(ctx, `SELECT default_id FROM registry_meta WHERE id=1`).Scan(&defaultID); err != nil {
		return err
	}
	if id == defaultID {
		return denied()
	}
	var state string
	err = tx.QueryRowContext(ctx, `SELECT state FROM projects WHERE id=?`, id).Scan(&state)
	if missing(err) {
		var rejected int
		e := tx.QueryRowContext(ctx, `SELECT 1 FROM rejected_provisions WHERE id=?`, id).Scan(&rejected)
		if e == nil {
			return tx.Commit()
		}
		return notFound()
	}
	if err != nil {
		return err
	}
	if state != "provisioning" {
		return denied()
	}
	if err = quarantinePending(absolute, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO rejected_provisions(id,at) VALUES(?,?) ON CONFLICT(id) DO NOTHING`, id, time.Now().UnixNano()); err != nil {
		return err
	}
	for _, statement := range []string{`DELETE FROM memberships WHERE project_id=?`, `DELETE FROM repositories WHERE project_id=?`, `DELETE FROM projects WHERE id=?`} {
		if _, err = tx.ExecContext(ctx, statement, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
