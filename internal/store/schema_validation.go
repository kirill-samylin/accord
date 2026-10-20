package store

import (
	"context"
	"database/sql"
	"fmt"
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
	for _, q := range queries {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return fmt.Errorf("existing board schema incomplete")
		}
		rows.Close()
	}
	var sequence int64
	if err := db.QueryRowContext(ctx, `SELECT sequence FROM metadata WHERE id=1`).Scan(&sequence); err != nil {
		return err
	}
	if version >= 2 {
		return db.QueryRowContext(ctx, `SELECT sequence FROM note_metadata WHERE id=1`).Scan(&sequence)
	}
	return nil
}
