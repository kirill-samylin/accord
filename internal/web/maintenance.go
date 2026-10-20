package web

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// ResetOffline invalidates restored browser state while retaining design assets
// and audit history. It must only run against offline staged restoration data.
func ResetOffline(ctx context.Context, dir string) error {
	info, err := os.Lstat(filepath.Join(dir, "web.db"))
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("existing private web database required")
	}
	db, err := openWebDB(dir)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM browser_sessions; DELETE FROM web_actions; DELETE FROM web_action_epochs; DELETE FROM preauth; INSERT INTO web_audit(event,principal,at) VALUES('offline_restore','',0);`); err != nil {
		return err
	}
	return tx.Commit()
}
