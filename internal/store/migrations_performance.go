package store

import (
	"context"
	"database/sql"
	"fmt"
)

// migratePerformance runs inside the single migration write transaction. Raw
// events and replay bytes are not rewritten. Trigger invalidation also covers
// external same-version SQL changes that preserve the high-water cursor.
func migratePerformance(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		`CREATE INDEX events_task_cursor ON events(json_extract(data,'$.task_key'),cursor)`,
		`CREATE INDEX events_type_cursor ON events(json_extract(data,'$.type'),cursor)`,
		`CREATE INDEX events_task_type_cursor ON events(json_extract(data,'$.task_key'),json_extract(data,'$.type'),cursor)`,
		`CREATE TABLE projection_control(id INTEGER PRIMARY KEY CHECK(id=1),reducer_version INTEGER NOT NULL,built_cursor INTEGER NOT NULL,dirty INTEGER NOT NULL,compatibility INTEGER NOT NULL,last_sec INTEGER NOT NULL,last_ns INTEGER NOT NULL)`,
		`INSERT INTO projection_control VALUES(1,1,0,1,0,0,0)`,
		`CREATE TABLE stats_groups(kind TEXT NOT NULL,group_key TEXT NOT NULL,PRIMARY KEY(kind,group_key)) WITHOUT ROWID`,
		`CREATE TABLE stats_prefix(kind TEXT NOT NULL,group_key TEXT NOT NULL,sec INTEGER NOT NULL,ns INTEGER NOT NULL,data TEXT NOT NULL,PRIMARY KEY(kind,group_key,sec,ns)) WITHOUT ROWID`,
		`CREATE TABLE blocked_prefix(kind TEXT NOT NULL,group_key TEXT NOT NULL,sec INTEGER NOT NULL,ns INTEGER NOT NULL,area TEXT NOT NULL,active TEXT NOT NULL,PRIMARY KEY(kind,group_key,sec,ns)) WITHOUT ROWID`,
		`CREATE TABLE blocked_open(task_key TEXT PRIMARY KEY,data TEXT NOT NULL,sec INTEGER NOT NULL,ns INTEGER NOT NULL)`,
		`CREATE INDEX blocked_open_time ON blocked_open(sec,ns)`,
	}
	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("performance migration: %w", err)
		}
	}
	for _, event := range []string{"INSERT", "UPDATE", "DELETE"} {
		query := `CREATE TRIGGER projection_dirty_` + event + ` AFTER ` + event + ` ON events BEGIN UPDATE projection_control SET dirty=1 WHERE id=1; END`
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	if err := rebuildProjection(ctx, tx); err != nil {
		return err
	}
	if err := validateMigrationProjection(ctx, tx); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `PRAGMA user_version=4`)
	return err
}
