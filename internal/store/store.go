// Package store owns durable transactions and authoritative server time.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"agent-board/internal/core"
	_ "modernc.org/sqlite"
)

type Store struct {
	db    *sql.DB
	cfg   core.Config
	clock core.Clock
}
type replay struct {
	Hash     string          `json:"hash"`
	Response json.RawMessage `json:"response"`
}
type board struct {
	Tasks     map[string]*core.Task    `json:"tasks"`
	Sessions  map[string]*core.Session `json:"sessions"`
	Roles     map[string]int64         `json:"roles"`
	Events    []core.Event             `json:"events"`
	Sequence  int64                    `json:"sequence"`
	EventBase int64                    `json:"-"`
	tx        *sql.Tx
}

// Open serializes transactions in process and obtains a SQLite write lock before
// reading board state, so separate server processes cannot claim the same task.
func Open(path string, cfg core.Config, clock core.Clock) (*Store, error) {
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 15 * time.Minute
	}
	if cfg.LeaseTTL == 0 {
		cfg.LeaseTTL = 15 * time.Minute
	}
	if cfg.SessionTTL < time.Second || cfg.LeaseTTL < time.Second {
		return nil, core.Fail("invalid_input", "TTL must be at least one second")
	}
	if clock == nil {
		clock = time.Now
	}
	if path == "" {
		return nil, core.Fail("invalid_input", "database path is required")
	}
	dsn, err := prepareDatabase(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, cfg: cfg, clock: clock}
	var schemaVersion int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&schemaVersion); err != nil {
		db.Close()
		return nil, err
	}
	if schemaVersion > 1 {
		db.Close()
		return nil, core.Fail("rule_violation", "database schema is newer than this server")
	}
	for _, q := range []string{
		`PRAGMA busy_timeout=5000`, `PRAGMA journal_mode=WAL`,
		`CREATE TABLE IF NOT EXISTS metadata (id INTEGER PRIMARY KEY CHECK(id=1),sequence INTEGER NOT NULL)`,
		`INSERT OR IGNORE INTO metadata VALUES(1,0)`,
		`CREATE TABLE IF NOT EXISTS tasks (key TEXT PRIMARY KEY,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS sessions (key TEXT PRIMARY KEY,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS role_generations (role TEXT PRIMARY KEY,generation INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS events (cursor INTEGER PRIMARY KEY AUTOINCREMENT,data TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS idempotency (key TEXT PRIMARY KEY,hash TEXT NOT NULL,response TEXT NOT NULL)`,
		`PRAGMA user_version=1`,
		`CREATE TABLE IF NOT EXISTS tokens (id TEXT PRIMARY KEY,hash TEXT UNIQUE NOT NULL,owner TEXT NOT NULL,machine TEXT NOT NULL,expires INTEGER NOT NULL,revoked INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("initialize database: %w", err)
		}
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

func transact[T any](s *Store, ctx context.Context, fn func(*board, time.Time) (T, error)) (T, error) {
	var zero T
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE metadata SET sequence=sequence WHERE id=1`); err != nil {
		return zero, err
	}
	b, err := loadBoard(ctx, tx)
	if err != nil {
		return zero, err
	}
	now := s.clock().UTC()
	s.expire(b, now)
	result, domainErr := fn(b, now)
	if err = saveBoard(ctx, tx, b); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return result, domainErr
}
func (s *Store) expire(b *board, now time.Time) {
	for _, t := range b.Tasks {
		if t.Lease != nil {
			session := b.Sessions[t.Lease.SessionID]
			if !t.Lease.ExpiresAt.After(now) || session == nil || !session.ExpiresAt.After(now) {
				old := t.Lease.SessionID
				finishAttempt(t, now, "lease_expired", "")
				t.Lease = nil
				t.State = "ready"
				t.ReadyAt = now
				t.Version++
				emit(b, now, "lease_expired", t.Key, old)
			}
		}
	}
}
func emit(b *board, now time.Time, kind, key, session string) {
	b.Events = append(b.Events, core.Event{Cursor: b.EventBase + int64(len(b.Events)+1), Type: kind, TaskKey: key, SessionID: session, At: now})
}
func sessionFor(b *board, a core.Actor, now time.Time) (*core.Session, error) {
	v := b.Sessions[a.SessionID]
	if v == nil || v.PrincipalID != a.Principal.ID {
		return nil, core.Fail("unauthorized", "session does not belong to credential")
	}
	if !v.ExpiresAt.After(now) {
		return nil, core.Fail("session_expired", "open a new session; old session cannot resume")
	}
	return v, nil
}

// Ready checks the database connection without renewing a session.
func (s *Store) Ready(ctx context.Context) error { return s.db.PingContext(ctx) }
