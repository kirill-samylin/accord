package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// existingDatabase validates before SQLite can create or interpret anything.
// It deliberately does not repair unsafe modes or manufacture missing parents.
func existingDatabase(path string) (string, error) {
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") {
		return "", fmt.Errorf("existing database requires a filesystem path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return "", fmt.Errorf("existing database must be a regular file with mode 0600")
	}
	parent, err := os.Stat(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	if !parent.IsDir() || parent.Mode().Perm()&0022 != 0 {
		return "", fmt.Errorf("database parent must not be group or other writable")
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err = privateRegularFile(absolute + suffix); err != nil {
			return "", err
		}
	}
	return absolute, nil
}

func openReadOnly(ctx context.Context, path string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err == nil {
		err = verifySchema(ctx, db)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func verifySchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version < 1 || version > SchemaVersion {
		return fmt.Errorf("backup requires a supported schema version")
	}
	return nil
}

// Backup makes a consistent online SQLite snapshot without copying live WAL.
// The destination is new and private from its first byte. The directory owner
// and other processes with the same OS identity remain trusted.
func Backup(ctx context.Context, source, output string) (core.BackupResult, error) {
	var result core.BackupResult
	absolute, err := existingDatabase(source)
	if err != nil {
		return result, err
	}
	if output == "" || output == ":memory:" || strings.HasPrefix(output, "file:") {
		return result, fmt.Errorf("backup output requires a new filesystem path")
	}
	target, err := filepath.Abs(output)
	if err != nil {
		return result, err
	}
	parent, err := os.Lstat(filepath.Dir(target))
	if err != nil {
		return result, err
	}
	if !parent.IsDir() || parent.Mode().Perm() != 0700 {
		return result, fmt.Errorf("backup output parent must be a private mode 0700 directory")
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, statErr := os.Lstat(target + suffix); !os.IsNotExist(statErr) {
			return result, fmt.Errorf("backup output sidecar already exists or cannot be inspected")
		}
	}
	if _, statErr := os.Lstat(target); !os.IsNotExist(statErr) {
		return result, fmt.Errorf("backup output must be a new private file")
	}
	db, err := openReadOnly(ctx, absolute)
	if err != nil {
		return result, err
	}
	defer db.Close()
	// CreateTemp uses O_EXCL and 0600. Stage in the same directory so hardlink
	// publication is atomic and cannot overwrite an existing final name.
	file, err := os.CreateTemp(filepath.Dir(target), ".accord-backup-*.db")
	if err != nil {
		return result, fmt.Errorf("backup output must be a new private file")
	}
	stage := file.Name()
	completed := false
	published := false
	created, err := file.Stat()
	if err != nil {
		file.Close()
		return result, err
	}
	defer func() {
		_ = file.Close()
		removeOwnSnapshot(stage, created)
		if !completed && published {
			removeOwnSnapshot(target, created)
		}
	}()
	if _, err = db.ExecContext(ctx, `VACUUM INTO ?`, stage); err != nil {
		return result, err
	}
	if err = checkSnapshot(ctx, stage); err != nil {
		return result, err
	}
	snapshot, err := openReadOnly(ctx, stage)
	if err != nil {
		return result, err
	}
	var snapshotVersion int
	err = snapshot.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&snapshotVersion)
	closeErr := snapshot.Close()
	if err != nil {
		return result, err
	}
	if closeErr != nil {
		return result, closeErr
	}
	if err = file.Sync(); err != nil {
		return result, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return result, err
	}
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return result, err
	}
	if err = file.Close(); err != nil {
		return result, err
	}
	if err = os.Link(stage, target); err != nil {
		return result, fmt.Errorf("cannot publish backup without overwriting: %w", err)
	}
	published = true
	if err = syncDirectory(filepath.Dir(target)); err != nil {
		return result, err
	}
	if err = os.Remove(stage); err != nil {
		return result, err
	}
	if err = syncDirectory(filepath.Dir(target)); err != nil {
		return result, err
	}
	result = core.BackupResult{SHA256: hex.EncodeToString(digest.Sum(nil)), CreatedAt: time.Now().UTC(), SchemaVersion: snapshotVersion}
	completed = true
	return result, nil
}

func removeOwnSnapshot(path string, created os.FileInfo) {
	// Never remove an operator's replacement of either the stage or final name.
	current, err := os.Lstat(path)
	if err == nil && os.SameFile(created, current) {
		_ = os.Remove(path)
	}
}

func checkSnapshot(ctx context.Context, path string) error {
	db, err := openReadOnly(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var status string
		if err = rows.Scan(&status); err != nil {
			return err
		}
		if status != "ok" {
			return fmt.Errorf("backup integrity check failed")
		}
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("backup integrity check returned no confirmation")
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
