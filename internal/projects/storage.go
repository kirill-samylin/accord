package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

func privateRoot(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil || path == "" {
		return "", errors.New("private projects root required")
	}
	missingParents := []string{}
	for parent := absolute; ; parent = filepath.Dir(parent) {
		info, e := os.Lstat(parent)
		if e == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 && parent != "/tmp" && parent != "/private/tmp") {
			return "", errors.New("unsafe projects directory")
		}
		if os.IsNotExist(e) {
			missingParents = append(missingParents, parent)
		} else if e != nil {
			return "", e
		}
		if parent == filepath.Dir(parent) {
			break
		}
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode().Perm() != 0700 {
		return "", errors.New("projects root must be private mode0700")
	}
	seen := map[string]bool{}
	for _, created := range missingParents {
		for _, dir := range []string{created, filepath.Dir(created)} {
			if seen[dir] {
				continue
			}
			seen[dir] = true
			file, e := os.Open(dir)
			if e != nil {
				return "", e
			}
			e = file.Sync()
			file.Close()
			if e != nil {
				return "", e
			}
		}
	}
	return absolute, nil
}
func transaction[T any](r *Registry, ctx context.Context, fn func(*sql.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE registry_meta SET default_id=default_id WHERE id=1`); err != nil {
		return zero, err
	}
	if err = checkGlobal(ctx, tx); err != nil {
		return zero, err
	}
	if err = r.catalogGate(ctx, tx); err != nil {
		return zero, err
	}
	value, err := fn(tx)
	if err != nil {
		return zero, err
	}
	commit := tx.Commit
	if action, ok := ctx.Value(catalogActionKey{}).(catalogAction); ok && action.Identity.Source == "legacy" {
		p := core.Principal{ID: action.Identity.TokenID, Owner: action.Identity.Owner, Machine: action.Identity.Machine}
		commit = func() error { return r.options.Default.GuardCredential(ctx, p, tx.Commit) }
	}
	if err = commit(); err != nil {
		return zero, err
	}
	return value, nil
}
func (r *Registry) Ready(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return err
	}
	return r.options.Default.Ready(ctx)
}
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, v := range r.runtimes {
		if id != r.defaultID {
			if v.browser != nil {
				v.browser.Close()
			}
			v.store.Close()
		}
	}
	return r.db.Close()
}
func notFound() error { return core.Fail("not_found", "project or membership not found") }
func denied() error   { return core.Fail("forbidden", "project permission is required") }
func invalid() error  { return core.Fail("invalid_input", "invalid project fields") }
func wrapped(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("project operation failed: %w", err)
}
