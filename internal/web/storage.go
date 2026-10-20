package web

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func openWebDB(dir string) (*sql.DB, error) {
	if dir == "" {
		return nil, fmt.Errorf("private web data directory required")
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	// Reject every existing symlink component before creating private directories.
	for p := absolute; ; p = filepath.Dir(p) {
		i, e := os.Lstat(p)
		if e == nil && (i.Mode()&os.ModeSymlink != 0 || !i.IsDir() || i.Mode().Perm()&0022 != 0 && p != "/private/tmp" && p != "/tmp") {
			return nil, fmt.Errorf("unsafe web data parent")
		}
		if e != nil && !os.IsNotExist(e) {
			return nil, e
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		return nil, err
	}
	i, err := os.Lstat(absolute)
	if err != nil || !i.IsDir() || i.Mode().Perm() != 0700 {
		return nil, fmt.Errorf("web data directory must have mode 0700")
	}
	path := filepath.Join(absolute, "web.db")
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		i, e := os.Lstat(path + suffix)
		if e == nil && (!i.Mode().IsRegular() || i.Mode().Perm() != 0600) {
			return nil, fmt.Errorf("unsafe web database or sidecar")
		}
		if e != nil && !os.IsNotExist(e) {
			return nil, e
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		err = f.Close()
	} else if os.IsExist(err) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: path}).String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var schema int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&schema); err != nil {
		db.Close()
		return nil, err
	}
	if schema != 0 && schema != 1 && schema != 2 {
		db.Close()
		return nil, fmt.Errorf("unsupported web database schema")
	}
	_, err = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; BEGIN IMMEDIATE;
 CREATE TABLE IF NOT EXISTS browser_sessions(hash TEXT PRIMARY KEY,token_id TEXT NOT NULL,csrf TEXT NOT NULL,expires INTEGER NOT NULL,board_session TEXT NOT NULL,pending_describe TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS preauth(hash TEXT PRIMARY KEY,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS web_actions(id TEXT PRIMARY KEY,session_hash TEXT NOT NULL,operation TEXT NOT NULL,payload TEXT NOT NULL DEFAULT '',session_id TEXT NOT NULL DEFAULT '',result TEXT NOT NULL DEFAULT '',created INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS web_action_epochs(id TEXT PRIMARY KEY,epoch INTEGER NOT NULL);
 CREATE TRIGGER IF NOT EXISTS web_action_epoch_cleanup AFTER DELETE ON web_actions BEGIN DELETE FROM web_action_epochs WHERE id=OLD.id; END;
 DELETE FROM web_action_epochs WHERE id NOT IN (SELECT id FROM web_actions);
 CREATE INDEX IF NOT EXISTS web_actions_session ON web_actions(session_hash);
 CREATE TABLE IF NOT EXISTS web_audit(id INTEGER PRIMARY KEY,event TEXT NOT NULL,principal TEXT NOT NULL,at INTEGER NOT NULL);
 PRAGMA user_version=2; COMMIT;`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func (s *Server) cleanupLoop() {
	defer close(s.stopped)
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.cleanup()
			_ = s.designs.cleanup(context.Background())
		}
	}
}
func (s *Server) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	now := time.Now().UnixNano()
	if _, err = tx.Exec(`DELETE FROM web_actions WHERE session_hash IN (SELECT hash FROM browser_sessions WHERE expires<=?)`, now); err != nil {
		return
	}
	if _, err = tx.Exec(`DELETE FROM browser_sessions WHERE expires<=?`, now); err != nil {
		return
	}
	if _, err = tx.Exec(`DELETE FROM preauth WHERE expires<=?`, now); err != nil {
		return
	}
	_ = tx.Commit()
}
