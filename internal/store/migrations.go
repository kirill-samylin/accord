package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const SchemaVersion = 3

// migrate is the sole DDL owner. A schema1 snapshot is published before any
// schema2 transaction; unsupported future versions are rejected before DDL.
func migrate(ctx context.Context, db *sql.DB, path string, cfg core.Config) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("database schema is newer than this server")
	}
	if version == SchemaVersion {
		return nil
	}
	if version == 1 || version == 2 {
		dir := path + ".migrations"
		if err := os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode().Perm() != 0700 {
			return fmt.Errorf("migration backup directory must be regular mode 0700")
		}
		id, err := identifier()
		if err != nil {
			return err
		}
		if err = syncDirectory(filepath.Dir(dir)); err != nil {
			return fmt.Errorf("persist migration backup directory: %w", err)
		}
		if err = migrationBackup(ctx, path, filepath.Join(dir, fmt.Sprintf("schema%d-%s.db", version, id))); err != nil {
			return fmt.Errorf("pre-migration backup: %w", err)
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// This write serializes two starting processes before either can apply DDL.
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metadata (id INTEGER PRIMARY KEY CHECK(id=1),sequence INTEGER NOT NULL)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata VALUES(1,0)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE metadata SET sequence=sequence WHERE id=1`); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == SchemaVersion {
		return nil
	}
	if version != 0 && version != 1 && version != 2 {
		return fmt.Errorf("unsupported migration source")
	}
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS tasks (key TEXT PRIMARY KEY,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS sessions (key TEXT PRIMARY KEY,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS role_generations (role TEXT PRIMARY KEY,generation INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS events (cursor INTEGER PRIMARY KEY AUTOINCREMENT,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS idempotency (key TEXT PRIMARY KEY,hash TEXT NOT NULL,response TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS tokens (id TEXT PRIMARY KEY,hash TEXT UNIQUE NOT NULL,owner TEXT NOT NULL,machine TEXT NOT NULL,expires INTEGER NOT NULL,revoked INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
	}
	if version < 2 {
		for _, q := range []string{`CREATE TABLE notes (key TEXT PRIMARY KEY,data TEXT NOT NULL)`, `CREATE TABLE note_metadata (id INTEGER PRIMARY KEY CHECK(id=1),sequence INTEGER NOT NULL)`, `INSERT INTO note_metadata VALUES(1,0)`} {
			if _, err = tx.ExecContext(ctx, q); err != nil {
				return err
			}
		}
	}
	for _, q := range []string{
		`CREATE TABLE config_history (version INTEGER PRIMARY KEY,sha256 TEXT NOT NULL,canonical_json TEXT NOT NULL,raw_yaml TEXT NOT NULL,actor TEXT NOT NULL,applied_at TEXT NOT NULL)`,
		`CREATE TABLE config_active (id INTEGER PRIMARY KEY CHECK(id=1),version INTEGER NOT NULL REFERENCES config_history(version))`,
		`ALTER TABLE tokens ADD COLUMN roles TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE tokens ADD COLUMN scopes_version INTEGER NOT NULL DEFAULT 1`,
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	if version == 0 && cfg.SkipBootstrap {
		if _, err = tx.ExecContext(ctx, `PRAGMA user_version=3`); err != nil {
			return err
		}
		return tx.Commit()
	}
	snapshot, err := bootstrapCompatibility(ctx, tx, cfg)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `INSERT INTO config_history VALUES(1,?,?,?,?,?)`, snapshot.SHA256(), string(snapshot.CanonicalJSON()), string(snapshot.CanonicalJSON()), "bootstrap_compat", now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO config_active VALUES(1,1)`); err != nil {
		return err
	}
	scopes, err := json.Marshal(legacyRoles)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE tokens SET roles=?`, string(scopes)); err != nil {
		return err
	}
	var high int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cursor),0) FROM events`).Scan(&high); err != nil {
		return err
	}
	event := core.Event{Cursor: high + 1, Type: "migration_legacy_scopes", Reason: string(scopes), At: now}
	eventRaw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO events(cursor,data) VALUES(?,?)`, event.Cursor, string(eventRaw)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `PRAGMA user_version=3`); err != nil {
		return err
	}
	return tx.Commit()
}

// Keep an operator-verifiable checksum beside the pre-migration snapshot. Both
// entries must be durable before the transaction is allowed to change schema.
func migrationBackup(ctx context.Context, source, target string) error {
	result, err := Backup(ctx, source, target)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(target+".sha256", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = fmt.Fprintf(file, "%s  %s\n", result.SHA256, filepath.Base(target)); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(target))
}
