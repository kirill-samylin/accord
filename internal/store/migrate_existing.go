package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"net/url"
)

// MigrateExisting performs DDL only. The operator must hold its maintenance
// lock with every writer stopped; no listener or board transaction is opened.
func MigrateExisting(ctx context.Context, path string) error {
	absolute, err := existingDatabase(path)
	if err != nil {
		return err
	}
	target := url.URL{Scheme: "file", Path: absolute, RawQuery: "mode=rw"}
	db, err := sql.Open("sqlite", target.String())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err != nil {
		return err
	}
	var version int
	if err = db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version < 1 || version > SchemaVersion {
		return core.Fail("validation", "existing supported board required")
	}
	if err = validateSchema(ctx, db, version); err != nil {
		return err
	}
	return migrate(ctx, db, absolute)
}
