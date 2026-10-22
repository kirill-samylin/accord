package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

type managedHostRecord struct {
	Host       core.ManagedHost `json:"host"`
	SecretHash string           `json:"secret_hash"`
	WorkerID   string           `json:"worker_id"`
}
type managedCommandRecord struct {
	Command      core.ManagedCommand `json:"command"`
	WorkerID     string              `json:"worker_id"`
	WorkerEpoch  int64               `json:"worker_epoch"`
	CreatorEpoch int64               `json:"creator_epoch"`
}
type managedEpochKey struct{}
type managedPrincipalsKey struct{}

type ManagedPrincipalGrant struct {
	Epoch         int64
	LegacyTokenID string
}

func WithManagedPrincipals(ctx context.Context, ids map[string]ManagedPrincipalGrant) context.Context {
	return context.WithValue(ctx, managedPrincipalsKey{}, ids)
}
func managedCreatorAllowed(ctx context.Context, b *board, h *managedHostRecord, id string, epoch int64, now time.Time) bool {
	if !managedControl(h, id) {
		return false
	}
	return managedPrincipalAllowed(ctx, b, id, epoch, now)
}
func managedCommandAllowed(ctx context.Context, b *board, h *managedHostRecord, c *managedCommandRecord, now time.Time) bool {
	if !managedCreatorAllowed(ctx, b, h, c.Command.CreatedBy, c.CreatorEpoch, now) {
		return false
	}
	if c.Command.DispatchID != "" {
		for _, related := range b.ManagedCommands {
			if related.Command.DispatchID == c.Command.DispatchID && (related.Command.Status == "accepted" || related.Command.Status == "applied") && !managedCreatorAllowed(ctx, b, h, related.Command.CreatedBy, related.CreatorEpoch, now) {
				return false
			}
		}
	}
	return true
}
func managedPrincipalAllowed(ctx context.Context, b *board, id string, epoch int64, now time.Time) bool {
	if ids, ok := ctx.Value(managedPrincipalsKey{}).(map[string]ManagedPrincipalGrant); ok {
		grant, exists := ids[id]
		if !exists || grant.Epoch != epoch {
			return false
		}
		if grant.LegacyTokenID == "" {
			return true
		}
		id = grant.LegacyTokenID
	}
	var expiry int64
	var revoked int
	return b.tx.QueryRowContext(ctx, `SELECT expires,revoked FROM tokens WHERE id=?`, id).Scan(&expiry, &revoked) == nil && revoked == 0 && time.Unix(0, expiry).After(now)
}

// WithManagedEpoch carries the registry-authoritative owner epoch. Clients
// cannot supply this value; single-board installations retain epoch zero.
func WithManagedEpoch(ctx context.Context, epoch int64) context.Context {
	return context.WithValue(ctx, managedEpochKey{}, epoch)
}
func managedEpoch(ctx context.Context) int64 { v, _ := ctx.Value(managedEpochKey{}).(int64); return v }
func managedHash(secret string) string {
	v := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(v[:])
}
func managedProof(h *managedHostRecord, secret string) bool {
	return len(secret) == 64 && subtle.ConstantTimeCompare([]byte(h.SecretHash), []byte(managedHash(secret))) == 1
}
func managedControl(h *managedHostRecord, p string) bool {
	if h.Host.OwnerPrincipalID == p {
		return true
	}
	for _, id := range h.Host.Controllers {
		if id == p {
			return true
		}
	}
	return false
}
func managedHost(b *board, id string) (*managedHostRecord, error) {
	h := b.ManagedHosts[id]
	if h == nil {
		return nil, core.Fail("not_found", "Компьютер не найден")
	}
	return h, nil
}
func managedRun(b *board, id string) (*core.ManagedRun, error) {
	r := b.ManagedRuns[id]
	if r == nil {
		return nil, core.Fail("not_found", "Запуск не найден")
	}
	return r, nil
}
func managedActor(ctx context.Context, b *board, a core.Actor, now time.Time) error {
	if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
		return err
	}
	s := b.Sessions[a.SessionID]
	if s == nil || s.PrincipalID != a.Principal.ID {
		return core.Fail("unauthorized", "Сессия не принадлежит credential")
	}
	if s.Role != "human" {
		return core.Fail("forbidden", "Управление требует отдельную human-сессию")
	}
	return nil
}

