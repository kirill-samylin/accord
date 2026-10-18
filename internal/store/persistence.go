package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
)

func loadBoard(ctx context.Context, tx *sql.Tx) (*board, error) {
	b := &board{Notes: map[string]*core.Note{}, Tasks: map[string]*core.Task{}, Sessions: map[string]*core.Session{}, Roles: map[string]int64{}, tx: tx}
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
	// table is always a static internal name, never external input.
	rows, err := tx.QueryContext(ctx, "SELECT key,data FROM "+table)
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
	tasks, sessions, notes := map[string]*core.Task{}, map[string]*core.Session{}, map[string]*core.Note{}
	original := b
	for part := b; part != nil; part = part.base {
		original = part
		for key := range part.dirtyTasks {
			tasks[key] = b.Tasks[key]
		}
		for key := range part.dirtySessions {
			sessions[key] = b.Sessions[key]
		}
		for key := range part.dirtyNotes {
			notes[key] = b.Notes[key]
		}
	}
	if b.Sequence != original.Sequence {
		if _, err := tx.ExecContext(ctx, `UPDATE metadata SET sequence=? WHERE id=1`, b.Sequence); err != nil {
			return err
		}
	}
	if b.NoteSequence != original.NoteSequence {
		if _, err := tx.ExecContext(ctx, `UPDATE note_metadata SET sequence=? WHERE id=1`, b.NoteSequence); err != nil {
			return err
		}
	}
	if err := saveEntities(ctx, tx, "tasks", tasks); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "sessions", sessions); err != nil {
		return err
	}
	if err := saveEntities(ctx, tx, "notes", notes); err != nil {
		return err
	}
	for role, generation := range b.Roles {
		if original.Roles[role] == generation {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO role_generations VALUES(?,?) ON CONFLICT(role) DO UPDATE SET generation=excluded.generation`, role, generation); err != nil {
			return err
		}
	}
	return persistNewEvents(ctx, b)
}
func saveEntities[T any](ctx context.Context, tx *sql.Tx, table string, values map[string]*T) error {
	if len(values) == 0 {
		return nil
	}
	// table names are internal constants. Prepare once per changed entity table.
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO "+table+"(key,data) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET data=excluded.data WHERE data<>excluded.data")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if _, err = stmt.ExecContext(ctx, key, string(raw)); err != nil {
			return err
		}
	}
	return nil
}
