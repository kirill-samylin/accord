package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func loadBoard(ctx context.Context, tx *sql.Tx) (*board, error) {
	b := &board{Profiles: map[string]*core.AgentProfile{}, Notes: map[string]*core.Note{}, Tasks: map[string]*core.Task{}, Sessions: map[string]*core.Session{}, Roles: map[string]int64{}, tx: tx}
	if err := tx.QueryRowContext(ctx, `SELECT sequence FROM metadata WHERE id=1`).Scan(&b.Sequence); err != nil {
		return nil, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cursor),0) FROM events`).Scan(&b.EventBase); err != nil {
		return nil, err
	}
	if err := loadEntities(ctx, tx, "tasks", b.Tasks); err != nil {
		return nil, err
	}
	if err := loadEntities(ctx, tx, "sessions", b.Sessions); err != nil {
		return nil, err
	}
	var schema int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&schema); err != nil {
		return nil, err
	}
	if schema >= 2 {
		if err := loadEntities(ctx, tx, "notes", b.Notes); err != nil {
			return nil, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT sequence FROM note_metadata WHERE id=1`).Scan(&b.NoteSequence); err != nil {
			return nil, err
		}
	}
	if schema >= 3 {
		if err := loadEntities(ctx, tx, "agent_profiles", b.Profiles); err != nil {
			return nil, err
		}
	}
	if schema >= 5 {
		b.ManagedHosts = map[string]*managedHostRecord{}
		b.ManagedRuns = map[string]*core.ManagedRun{}
		b.ManagedCommands = map[string]*managedCommandRecord{}
		if err := loadEntities(ctx, tx, "managed_hosts", b.ManagedHosts); err != nil {
			return nil, err
		}
		if err := loadEntities(ctx, tx, "managed_runs", b.ManagedRuns); err != nil {
			return nil, err
		}
		if err := loadEntities(ctx, tx, "managed_commands", b.ManagedCommands); err != nil {
			return nil, err
		}
	}
	if schema >= 6 {
		if err := loadAutomation(ctx, tx, b); err != nil {
			return nil, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT role,generation FROM role_generations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		var generation int64
		if err = rows.Scan(&role, &generation); err != nil {
			return nil, err
		}
		b.Roles[role] = generation
	}
	return b, rows.Err()
}
func loadEntities[T any](ctx context.Context, tx *sql.Tx, table string, target map[string]*T) error {
	// table is always a static internal name, never external input. Archived
	// history rows are not live state; AutomationArchive pages them.
	rows, err := tx.QueryContext(ctx, "SELECT key,data FROM "+table+" WHERE substr(key,1,8)<>'"+automationArchivePrefix+"'")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var raw []byte
		if err = rows.Scan(&key, &raw); err != nil {
			return err
		}
		var v T
		if err = json.Unmarshal(raw, &v); err != nil {
			return err
		}
		target[key] = &v
	}
	return rows.Err()
}
func saveBoard(ctx context.Context, tx *sql.Tx, b *board) error {
	for _, m := range b.archive {
		raw, err := json.Marshal(m.value)
		if err != nil {
			return err
		}
		// Static table names only; a duplicate archive key fails the transaction,
		// and so does a live row that is missing (nothing may be archived twice
		// or silently dropped).
		res, err := tx.ExecContext(ctx, "UPDATE "+m.table+" SET key=?,data=? WHERE key=?", automationArchivePrefix+m.key, string(raw), m.key)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n != 1 {
			return fmt.Errorf("archive %s row %q: %d live rows renamed, want 1", m.table, m.key, n)
		}
	}
	b.archive = nil
	if err := saveAutomation(ctx, tx, b); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "managed_hosts", b.ManagedHosts); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "managed_runs", b.ManagedRuns); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "managed_commands", b.ManagedCommands); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "agent_profiles", b.Profiles); err != nil {
		return err
	}
	for _, v := range b.ProfileRevisions {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO agent_profile_revisions(profile_id,version,data) VALUES(?,?,?)`, v.ID, v.Version, string(raw)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE metadata SET sequence=? WHERE id=1`, b.Sequence); err != nil {
		return err
	}
	if len(b.Notes) > 0 {
		if err := saveEntities(ctx, tx, "notes", b.Notes); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE note_metadata SET sequence=? WHERE id=1`, b.NoteSequence); err != nil {
			return err
		}
	}
	if err := saveEntities(ctx, tx, "tasks", b.Tasks); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "sessions", b.Sessions); err != nil {
		return err
	}
	for role, generation := range b.Roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO role_generations VALUES(?,?) ON CONFLICT(role) DO UPDATE SET generation=excluded.generation WHERE generation<>excluded.generation`, role, generation); err != nil {
			return err
		}
	}
	for _, e := range b.Events {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO events(cursor,data) VALUES(?,?)`, e.Cursor, string(raw)); err != nil {
			return err
		}
	}
	return nil
}
func saveEntities[T any](ctx context.Context, tx *sql.Tx, table string, values map[string]*T) error {
	for key, v := range values {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO "+table+"(key,data) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET data=excluded.data WHERE data<>excluded.data", key, string(raw)); err != nil {
			return err
		}
	}
	return nil
}
