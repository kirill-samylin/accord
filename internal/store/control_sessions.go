package store

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"agent-board/internal/core"
)

type controlCredential struct {
	owner string
	valid bool
}

// controlCredentials intentionally batches token validity without exposing IDs.
func controlCredentials(ctx context.Context, b *board, now time.Time) (map[string]controlCredential, error) {
	rows, err := b.tx.QueryContext(ctx, `SELECT id,owner,expires,revoked FROM tokens`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]controlCredential{}
	for rows.Next() {
		var id, owner string
		var expires int64
		var revoked bool
		if err := rows.Scan(&id, &owner, &expires, &revoked); err != nil {
			return nil, err
		}
		result[id] = controlCredential{owner: owner, valid: !revoked && expires > now.UnixNano()}
	}
	return result, rows.Err()
}
func controlStatus(sess *core.Session, credentials map[string]controlCredential, now time.Time) string {
	if !credentials[sess.PrincipalID].valid {
		return "credential_invalid"
	}
	if !sess.ExpiresAt.After(now) {
		return "expired"
	}
	return "active"
}
func controlLease(b *board, t *core.Task, credentials map[string]controlCredential, now time.Time) bool {
	if t.Type != "task" || t.State != "in_progress" || t.Lease == nil || !t.Lease.ExpiresAt.After(now) || t.Lease.Generation != t.Generation {
		return false
	}
	sess := b.Sessions[t.Lease.SessionID]
	return sess != nil && controlStatus(sess, credentials, now) == "active" && (!core.Singleton(sess.Role) || b.Roles[sess.Role] == sess.RoleGeneration)
}
func validateControlSessions(in core.ControlSessionsInput) error {
	if in.Limit < 0 || in.Limit > 100 || in.Role != "" && !core.ValidRole(in.Role) || len(in.Owner) > 128 || in.SessionID != "" && !core.ValidControlSessionID(in.SessionID) {
		return core.Fail("validation", "Неверный фильтр сессий")
	}
	if in.Kind != "" && in.Kind != "all" && in.Kind != "agent" && in.Kind != "human" {
		return core.Fail("validation", "Неверный тип сессий")
	}
	if in.Role != "" && (in.Kind == "agent" && in.Role == "human" || in.Kind == "human" && in.Role != "human") {
		return core.Fail("validation", "Роль не соответствует типу сессии")
	}
	if in.State != "" && in.State != "active" && in.State != "expired" && in.State != "credential_invalid" {
		return core.Fail("validation", "Неверное состояние сессии")
	}
	return nil
}

// ControlSessions leaves native TTL-only Sessions semantics unchanged.
func (s *Store) ControlSessions(ctx context.Context, a core.Actor, in core.ControlSessionsInput) (core.ControlSessionsPage, error) {
	if in.Kind == "all" {
		in.Kind = ""
	}
	if err := validateControlSessions(in); err != nil {
		return core.ControlSessionsPage{}, err
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.ControlSessionsPage, error) {
		page := core.ControlSessionsPage{ObservedAt: now, Sessions: []core.ControlSession{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		credentials, err := controlCredentials(ctx, b, now)
		if err != nil {
			return page, err
		}
		orders, err := sessionOrders(ctx, b)
		if err != nil {
			return page, err
		}
		lastEvents, err := controlLastEvents(ctx, b)
		if err != nil {
			return page, err
		}
		sessions := make([]*core.Session, 0, len(b.Sessions))
		var high int64
		for _, sess := range b.Sessions {
			v := *sess
			if v.Order == 0 {
				v.Order = orders[v.SessionID]
			}
			if v.Order > high {
				high = v.Order
			}
			sessions = append(sessions, &v)
		}
		sort.Slice(sessions, func(i, j int) bool { return sessions[i].Order < sessions[j].Order })
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		c, limit, err := pageCursor("control_sessions", filter, in.Cursor, in.Limit, high)
		if err != nil {
			return page, err
		}
		tasks := map[string]*core.Task{}
		for _, t := range b.Tasks {
			if controlLease(b, t, credentials, now) {
				v := viewTask(b, t)
				tasks[t.Lease.SessionID] = &v
			}
		}
		last := c.After
		more := false
		for _, sess := range sessions {
			if sess.Order <= c.After || sess.Order > c.High {
				continue
			}
			status := controlStatus(sess, credentials, now)
			owner := sess.Owner
			if owner == "" {
				owner = credentials[sess.PrincipalID].owner
			}
			if in.SessionID != "" && in.SessionID != sess.SessionID || in.Role != "" && in.Role != sess.Role || in.Owner != "" && in.Owner != owner {
				continue
			}
			if in.Kind == "agent" && sess.Role == "human" || in.Kind == "human" && sess.Role != "human" {
				continue
			}
			if in.State != "" && in.State != status || in.State == "" && !in.IncludeClosed && status != "active" {
				continue
			}
			if len(page.Sessions) == limit {
				more = true
				break
			}
			page.Sessions = append(page.Sessions, core.ControlSession{SessionID: sess.SessionID, Role: sess.Role, RunID: sess.RunID, Machine: sess.Machine, Owner: owner, RoleGeneration: sess.RoleGeneration, ExpiresAt: sess.ExpiresAt, Status: status, Task: tasks[sess.SessionID], LastEventAt: lastEvents[sess.SessionID]})
			last = sess.Order
		}
		page.PageInfo = finishPage(c, last, more)
		return page, nil
	})
}
func controlLastEvents(ctx context.Context, b *board) (map[string]*time.Time, error) {
	// Select the event with the largest immutable cursor, not a derived last_seen.
	rows, err := b.tx.QueryContext(ctx, `SELECT e.data FROM events e JOIN (SELECT max(cursor) AS cursor FROM events WHERE json_extract(data,'$.session_id') IS NOT NULL GROUP BY json_extract(data,'$.session_id')) last ON e.cursor=last.cursor`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]*time.Time{}
	for rows.Next() {
		var raw []byte
		var e core.Event
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		at := e.At
		result[e.SessionID] = &at
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, e := range b.Events {
		if e.SessionID != "" {
			at := e.At
			result[e.SessionID] = &at
		}
	}
	return result, nil
}
