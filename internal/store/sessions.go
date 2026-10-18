package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func (s *Store) Describe(ctx context.Context, p core.Principal, in core.OpenSession) (core.SessionView, error) {
	return mutate(s, ctx, core.Actor{Principal: p}, in.RequestID, "describe", in, func(b *board, _ *core.Session, now time.Time) (core.SessionView, error) {
		var zero core.SessionView
		if !validRole(b, in.Role) || !bounded(in.RunID, 128) || in.Machine != p.Machine {
			return zero, core.Fail("invalid_input", "valid role, run_id and token machine required")
		}
		if err := roleAuthorized(ctx, b, p, in.Role); err != nil {
			return zero, err
		}
		if err := toolAllowed(b, in.Role, "describe"); err != nil {
			return zero, err
		}
		if singleton(b, in.Role) {
			for _, sess := range b.Sessions {
				if sess.Role == in.Role && sessionLive(b, sess, now) {
					return zero, core.Fail("role_held", "singleton role has an active session")
				}
			}
		}
		id, err := identifier()
		if err != nil {
			return zero, err
		}
		generation := int64(0)
		if singleton(b, in.Role) {
			b.Roles[in.Role]++
			generation = b.Roles[in.Role]
		}
		v := core.SessionView{SessionID: id, Role: in.Role, RoleGeneration: generation, ExpiresAt: now.Add(b.Config.SessionTTL), Project: b.Config.Project}
		if singleton(b, in.Role) {
			expiry := now.Add(b.Config.RoleTTL)
			v.RoleExpiresAt = &expiry
		}
		sessionTiming(b, &v)
		b.dirtySessions[id] = true
		b.Sessions[id] = &core.Session{SessionView: v, PrincipalID: p.ID, RunID: in.RunID, Machine: in.Machine, Owner: p.Owner}
		emit(b, now, "session_opened", "", id)
		sess := b.Sessions[id]
		sess.Order = b.Events[len(b.Events)-1].Cursor
		v.Notes = scopeNotes(b, "role", in.Role)
		v.Rules = ruleHints(b)
		v.MyTasks = []core.Task{}
		v.Queue = sessionQueueSummary(b, sess, now)
		return v, nil
	})
}
func (s *Store) Heartbeat(ctx context.Context, a core.Actor, in core.HeartbeatInput) (core.SessionView, error) {
	return mutate(s, ctx, a, in.RequestID, "heartbeat", in, func(b *board, sess *core.Session, now time.Time) (core.SessionView, error) {
		if hasRoleLease(sess) && in.RoleGeneration != sess.RoleGeneration {
			return core.SessionView{}, core.Fail("lease_lost", "singleton generation does not match")
		}
		var active *core.Task
		for _, t := range b.ownedTasks(a.SessionID) {
			if t.Lease != nil && t.Lease.SessionID == a.SessionID {
				active = t
			}
		}
		if active != nil && (in.TaskKey != active.Key || in.Generation != active.Generation) {
			return core.SessionView{}, core.Fail("lease_lost", "heartbeat must acknowledge current task generation")
		}
		if in.TaskKey != "" {
			t, err := owned(b, a, in.TaskKey, in.Generation)
			if err != nil {
				return core.SessionView{}, err
			}
			renew(b, t, now)
		}
		sess.ExpiresAt = now.Add(b.Config.SessionTTL)
		if hasRoleLease(sess) {
			expiry := now.Add(b.Config.RoleTTL)
			sess.RoleExpiresAt = &expiry
		}
		sessionTiming(b, &sess.SessionView)
		return sess.SessionView, nil
	})
}
