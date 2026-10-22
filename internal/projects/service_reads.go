package projects

import (
	"agent-board/internal/core"
	"context"
)

func (s *service) Get(ctx context.Context, a core.Actor, in core.GetInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, false, s.board.Get)
}
func (s *service) Events(ctx context.Context, a core.Actor, in core.EventsInput) (core.EventsPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.Events)
}
func (s *service) ListTasks(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ListTasks)
}
func (s *service) ListEpics(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ListEpics)
}
func (s *service) Sessions(ctx context.Context, a core.Actor, in core.ListInput) (core.SessionsPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.Sessions)
}
func (s *service) ListNotes(ctx context.Context, a core.Actor, in core.NotesInput) (core.NotesPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ListNotes)
}
func (s *service) Stats(ctx context.Context, a core.Actor, in core.StatsInput) (core.Stats, error) {
	return actorCall(s, ctx, a, in, false, s.board.Stats)
}
func (s *service) ControlOverview(ctx context.Context, a core.Actor) (core.ControlOverview, error) {
	return withAccess(s, ctx, false, a.SessionID, func(ctx context.Context, access access) (core.ControlOverview, error) {
		if a.Principal != access.principal {
			return core.ControlOverview{}, denied()
		}
		return s.board.ControlOverview(ctx, a)
	})
}
func (s *service) ControlSessions(ctx context.Context, a core.Actor, in core.ControlSessionsInput) (core.ControlSessionsPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ControlSessions)
}
func (s *service) ControlTasks(ctx context.Context, a core.Actor, in core.ControlTasksInput) (core.TasksPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ControlTasks)
}
func (s *service) RecentEvents(ctx context.Context, a core.Actor, in core.RecentEventsInput) (core.RecentEventsPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.RecentEvents)
}
func (s *service) ListDecisions(ctx context.Context, a core.Actor, in core.DecisionsInput) (core.DecisionsPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ListDecisions)
}
func (s *service) ListQuestions(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.ListQuestions)
}

func (s *service) Similar(ctx context.Context, a core.Actor, in core.SimilarInput) (core.SimilarPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.Similar)
}
func (s *service) ControlTeam(ctx context.Context, a core.Actor) (core.TeamView, error) {
	return withAccess(s, ctx, false, a.SessionID, func(ctx context.Context, access access) (core.TeamView, error) {
		if a.Principal != access.principal {
			return core.TeamView{}, denied()
		}
		return s.board.ControlTeam(ctx, a)
	})
}
