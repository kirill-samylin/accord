package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func (s *Store) Describe(ctx context.Context, p core.Principal, in core.OpenSession) (core.SessionView, error) {
	return mutate(s, ctx, core.Actor{Principal: p}, in.RequestID, "describe", in, func(b *board, _ *core.Session, now time.Time) (core.SessionView, error) {
		var zero core.SessionView
		if !core.ValidRole(in.Role) || !bounded(in.RunID, 128) || in.Machine != p.Machine {
			return zero, core.Fail("invalid_input", "valid role, run_id and token machine required")
		}
		profile, err := sessionProfile(b, in)
		if err != nil {
			return zero, err
		}
		if core.Singleton(in.Role) {
			for _, sess := range b.Sessions {
				if sess.Role == in.Role && sess.ExpiresAt.After(now) {
					return zero, core.Fail("role_held", "singleton role has an active session")
				}
			}
		}
		id, err := identifier()
		if err != nil {
			return zero, err
		}
		generation := int64(0)
		if core.Singleton(in.Role) {
			b.Roles[in.Role]++
			generation = b.Roles[in.Role]
		}
		v := core.SessionView{Profile: profile, SessionID: id, Role: in.Role, RoleGeneration: generation, ExpiresAt: now.Add(s.cfg.SessionTTL), Project: s.cfg.Project}
		s.sessionTiming(&v)
		b.Sessions[id] = &core.Session{SessionView: v, PrincipalID: p.ID, RunID: in.RunID, Machine: in.Machine, Owner: p.Owner}
		emit(b, now, "session_opened", "", id)
		sess := b.Sessions[id]
		sess.Order = b.Events[len(b.Events)-1].Cursor
		v.Notes = scopeNotes(b, "role", in.Role)
		v.Rules = map[string]string{"independent_review": "block", "one_task_per_agent": "block", "cycle_check": "block", "singleton_role": "block"}
		v.MyTasks = []core.Task{}
		v.Queue = sessionQueueSummary(b, sess)
		return v, nil
	})
}
func (s *Store) Heartbeat(ctx context.Context, a core.Actor, in core.HeartbeatInput) (core.SessionView, error) {
	return mutate(s, ctx, a, in.RequestID, "heartbeat", in, func(b *board, sess *core.Session, now time.Time) (core.SessionView, error) {
		if core.Singleton(sess.Role) && in.RoleGeneration != sess.RoleGeneration {
			return core.SessionView{}, core.Fail("lease_lost", "singleton generation does not match")
		}
		var active *core.Task
		for _, t := range b.Tasks {
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
			s.renew(t, now)
		}
		sess.ExpiresAt = now.Add(s.cfg.SessionTTL)
		s.sessionTiming(&sess.SessionView)
		view := sess.SessionView
		view.Queue = sessionQueueSummary(b, sess)
		return view, nil
	})
}

func (s *Store) sessionTiming(v *core.SessionView) {
	v.SessionTTLSeconds = s.cfg.SessionTTL.Seconds()
	v.TaskTTLSeconds = s.cfg.LeaseTTL.Seconds()
	v.HeartbeatEverySeconds = min(s.cfg.SessionTTL, s.cfg.LeaseTTL).Seconds() / 3
}
