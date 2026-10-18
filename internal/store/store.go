// Package store owns durable transactions and authoritative server time.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"agent-board/internal/core"
	_ "modernc.org/sqlite"
)

type Store struct {
	db          *sql.DB
	cfg         core.Config
	clock       core.Clock
	writer      *sql.Conn
	gate        chan struct{}
	life        sync.Mutex
	closed      bool
	cache       *board
	dataVersion int64
}
type replay struct {
	Hash     string          `json:"hash"`
	Response json.RawMessage `json:"response"`
}
type board struct {
	Notes         map[string]*core.Note    `json:"notes"`
	NoteSequence  int64                    `json:"note_sequence"`
	Config        core.Config              `json:"config"`
	Tasks         map[string]*core.Task    `json:"tasks"`
	Sessions      map[string]*core.Session `json:"sessions"`
	Roles         map[string]int64         `json:"roles"`
	Events        []core.Event             `json:"events"`
	Sequence      int64                    `json:"sequence"`
	EventBase     int64                    `json:"-"`
	base          *board
	dirtyTasks    map[string]bool
	dirtySessions map[string]bool
	dirtyNotes    map[string]bool
	copyErr       error
	gateMemo      map[string]bool
	index         *snapshotIndexes
	facts         map[string]taskFact
	tx            *sql.Tx
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
	if cfg.SessionTTL < time.Second || cfg.LeaseTTL < time.Second || cfg.SessionTTL > 24*time.Hour || cfg.LeaseTTL > 24*time.Hour {
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
	if dsn == ":memory:" {
		id, err := identifier()
		if err != nil {
			return nil, err
		}
		dsn = "file:accord-memory-" + id + "?mode=memory&cache=shared"
	}
	db, err := sql.Open("sqlite", connectionDSN(dsn))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	s := newStore(db, cfg, clock)
	err = migrate(context.Background(), db, path, cfg)
	if err == nil {
		err = startupRetry(context.Background(), func(ctx context.Context) error {
			_, err := db.ExecContext(ctx, `PRAGMA journal_mode=WAL`)
			return err
		})
	}
	if err == nil && !cfg.SkipBootstrap {
		err = s.checkStartup(context.Background(), cfg)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func connectionDSN(dsn string) string {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_txlock=immediate&_pragma=" + url.QueryEscape("busy_timeout(5000)") + "&_pragma=" + url.QueryEscape("cache_size(-8192)")
}
func newStore(db *sql.DB, cfg core.Config, clock core.Clock) *Store {
	return &Store{db: db, cfg: cfg, clock: clock, gate: make(chan struct{}, 1)}
}

// Close stops admitting transactions, waits for the current owner, then releases
// the retained connection before closing its pool. Waiting callers fail closed.
func (s *Store) Close() error {
	s.life.Lock()
	s.closed = true
	s.life.Unlock()
	s.gate <- struct{}{}
	defer func() { <-s.gate }()
	if s.writer != nil {
		s.writer.Close()
		s.writer = nil
	}
	s.cache = nil
	return s.db.Close()
}

func transact[T any](s *Store, ctx context.Context, fn func(*board, time.Time) (T, error)) (T, error) {
	return transaction(s, ctx, true, fn)
}
func (s *Store) expire(b *board, now time.Time) {
	for _, t := range b.leasedTasks() {
		if t.Lease != nil {
			session := b.Sessions[t.Lease.SessionID]
			if !t.Lease.ExpiresAt.After(now) || !sessionLive(b, session, now) {
				t = b.editTask(t.Key)
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
	b.gateMemo = map[string]bool{}
	b.Events = append(b.Events, core.Event{Cursor: b.EventBase + int64(len(b.Events)+1), Type: kind, TaskKey: key, SessionID: session, At: now})
}
func sessionFor(b *board, a core.Actor, now time.Time) (*core.Session, error) {
	v := b.Sessions[a.SessionID]
	if v == nil || v.PrincipalID != a.Principal.ID {
		return nil, core.Fail("unauthorized", "session does not belong to credential")
	}
	if !sessionLive(b, v, now) {
		return nil, core.Fail("session_expired", "open a new session; old session cannot resume")
	}
	return v, nil
}

// Ready checks the database connection without renewing a session.
func (s *Store) Ready(ctx context.Context) error { return s.db.PingContext(ctx) }
