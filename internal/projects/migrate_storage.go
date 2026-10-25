package projects

import (
	"agent-board/internal/store"
	"context"
	"errors"
	"os"
	"path/filepath"
)

// MigrateStorage changes existing published boards only. A complete read-only
// preflight precedes the first DDL; interrupted multi-board work resumes forward.
// It neither opens browser storage nor resets credentials, sessions or history.
func MigrateStorage(ctx context.Context, root, defaultDB, defaultWeb string) (int, error) {
	if root == "" {
		return 0, errors.New("existing registry required")
	}
	marker := filepath.Join(root, "MULTIPROJECT_REQUIRES_CAPABILITY")
	if info, err := os.Lstat(marker); err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return 0, errors.New("existing activation marker required")
	}
	if err := Preflight(root, defaultDB, defaultWeb); err != nil {
		return 0, err
	}
	registry, err := readPrivateSQLite(filepath.Join(root, "registry.sqlite"))
	if err != nil {
		return 0, err
	}
	var defaultID string
	if err = registry.QueryRowContext(ctx, `SELECT default_id FROM registry_meta WHERE id=1`).Scan(&defaultID); err != nil {
		registry.Close()
		return 0, err
	}
	rows, err := registry.QueryContext(ctx, `SELECT id FROM projects WHERE state IN ('active','archived') ORDER BY id`)
	if err != nil {
		registry.Close()
		return 0, err
	}
	paths := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		path := filepath.Join(root, "projects", id, "board.db")
		if id == defaultID {
			path = defaultDB
		}
		paths = append(paths, path)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	registry.Close()
	if err != nil {
		return 0, err
	}
	// The registry itself migrates 1→2 through the same copying open path, so
	// the capable upgrade re-inventories a schema2 authority afterwards.
	absolute, err := privateRoot(root)
	if err != nil {
		return 0, err
	}
	migrated, err := openRegistry(absolute)
	if err != nil {
		return 0, err
	}
	if err = migrated.Close(); err != nil {
		return 0, err
	}
	for _, path := range paths {
		if err = store.MigrateExisting(ctx, path); err != nil {
			return 0, err
		}
	}
	return len(paths), nil
}
