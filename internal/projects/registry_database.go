package projects

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

// openRegistry opens an already initialized authority. Missing tables or meta
// must never be repaired into empty ACL state after the activation marker exists.
func openRegistry(root string) (*sql.DB, error)   { return registryDatabase(root, false) }
func createRegistry(root string) (*sql.DB, error) { return registryDatabase(root, true) }
func registryDatabase(root string, fresh bool) (*sql.DB, error) {
	path := filepath.Join(root, "registry.sqlite")
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, err := os.Lstat(path + suffix)
		if err == nil && (!info.Mode().IsRegular() || info.Mode().Perm() != 0600) {
			return nil, errors.New("unsafe registry database")
		}
		if err != nil && (!os.IsNotExist(err) || suffix == "" && !fresh) {
			return nil, errors.New("existing registry authority required")
		}
		if fresh && err == nil {
			return nil, errors.New("fresh registry path already exists")
		}
	}
	if fresh {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err = file.Close(); err != nil {
			return nil, err
		}
	}
	// mode=rw prevents SQLite from recreating an authority deleted after Lstat.
	target := &url.URL{Scheme: "file", Path: path, RawQuery: "mode=rw"}
	db, err := sql.Open("sqlite", target.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA busy_timeout=5000`); err == nil {
		if fresh {
			_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; BEGIN IMMEDIATE;` + schema + `COMMIT;`)
		} else {
			err = validateRegistry(db)
		}
	}
	if err == nil && !fresh {
		_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL;`)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func validateRegistry(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 1 {
		return errors.New("existing registry schema1 authority required")
	}
	var integrity string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return errors.New("registry authority integrity failure")
	}
	// Read required columns before any schema DDL. Empty missing auth tables must
	// not silently replace lost membership or logout/rejection tombstones.
	for _, query := range []string{
		`SELECT id,default_id,default_db,default_web,signing_key FROM registry_meta LIMIT 0`,
		`SELECT id,at FROM rejected_provisions LIMIT 0`,
		`SELECT id,source,token_id,owner,machine FROM identities LIMIT 0`,
		`SELECT identity_id,hash,expires,revoked FROM credentials LIMIT 0`,
		`SELECT id,name,state,version,creator,created FROM projects LIMIT 0`,
		`SELECT project_id,identity_id,role,local_id,active,version,access_epoch FROM memberships LIMIT 0`,
		`SELECT id,project_id,url,branch,version FROM repositories LIMIT 0`,
		`SELECT hash,identity_id,csrf,expires FROM sessions LIMIT 0`,
		`SELECT hash,expires FROM session_tombstones LIMIT 0`,
		`SELECT project_id,session_id,identity_id,access_epoch FROM session_epochs LIMIT 0`,
		`SELECT hash,csrf,expires FROM preauth LIMIT 0`,
		`SELECT id,session_hash,operation,project_id,access_epoch,payload,result,created FROM actions LIMIT 0`,
		`SELECT id,event,identity_id,project_id,at FROM audit LIMIT 0`,
	} {
		rows, err := db.Query(query)
		if err != nil {
			return errors.New("registry authority schema incomplete")
		}
		rows.Close()
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM registry_meta`).Scan(&count); err != nil || count != 1 {
		return errors.New("registry authority metadata missing")
	}
	var id, dbPath, webPath, key string
	if err := db.QueryRow(`SELECT default_id,default_db,default_web,signing_key FROM registry_meta WHERE id=1`).Scan(&id, &dbPath, &webPath, &key); err != nil {
		return errors.New("registry authority metadata missing")
	}
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != 32 || !validID(id, "p_") || !filepath.IsAbs(dbPath) || !filepath.IsAbs(webPath) {
		return errors.New("registry authority metadata invalid")
	}
	var state string
	if err = db.QueryRow(`SELECT state FROM projects WHERE id=?`, id).Scan(&state); err != nil || state != "active" {
		return errors.New("registry authority default binding missing")
	}
	return nil
}
