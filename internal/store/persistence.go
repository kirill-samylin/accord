package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
)

func loadBoard(ctx context.Context, tx *sql.Tx) (*board, error) {
	b := &board{Tasks: map[string]*core.Task{}, Sessions: map[string]*core.Session{}, Roles: map[string]int64{}, tx: tx}
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
	if _, err := tx.ExecContext(ctx, `UPDATE metadata SET sequence=? WHERE id=1`, b.Sequence); err != nil {
		return err
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
