package projects

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"
)

// migrateRegistry upgrades a schema1 authority to schema2 (docs/admin.md). A
// consistent copy registry1-<sha256>.db is durable in <root>/registry.sqlite.migrations
// before any DDL; DDL and people bootstrap commit together or not at all.
func migrateRegistry(root string, db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == RegistrySchemaVersion {
		return nil
	}
	if version != 1 {
		return errors.New("unsupported registry migration source")
	}
	if err := registryMigrationCopy(root, db, version); err != nil {
		return fmt.Errorf("registry pre-migration copy: %w", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// The write serializes concurrent openers; re-read the version under the lock.
	if _, err = tx.Exec(`UPDATE registry_meta SET default_id=default_id WHERE id=1`); err != nil {
		return err
	}
	if err = tx.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == RegistrySchemaVersion {
		return nil
	}
	if _, err = tx.Exec(registryMigration1to2); err != nil {
		return fmt.Errorf("registry migration DDL: %w", err)
	}
	if err = bootstrapPeople(tx); err != nil {
		return fmt.Errorf("registry people bootstrap: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return validateRegistry(db)
}

// registryMigrationCopy publishes a VACUUM INTO snapshot named by its own
// SHA-256 so operators can verify the copy; symlinks and foreign modes refuse.
func registryMigrationCopy(root string, db *sql.DB, version int) error {
	dir := filepath.Join(root, "registry.sqlite.migrations")
	if err := os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return errors.New("migration copy directory must be a private mode0700 directory")
	}
	if err = syncRegistryDirectory(root); err != nil {
		return err
	}
	stage, err := os.CreateTemp(dir, ".registry-copy-*.db")
	if err != nil {
		return err
	}
	stagePath := stage.Name()
	defer os.Remove(stagePath)
	defer stage.Close()
	if _, err = db.Exec(`VACUUM INTO ?`, stagePath); err != nil {
		return err
	}
	digest := sha256.New()
	if _, err = io.Copy(digest, stage); err != nil {
		return err
	}
	if err = stage.Sync(); err != nil {
		return err
	}
	final := filepath.Join(dir, fmt.Sprintf("registry%d-%s.db", version, hex.EncodeToString(digest.Sum(nil))))
	if err = os.Link(stagePath, final); err != nil && !os.IsExist(err) {
		return err
	}
	// An existing name holds byte-identical content: the hash is the name.
	published, err := os.Lstat(final)
	if err != nil {
		return err
	}
	if !published.Mode().IsRegular() || published.Mode().Perm() != 0600 {
		return errors.New("migration copy must be a regular mode0600 file")
	}
	return syncRegistryDirectory(dir)
}

// bootstrapPeople gives every identity its own person. Owner labels are not
// keys, so equal labels are never merged; an admin links them by hand later.
// Active owners of the default project become the first global admins.
func bootstrapPeople(tx *sql.Tx) error {
	var defaultID string
	if err := tx.QueryRow(`SELECT default_id FROM registry_meta WHERE id=1`).Scan(&defaultID); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT i.id,i.owner,i.machine,EXISTS(SELECT 1 FROM memberships m WHERE m.project_id=? AND m.identity_id=i.id AND m.role='owner' AND m.active=1) FROM identities i WHERE i.person_id='' ORDER BY i.id`, defaultID)
	if err != nil {
		return err
	}
	type row struct {
		id, owner, machine string
		admin              bool
	}
	items := []row{}
	for rows.Next() {
		var item row
		if err = rows.Scan(&item.id, &item.owner, &item.machine, &item.admin); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	taken := map[string]bool{}
	now := time.Now().UnixNano()
	for _, item := range items {
		name := uniquePersonName(item.owner, item.machine, taken)
		role := "member"
		if item.admin {
			role = "admin"
		}
		pid, err := random("u_", 16)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO people(id,name,note,role,active,version,created,updated) VALUES(?,?,'',?,1,1,?,?)`, pid, name, role, now, now); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE identities SET person_id=? WHERE id=?`, pid, item.id); err != nil {
			return err
		}
	}
	return nil
}
func uniquePersonName(owner, machine string, taken map[string]bool) string {
	name := truncateRunes(owner, MaxPersonName)
	if taken[name] {
		name = truncateRunes(owner+" ("+machine+")", MaxPersonName)
	}
	for base, n := name, 2; taken[name]; n++ {
		name = truncateRunes(fmt.Sprintf("%s #%d", base, n), MaxPersonName)
	}
	taken[name] = true
	return name
}
func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
