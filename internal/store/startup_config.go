package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"time"
)

func (s *Store) checkStartup(ctx context.Context, cfg core.Config) error {
	v, err := s.Config(ctx)
	if err != nil {
		return err
	}
	snapshot, result := config.Parse(v.Config, config.Options{})
	if !result.Valid {
		return core.Fail("CONFIG_INTERNAL", "invalid persisted configuration")
	}
	active := runtimeConfig(snapshot, v.ConfigVersion)
	if cfg.ExplicitProject && cfg.Project != active.Project || cfg.ExplicitTaskTTL && cfg.LeaseTTL != active.LeaseTTL || cfg.ExplicitSessionTTL && cfg.SessionTTL != active.SessionTTL {
		return core.Fail("CONFIG_CONFLICT", "explicit project/TTL flags conflict with persisted configuration")
	}
	return nil
}

// DatabaseSchema inspects an existing private database without migrating it.
func DatabaseSchema(ctx context.Context, path string) (int, error) {
	absolute, err := existingDatabase(path)
	if err != nil {
		return 0, err
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var n int
	err = startupRetry(ctx, func(attempt context.Context) error {
		return db.QueryRowContext(attempt, `PRAGMA user_version`).Scan(&n)
	})
	if err == nil && (n < 0 || n > SchemaVersion) {
		return 0, core.Fail("CONFIG_UNSUPPORTED", "database schema is newer than this binary")
	}
	return n, err
}

// Sweep materializes expired leases and reads the interval anew on every cycle.
func (s *Store) Sweep(ctx context.Context) (time.Duration, error) {
	return transact(s, ctx, func(b *board, _ time.Time) (time.Duration, error) {
		return time.ParseDuration(b.Config.Workflow.Leases.SweepInterval)
	})
}
