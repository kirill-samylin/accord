package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type catalogActionKey struct{}
type catalogAction struct {
	ID, Result string
	Identity   Identity
}

func finishCatalog(ctx context.Context, tx *sql.Tx) error {
	a, ok := ctx.Value(catalogActionKey{}).(catalogAction)
	if !ok {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE actions SET result=? WHERE id=?`, a.Result, a.ID)
	return err
}
func (r *Registry) issue(ctx context.Context, b browserIdentity, operation, pid string) (string, error) {
	return transaction(r, ctx, func(tx *sql.Tx) (string, error) {
		if e := cleanupCatalog(ctx, tx); e != nil {
			return "", e
		}
		epoch := int64(0)
		if pid != "" {
			a, e := r.access(ctx, tx, b.Identity, pid, false)
			if e != nil {
				return "", e
			}
			epoch = a.member.Epoch
		}
		// Retain bound actions for the browser lifetime and cap every session.
		var count int
		if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM actions WHERE session_hash=?`, b.Hash).Scan(&count); e != nil {
			return "", e
		}
		if count >= 2048 {
			if _, e := tx.ExecContext(ctx, `DELETE FROM actions WHERE id IN (SELECT id FROM actions WHERE session_hash=? AND payload='' ORDER BY created LIMIT 128)`, b.Hash); e != nil {
				return "", e
			}
			if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM actions WHERE session_hash=?`, b.Hash).Scan(&count); e != nil {
				return "", e
			}
			if count >= 2048 {
				return "", core.Fail("conflict", "browser action capacity reached; sign in again")
			}
		}
		id, e := random("", 32)
		if e != nil {
			return "", e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO actions(id,session_hash,operation,project_id,access_epoch,created) VALUES(?,?,?,?,?,?)`, id, b.Hash, operation, pid, epoch, time.Now().UnixNano())
		return id, e
	})
}
func (r *Registry) bindCatalog(ctx context.Context, b browserIdentity, id, operation, pid string, payload any, manage bool) (string, error) {
	raw, e := json.Marshal(payload)
	if e != nil {
		return "", e
	}
	return transaction(r, ctx, func(tx *sql.Tx) (string, error) {
		var identity, csrf string
		var expires int64
		e := tx.QueryRowContext(ctx, `SELECT identity_id,csrf,expires FROM sessions WHERE hash=?`, b.Hash).Scan(&identity, &csrf, &expires)
		if e != nil || identity != b.Identity.ID || expires <= time.Now().UnixNano() {
			return "", core.Fail("unauthorized", "session ended")
		}
		epoch := int64(0)
		if pid != "" {
			a, e := r.access(ctx, tx, b.Identity, pid, false)
			if e != nil {
				return "", e
			}
			if manage && !a.project.CanManage {
				return "", denied()
			}
			epoch = a.member.Epoch
		}
		var owner, op, target, old, result string
		var issued int64
		e = tx.QueryRowContext(ctx, `SELECT session_hash,operation,project_id,access_epoch,payload,result FROM actions WHERE id=?`, id).Scan(&owner, &op, &target, &issued, &old, &result)
		if e != nil || owner != b.Hash || op != operation || target != pid || issued != epoch {
			return "", core.Fail("idempotency_conflict", "refresh project form")
		}
		if old != "" && old != string(raw) {
			return "", core.Fail("idempotency_conflict", "action fields changed")
		}
		if old == "" {
			_, e = tx.ExecContext(ctx, `UPDATE actions SET payload=? WHERE id=?`, string(raw), id)
		}
		return result, e
	})
}
func actionResult(target string) string {
	raw, _ := json.Marshal(map[string]any{"ok": true, "redirect": target})
	return string(raw)
}

// Recheck inside the business transaction, before mutation. Binding the payload
// in an earlier transaction does not by itself serialize duplicate submissions.
type catalogReplay struct{ Result string }

func (e *catalogReplay) Error() string { return "catalog action already committed" }
func (r *Registry) catalogGate(ctx context.Context, tx *sql.Tx) error {
	a, ok := ctx.Value(catalogActionKey{}).(catalogAction)
	if !ok {
		return nil
	}
	var pid, op, owner, result string
	var epoch int64
	if err := tx.QueryRowContext(ctx, `SELECT project_id,operation,session_hash,access_epoch,result FROM actions WHERE id=?`, a.ID).Scan(&pid, &op, &owner, &epoch, &result); err != nil {
		return denied()
	}
	var identity string
	var expires int64
	if err := tx.QueryRowContext(ctx, `SELECT identity_id,expires FROM sessions WHERE hash=?`, owner).Scan(&identity, &expires); err != nil || identity != a.Identity.ID || expires <= time.Now().UnixNano() {
		return core.Fail("unauthorized", "session ended")
	}
	if _, err := r.identity(ctx, tx, identity); err != nil {
		return err
	}
	if pid != "" {
		access, err := r.access(ctx, tx, a.Identity, pid, false)
		if err != nil {
			return err
		}
		if access.member.Epoch != epoch {
			return denied()
		}
		if op != "switch:"+pid && !access.project.CanManage {
			return denied()
		}
	}
	if result != "" {
		return &catalogReplay{Result: result}
	}
	return nil
}

// Keep browser metadata bounded even when a client only signs in and reads a
// project. Native session epoch mappings are durable replay state and survive.
func cleanupCatalog(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().UnixNano()
	for _, statement := range []string{`DELETE FROM actions WHERE session_hash IN (SELECT hash FROM sessions WHERE expires<=?)`, `DELETE FROM sessions WHERE expires<=?`, `DELETE FROM preauth WHERE expires<=?`, `DELETE FROM session_tombstones WHERE expires<=?`} {
		if _, err := tx.ExecContext(ctx, statement, now); err != nil {
			return err
		}
	}
	return nil
}
