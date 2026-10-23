package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// A declared existing schema must be complete. Missing authority or profile
// metadata is damage, never permission to synthesize a fresh table.
func validateSchema(ctx context.Context, db *sql.DB, version int) error {
	queries := []string{`SELECT id,sequence FROM metadata LIMIT 0`, `SELECT key,data FROM tasks LIMIT 0`, `SELECT key,data FROM sessions LIMIT 0`, `SELECT role,generation FROM role_generations LIMIT 0`, `SELECT cursor,data FROM events LIMIT 0`, `SELECT key,hash,response FROM idempotency LIMIT 0`, `SELECT id,hash,owner,machine,expires,revoked FROM tokens LIMIT 0`}
	if version >= 2 {
		queries = append(queries, `SELECT key,data FROM notes LIMIT 0`, `SELECT id,sequence FROM note_metadata LIMIT 0`)
	}
	if version >= 3 {
		queries = append(queries, `SELECT key,data FROM agent_profiles LIMIT 0`, `SELECT profile_id,version,data FROM agent_profile_revisions LIMIT 0`)
	}
	if version >= 4 {
		queries = append(queries, `SELECT id,version FROM coordination_metadata LIMIT 0`)
	}
	if version >= 5 {
		queries = append(queries, `SELECT key,data FROM managed_hosts LIMIT 0`, `SELECT key,data FROM managed_runs LIMIT 0`, `SELECT key,data FROM managed_commands LIMIT 0`, `SELECT id,version FROM managed_metadata LIMIT 0`)
	}
	if version >= 6 {
		for _, table := range automationTables {
			queries = append(queries, "SELECT key,data FROM "+table+" LIMIT 0")
		}
		queries = append(queries, `SELECT id,version FROM automation_metadata LIMIT 0`)
	}
	for _, q := range queries {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return fmt.Errorf("existing board schema incomplete")
		}
		rows.Close()
	}
	if version >= 6 {
		var count, v int
		if err := db.QueryRowContext(ctx, `SELECT count(*),COALESCE(MAX(version),0) FROM automation_metadata`).Scan(&count, &v); err != nil || count != 1 || v != 1 {
			return fmt.Errorf("existing automation metadata incomplete")
		}
		if err := db.QueryRowContext(ctx, `SELECT version FROM automation_metadata WHERE id=1`).Scan(&v); err != nil || v != 1 {
			return fmt.Errorf("existing automation metadata identity missing")
		}
		for _, name := range []string{"automation_step_execution", "automation_cap_attempt"} {
			if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&count); err != nil || count != 1 {
				return fmt.Errorf("existing automation constraint missing")
			}
			var actual string
			if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&actual); err != nil {
				return fmt.Errorf("existing automation constraint unreadable")
			}
			matched := false
			for _, q := range automationDDL() {
				if strings.Contains(q, "INDEX "+name+" ") {
					matched = strings.TrimSpace(actual) == strings.TrimSpace(q)
				}
			}
			if !matched {
				return fmt.Errorf("existing automation constraint changed")
			}
		}
	}
	var sequence int64
	if err := db.QueryRowContext(ctx, `SELECT sequence FROM metadata WHERE id=1`).Scan(&sequence); err != nil {
		return err
	}
	if version >= 4 {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM coordination_metadata`).Scan(&count); err != nil || count != 1 {
			return fmt.Errorf("existing coordination metadata incomplete")
		}
		var v int
		if err := db.QueryRowContext(ctx, `SELECT version FROM coordination_metadata WHERE id=1`).Scan(&v); err != nil || v != 1 {
			return fmt.Errorf("existing coordination metadata incomplete")
		}
	}
	if version >= 2 {
		if version >= 5 {
			var count, v int
			if err := db.QueryRowContext(ctx, `SELECT count(*),COALESCE(MAX(version),0) FROM managed_metadata`).Scan(&count, &v); err != nil || count != 1 || v != 1 {
				return fmt.Errorf("existing managed metadata incomplete")
			}
			if err := db.QueryRowContext(ctx, `SELECT version FROM managed_metadata WHERE id=1`).Scan(&v); err != nil || v != 1 {
				return fmt.Errorf("existing managed metadata identity missing")
			}
		}
		return db.QueryRowContext(ctx, `SELECT sequence FROM note_metadata WHERE id=1`).Scan(&sequence)
	}
	return nil
}