// managedMutate checks current host permission before idempotent replay too.
// Payload secrets are hashed in-place before hashing; no raw secret is persisted.
func managedMutate[T any](s *Store, ctx context.Context, a core.Actor, request, method string, input any, check func(*board, time.Time) error, fn func(*board, time.Time) (T, error)) (T, error) {
	var zero T
	if strings.TrimSpace(request) == "" || len(request) > 128 {
		return zero, core.Fail("validation", "Нужен request_id до128символов")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, err
	}
	hash := managedHash(method + "\n" + string(raw))
	return transact(s, ctx, func(b *board, now time.Time) (T, error) {
		if err := managedActor(ctx, b, a, now); err != nil {
			return zero, err
		}
		if err := check(b, now); err != nil {
			return zero, err
		}
		host, run := managedTarget(input)
		if run != "" {
			r, e := managedRun(b, run)
			if e != nil {
				return zero, e
			}
			host = r.HostID
		}
		if host != "" {
			h, e := managedHost(b, host)
			if e != nil {
				return zero, e
			}
			if !managedPrincipalAllowed(ctx, b, h.Host.OwnerPrincipalID, h.Host.OwnerAccessEpoch, now) {
				return zero, core.Fail("unauthorized", "Владелец компьютера потерял доступ")
			}
		}
		key := a.Principal.ID + "/" + a.SessionID + "/managed/" + request
		var oldHash, saved string
		err := b.tx.QueryRowContext(ctx, `SELECT hash,response FROM idempotency WHERE key=?`, key).Scan(&oldHash, &saved)
		if err == nil {
			if oldHash != hash {
				return zero, core.Fail("idempotency_conflict", "Повтор содержит другие поля")
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
		data, err := json.Marshal(b)
		if err != nil {
			return zero, err
		}
		var candidate board
		if err = json.Unmarshal(data, &candidate); err != nil {
			return zero, err
		}
		candidate.tx = b.tx
		candidate.EventBase = b.EventBase
		out, err := fn(&candidate, now)
		if err != nil {
			return zero, err
		}
		candidate.Sessions[a.SessionID].ExpiresAt = now.Add(s.cfg.SessionTTL)
		response, err := json.Marshal(out)
		if err != nil {
			return zero, err
		}
		if _, err = b.tx.ExecContext(ctx, `INSERT INTO idempotency VALUES(?,?,?)`, key, hash, string(response)); err != nil {
			return zero, err
		}
		*b = candidate
		return out, nil
	})
}
func managedID(prefix string) (string, error) {
	id, err := identifier()
	if err != nil {
		return "", err
	}
	return prefix + id[:32], nil
}
func managedHostView(h *managedHostRecord, a core.Actor) core.ManagedHost {
	v := h.Host
	v.CanControl = managedControl(h, a.Principal.ID)
	return v
}
func managedAllowed(b *board, a core.Actor, host, run string, owner bool) (*managedHostRecord, error) {
	if run != "" {
		r, err := managedRun(b, run)
		if err != nil {
			return nil, err
		}
		host = r.HostID
	}
	h, err := managedHost(b, host)
	if err != nil {
		return nil, err
	}
	if owner && h.Host.OwnerPrincipalID != a.Principal.ID || !owner && !managedControl(h, a.Principal.ID) {
		return nil, core.Fail("forbidden", "Нет локального разрешения владельца компьютера")
	}
	return h, nil
}
func managedActive(state string) bool {
	return state == "queued" || state == "starting" || state == "running" || state == "stopping" || state == "needs_recovery"
}
