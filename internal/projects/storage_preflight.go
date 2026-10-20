package projects

import (
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

// Preflight precedes every default board/web constructor in multiproject serve.
// An activated authority or published storage is never reconstructed from empty
// SQLite files; web orphan recovery must not run after metadata loss.
func Preflight(root, defaultDB, defaultWeb string) error {
	if root == "" {
		return nil
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	marker := filepath.Join(absolute, "MULTIPROJECT_REQUIRES_CAPABILITY")
	info, err := os.Lstat(marker)
	if os.IsNotExist(err) {
		entries, e := os.ReadDir(absolute)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if len(entries) != 0 {
			return errors.New("unmarked registry state requires inspection")
		}
		return validateFreshDefault(defaultDB, defaultWeb)
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return errors.New("unsafe registry activation marker")
	}
	db, err := readPrivateSQLite(filepath.Join(absolute, "registry.sqlite"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err = validateRegistry(db); err != nil {
		return err
	}
	var id, boundDB, boundWeb string
	if err = db.QueryRow(`SELECT default_id,default_db,default_web FROM registry_meta WHERE id=1`).Scan(&id, &boundDB, &boundWeb); err != nil {
		return err
	}
	expectedDB, e := filepath.Abs(defaultDB)
	if e != nil {
		return e
	}
	expectedWeb, e := filepath.Abs(defaultWeb)
	if e != nil {
		return e
	}
	if boundDB != expectedDB || boundWeb != expectedWeb {
		return errors.New("registry default paths changed")
	}
	rows, err := db.Query(`SELECT id FROM projects WHERE state IN ('active','archived')`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var pid string
		if err = rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, pid)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, pid := range ids {
		if !validID(pid, "p_") {
			return errors.New("invalid published project inventory")
		}
		board, web := boundDB, boundWeb
		if pid != id {
			base := filepath.Join(absolute, "projects", pid)
			board = filepath.Join(base, "board.db")
			web = filepath.Join(base, "web")
		}
		if err = validateProjectStorage(board, web); err != nil {
			return err
		}
	}
	return nil
}
func validateFreshDefault(board, web string) error {
	if _, err := os.Lstat(board); err == nil {
		if err = validateBoardStorage(board); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(filepath.Join(web, "web.db")); err == nil {
		return validateWebStorage(web)
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(web)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) != 0 {
		return errors.New("default web metadata missing; preserve storage for inspection")
	}
	return nil
}
func readPrivateSQLite(path string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for parent := filepath.Dir(absolute); ; parent = filepath.Dir(parent) {
		info, e := os.Lstat(parent)
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 && parent != "/tmp" && parent != "/private/tmp" {
			return nil, errors.New("unsafe existing storage path")
		}
		if parent == filepath.Dir(parent) {
			break
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, e := os.Lstat(absolute + suffix)
		if e == nil && (!info.Mode().IsRegular() || info.Mode().Perm() != 0600) {
			return nil, errors.New("unsafe existing database")
		}
		if e != nil && (!os.IsNotExist(e) || suffix == "") {
			return nil, errors.New("existing database required")
		}
		if suffix == "" && info != nil && info.Size() == 0 {
			return nil, errors.New("existing database is empty")
		}
	}
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: absolute, RawQuery: "mode=ro"}).String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var integrity string
	if err = db.QueryRow(`PRAGMA quick_check`).Scan(&integrity); err != nil || integrity != "ok" {
		db.Close()
		return nil, errors.New("existing database integrity failure")
	}
	return db, nil
}
func requireColumns(db *sql.DB, queries []string) error {
	for _, query := range queries {
		rows, err := db.Query(query)
		if err != nil {
			return errors.New("published storage schema incomplete")
		}
		rows.Close()
	}
	return nil
}
func validateProjectStorage(board, web string) error {
	if err := validateBoardStorage(board); err != nil {
		return err
	}
	return validateWebStorage(web)
}
func validateBoardStorage(path string) error {
	db, err := readPrivateSQLite(path)
	if err != nil {
		return err
	}
	defer db.Close()
	var version int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 2 {
		return errors.New("published board schema unsupported")
	}
	if err = requireColumns(db, []string{`SELECT id,sequence FROM metadata LIMIT 0`, `SELECT key,data FROM tasks LIMIT 0`, `SELECT key,data FROM sessions LIMIT 0`, `SELECT role,generation FROM role_generations LIMIT 0`, `SELECT cursor,data FROM events LIMIT 0`, `SELECT key,hash,response FROM idempotency LIMIT 0`, `SELECT id,hash,owner,machine,expires,revoked FROM tokens LIMIT 0`, `SELECT key,data FROM notes LIMIT 0`, `SELECT id,sequence FROM note_metadata LIMIT 0`}); err != nil {
		return err
	}
	var sequence int64
	return db.QueryRow(`SELECT sequence FROM metadata WHERE id=1`).Scan(&sequence)
}
func validateWebStorage(dir string) error {
	db, err := readPrivateSQLite(filepath.Join(dir, "web.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	var version int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 1 && version != 2 {
		return errors.New("published web schema unsupported")
	}
	queries := []string{`SELECT hash,token_id,csrf,expires,board_session,pending_describe FROM browser_sessions LIMIT 0`, `SELECT hash,csrf,expires FROM preauth LIMIT 0`, `SELECT id,session_hash,operation,payload,session_id,result,created FROM web_actions LIMIT 0`, `SELECT id,event,principal,at FROM web_audit LIMIT 0`, `SELECT seq,asset_id,design_id,version,scope,scope_key,title,filename,media_type,size,sha256,uploader_principal,uploader_owner,created_at,status,request_id,metadata_hash,blob,preview,preview_size,reserved FROM design_assets LIMIT 0`}
	if version == 2 {
		queries = append(queries, `SELECT id,epoch FROM web_action_epochs LIMIT 0`)
	}
	return requireColumns(db, queries)
}
