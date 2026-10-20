package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"strings"
	"time"
)

type service struct {
	registry  *Registry
	projectID string
	board     *store.Store
}

func (s *service) Authenticate(ctx context.Context, secret string) (core.Principal, error) {
	who, err := s.registry.Authenticate(ctx, secret)
	if err != nil {
		return core.Principal{}, err
	}
	if expected, e := accountFrom(ctx); e != nil || expected.ID != who.ID {
		return core.Principal{}, denied()
	}
	return withAccess(s, ctx, false, "", func(_ context.Context, a access) (core.Principal, error) { return a.principal, nil })
}
func (s *service) AuthenticateID(ctx context.Context, id string) (core.Principal, time.Time, error) {
	a, err := withAccess(s, ctx, false, "", func(_ context.Context, a access) (access, error) {
		if id != a.principal.ID {
			return a, denied()
		}
		return a, nil
	})
	return a.principal, a.identity.Expires, err
}
func (s *service) Describe(ctx context.Context, p core.Principal, in core.OpenSession) (core.SessionView, error) {
	if strings.TrimSpace(in.RequestID) == "" || len(in.RequestID) > 128 {
		return core.SessionView{}, invalid()
	}
	return withAccess(s, ctx, false, "", func(ctx context.Context, a access) (core.SessionView, error) {
		if p != a.principal || (a.member.Role == "viewer" || a.project.State != "active") && in.Role != "human" {
			return core.SessionView{}, denied()
		}
		in.RequestID = s.registry.describeID(a, in.RequestID)
		view, err := s.board.Describe(ctx, p, in)
		if err != nil {
			return view, err
		}
		_, err = a.tx.ExecContext(ctx, `INSERT INTO session_epochs(project_id,session_id,identity_id,access_epoch) VALUES(?,?,?,?) ON CONFLICT(project_id,session_id) DO NOTHING`, a.project.ID, view.SessionID, a.identity.ID, a.member.Epoch)
		if err == nil {
			err = s.registry.sessionAllowed(ctx, a.tx, a, view.SessionID)
		}
		return view, err
	})
}
func actorCall[I, O any](s *service, ctx context.Context, a core.Actor, in I, write bool, call func(context.Context, core.Actor, I) (O, error)) (O, error) {
	return withAccess(s, ctx, write, a.SessionID, func(ctx context.Context, access access) (O, error) {
		var zero O
		if a.Principal != access.principal {
			return zero, denied()
		}
		return call(ctx, a, in)
	})
}
func (s *service) CurrentProject(ctx context.Context) (*core.ProjectView, error) {
	project, err := withAccess(s, ctx, false, "", func(_ context.Context, a access) (Project, error) { return a.project, nil })
	return &project, err
}
func (s *service) Epoch(ctx context.Context) (int64, error) {
	return withAccess(s, ctx, false, "", func(_ context.Context, a access) (int64, error) { return a.member.Epoch, nil })
}
func (s *service) SessionAllowed(ctx context.Context, id string) error {
	_, err := withAccess(s, ctx, false, id, func(context.Context, access) (bool, error) { return true, nil })
	return err
}
func (s *service) Guard(ctx context.Context, write bool, session string, call func() error) error {
	_, err := withAccess(s, ctx, write, session, func(_ context.Context, a access) (bool, error) {
		if a.identity.Source == "legacy" && s.projectID == s.registry.defaultID {
			return true, s.registry.options.Default.GuardCredential(ctx, a.principal, call)
		}
		return true, call()
	})
	return err
}
