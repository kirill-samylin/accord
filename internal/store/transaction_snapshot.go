package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func (s *Store) acquire(ctx context.Context) error {
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	s.life.Lock()
	closed := s.closed
	s.life.Unlock()
	if closed {
		<-s.gate
		return sql.ErrConnDone
	}
	return nil
}

// transaction holds a dedicated SQLite immediate transaction before reading the
// clock, data_version, authority or configuration. data_version belongs only to
// this retained connection; a replacement always starts with an empty cache.
func transaction[T any](s *Store, ctx context.Context, expiry bool, fn func(*board, time.Time) (T, error)) (T, error) {
	var zero T
	if err := s.acquire(ctx); err != nil {
		return zero, err
	}
	defer func() { <-s.gate }()
	committed := false
	defer func() {
		if !committed {
			s.cache = nil
			if s.writer != nil {
				s.writer.Close()
				s.writer = nil
			}
		}
	}()
	var err error
	if s.writer == nil {
		err = startupRetry(ctx, func(attempt context.Context) error {
			s.writer, err = s.db.Conn(attempt)
			return err
		})
		if err != nil {
			return zero, err
		}
		s.cache = nil
	}
	tx, err := beginImmediate(ctx, s.writer)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	var version int64
	if err = tx.QueryRowContext(ctx, `PRAGMA data_version`).Scan(&version); err != nil {
		return zero, err
	}
	source := s.cache
	if source == nil || version != s.dataVersion {
		source, err = loadBoard(ctx, tx)
		if err != nil {
			return zero, err
		}
	}
	b := source.fork()
	b.tx = tx
	b.Config, err = cachedConfig(ctx, tx, source.Config, s.cfg)
	if err != nil {
		return zero, err
	}
	now := s.clock().UTC()
	before := metricFacts(b)
	if expiry {
		s.expire(b, now)
		enrichEvents(b, before, now)
		before = metricFacts(b)
	}
	result, domainErr := fn(b, now)
	if domainErr != nil {
		var domain *core.Error
		if !expiry || !errors.As(domainErr, &domain) {
			return zero, domainErr
		}
	}
	if b.copyErr != nil {
		return zero, b.copyErr
	}
	enrichEvents(b, before, now)
	// The public result is detached while failure can still roll back the commit.
	result, err = detach(result)
	if err != nil {
		return zero, err
	}
	if err = saveBoard(ctx, tx, b); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	b.EventBase += int64(len(b.Events))
	b.Events = nil
	b.commitIndexes()
	b.tx, b.base = nil, nil
	b.dirtyTasks, b.dirtySessions, b.dirtyNotes = nil, nil, nil
	b.gateMemo = nil
	s.cache, s.dataVersion = b, version
	committed = true
	return result, domainErr
}

// cachedConfig is bounded to the current committed revision. Persisted bytes and
// hash are checked in every authoritative transaction. Decode owns private maps.
func cachedConfig(ctx context.Context, tx *sql.Tx, prior, bootstrap core.Config) (core.Config, error) {
	var version int64
	var hash string
	var raw []byte
	var schema int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&schema); err != nil {
		return core.Config{}, err
	}
	if schema < 3 {
		return bootstrap, nil
	}
	err := tx.QueryRowContext(ctx, `SELECT h.version,h.sha256,h.canonical_json FROM config_history h JOIN config_active a ON a.version=h.version WHERE a.id=1`).Scan(&version, &hash, &raw)
	if err == sql.ErrNoRows && bootstrap.SkipBootstrap {
		return bootstrap, nil
	}
	if err != nil {
		return core.Config{}, err
	}
	if prior.Snapshot != nil && prior.ConfigVersion == version && prior.Snapshot.SHA256() == hash && bytes.Equal(prior.Snapshot.CanonicalJSON(), raw) {
		return runtimeConfig(prior.Snapshot, version), nil
	}
	snapshot, validation := config.Parse(raw, config.Options{})
	if !validation.Valid || snapshot.SHA256() != hash || !bytes.Equal(snapshot.CanonicalJSON(), raw) || !json.Valid(raw) {
		return core.Config{}, core.Fail("CONFIG_INTERNAL", "persisted configuration bytes/hash invalid")
	}
	return runtimeConfig(snapshot, version), nil
}

func readTaskFor(b *board, key string) (*core.Task, error) {
	t := b.Tasks[key]
	if t == nil {
		return nil, core.Fail("not_found", "task does not exist")
	}
	return t, nil
}
