package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func (s *Store) Config(ctx context.Context) (core.ConfigView, error) {
	var v core.ConfigView
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT h.version,h.sha256,h.canonical_json FROM config_history h JOIN config_active a ON h.version=a.version WHERE a.id=1`).Scan(&v.ConfigVersion, &v.SHA256, &raw)
	v.Config = raw
	return v, err
}
func (s *Store) ConfigHistory(ctx context.Context, after int64, limit int) (core.ConfigHistory, error) {
	return configHistory(ctx, s.db, after, limit)
}

type historyQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func configHistory(ctx context.Context, q historyQuerier, after int64, limit int) (core.ConfigHistory, error) {
	out := core.ConfigHistory{Revisions: []core.ConfigRevision{}}
	if after < 0 || limit < 1 || limit > 200 {
		return out, core.Fail("VALIDATION", "history requires after>=0 and limit 1..200")
	}
	rows, err := q.QueryContext(ctx, `SELECT version,sha256,canonical_json,actor,applied_at FROM config_history WHERE version>? ORDER BY version LIMIT ?`, after, limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v core.ConfigRevision
		var at string
		var raw []byte
		if err = rows.Scan(&v.ConfigVersion, &v.SHA256, &raw, &v.Actor, &at); err != nil {
			return out, err
		}
		v.Config = raw
		v.AppliedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return out, err
		}
		if len(out.Revisions) == limit {
			out.HasMore = true
			break
		}
		out.Revisions = append(out.Revisions, v)
		out.NextVersion = v.ConfigVersion
	}
	return out, rows.Err()
}

// ApplyConfig is local administration only. Parsing is repeated here so callers
// cannot substitute an unvalidated candidate. CAS precedes board safety checks.
func (s *Store) ApplyConfig(ctx context.Context, raw []byte, expected int64, actor string) (core.ConfigView, error) {
	var out core.ConfigView
	if !bounded(actor, 128) || expected < 0 {
		return out, core.Fail("VALIDATION", "actor and expected-version required")
	}
	snapshot, result := config.Parse(raw, config.Options{})
	if !result.Valid {
		p := result.Errors[0]
		return out, core.Fail(p.Code, p.Path+": "+p.Message)
	}
	err := s.adminTransaction(ctx, func(b *board, now time.Time) error {
		if b.Config.ConfigVersion != expected {
			return core.Fail("CONFIG_CONFLICT", "active configuration version differs")
		}
		if b.Config.Snapshot != nil && b.Config.Snapshot.SHA256() == snapshot.SHA256() {
			emit(b, now, "config_applied", "", "")
			b.Events[len(b.Events)-1].Reason = fmt.Sprintf("version=%d hash=%s actor=%s no_op=true", expected, snapshot.SHA256(), actor)
			out = core.ConfigView{ConfigVersion: expected, SHA256: snapshot.SHA256(), Config: snapshot.CanonicalJSON()}
			return nil
		}
		if err := configInUse(b, snapshot, now); err != nil {
			return err
		}
		version := expected + 1
		if _, err := b.tx.ExecContext(ctx, `INSERT INTO config_history VALUES(?,?,?,?,?,?)`, version, snapshot.SHA256(), string(snapshot.CanonicalJSON()), string(raw), actor, now.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if _, err := b.tx.ExecContext(ctx, `INSERT INTO config_active(id,version) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET version=excluded.version`, version); err != nil {
			return err
		}
		b.Config = runtimeConfig(snapshot, version)
		emit(b, now, "config_applied", "", "")
		b.Events[len(b.Events)-1].Reason = fmt.Sprintf("version=%d hash=%s actor=%s", version, snapshot.SHA256(), actor)
		out = core.ConfigView{ConfigVersion: version, SHA256: snapshot.SHA256(), Config: snapshot.CanonicalJSON()}
		return nil
	})
	return out, err
}

func equalJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}
func configInUse(b *board, snapshot *config.Snapshot, now time.Time) error {
	old, next := b.Config.Workflow, snapshot.Decode()
	keys := map[string]bool{}
	changed := !equalJSON(old.Leases, next.Leases) || !equalJSON(old.Roles, next.Roles) || !equalJSON(old.Reopen, next.Reopen) || !equalJSON(old.Rules, next.Rules) || !equalJSON(old.Areas, next.Areas) || !equalJSON(old.MCP, next.MCP) || !equalJSON(old.Auth, next.Auth)
	for _, sess := range b.Sessions {
		if sessionLive(b, sess, now) && changed {
			keys["session:"+sess.SessionID] = true
		}
	}
	for _, t := range b.Tasks {
		_, exists := next.Roles[t.Role]
		if t.Type == "task" && !terminalState(t.State) && !exists {
			keys["task:"+t.Key] = true
		}
		if t.Lease != nil && t.Lease.ExpiresAt.After(now) && changed {
			keys["task:"+t.Key] = true
		}
	}
	counts, bytes := map[string]int{}, map[string]int{}
	for _, n := range b.Notes {
		if n.ScopeType == "role" {
			if _, exists := next.Roles[n.ScopeID]; !exists {
				keys["note:"+n.Key] = true
			}
		}
		if n.DeletedAt != nil {
			continue
		}
		scope := n.ScopeType + ":" + n.ScopeID
		counts[scope]++
		bytes[scope] += len(n.Text)
	}
	for _, n := range b.Notes {
		if n.DeletedAt != nil {
			continue
		}
		countCap, byteCap := next.Notes.MaxPerRole, next.Notes.MaxBytesPerRole
		if n.ScopeType == "epic" {
			countCap, byteCap = next.Notes.MaxPerEpic, next.Notes.MaxBytesPerEpic
		}
		scope := n.ScopeType + ":" + n.ScopeID
		if counts[scope] > countCap || bytes[scope] > byteCap {
			keys["note:"+n.Key] = true
		}
	}
	if len(keys) > 0 {
		return core.Fail("CONFIG_IN_USE", "configuration affects "+strings.Join(sortedKeys(keys), ","))
	}
	return nil
}
func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

// Config reads authenticated HTTP state under one writer-serialized snapshot.
func (s *Store) ReadConfig(ctx context.Context, p core.Principal) (core.ConfigView, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ConfigView, error) {
		if err := authenticatePrincipal(ctx, b, p, now); err != nil {
			return core.ConfigView{}, err
		}
		return core.ConfigView{ConfigVersion: b.Config.ConfigVersion, SHA256: b.Config.Snapshot.SHA256(), Config: b.Config.Snapshot.CanonicalJSON()}, nil
	})
}

func (s *Store) ReadConfigHistory(ctx context.Context, p core.Principal, after int64, limit int) (core.ConfigHistory, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ConfigHistory, error) {
		if err := authenticatePrincipal(ctx, b, p, now); err != nil {
			return core.ConfigHistory{}, err
		}
		return configHistory(ctx, b.tx, after, limit)
	})
}
