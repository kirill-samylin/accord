package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"
)

func automationClone(b *board) (*board, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	var c board
	if err = json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.tx = b.tx
	c.EventBase = b.EventBase
	return &c, nil
}

// Authorization precedes replay; a request ID is stable across fresh sessions.
func automationMutate[T any](s *Store, ctx context.Context, a core.Actor, request, method string, input any, check func(*board, time.Time) error, fn func(*board, time.Time) (T, error)) (T, error) {
	var zero T
	if !bounded(request, 128) {
		return zero, core.Fail("validation", "Нужен request_id до128байт")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, err
	}
	hash := managedHash(method + "\n" + string(raw))
	return transact(s, ctx, func(b *board, now time.Time) (T, error) {
		requiresWrite := true
		if in, ok := input.(core.WorkflowAttestInput); ok && automationTerminalAttest(in.Kind) {
			requiresWrite = false
		}
		_, p, err := automationActor(ctx, b, a, now, requiresWrite)
		if err != nil {
			return zero, err
		}
		if err = check(b, now); err != nil {
			return zero, err
		}
		capKey := ""
		if ident, ok := ExecutionIdentityFromContext(ctx); ok {
			capKey = "/" + ident.Capability.ID
		}
		key := a.Principal.ID + "/automation/" + strconv.FormatInt(p.Epoch, 10) + capKey + "/" + request
		var oldHash, saved string
		err = b.tx.QueryRowContext(ctx, `SELECT hash,response FROM idempotency WHERE key=?`, key).Scan(&oldHash, &saved)
		if err == nil {
			if oldHash != hash {
				return zero, core.Fail("idempotency_conflict", "request_id использован с другим содержимым")
			}
			var out T
			err = json.Unmarshal([]byte(saved), &out)
			return out, err
		}
		if err != sql.ErrNoRows {
			return zero, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return zero, err
		}
		c, err := automationClone(b)
		if err != nil {
			return zero, err
		}
		out, err := fn(c, now)
		if err != nil {
			return zero, err
		}
		reconcileEpics(c, now, a.SessionID)
		data, err := json.Marshal(out)
		if err != nil {
			return zero, err
		}
		if _, err = b.tx.ExecContext(ctx, `INSERT INTO idempotency VALUES(?,?,?)`, key, hash, string(data)); err != nil {
			return zero, err
		}
		c.Sessions[a.SessionID].ExpiresAt = now.Add(s.cfg.SessionTTL)
		*b = *c
		return out, nil
	})
}
