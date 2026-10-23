package projects

import (
	"agent-board/internal/core"
	"context"
)

func (s *service) Get(ctx context.Context, a core.Actor, in core.GetInput) (core.Task, error) {
	return nativeActorCall(s, ctx, a, in, false, "get", s.board.Get)
}
func (s *service) Events(ctx context.Context, a core.Actor, in core.EventsInput) (core.EventsPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "events", s.board.Events)
}
func (s *service) ListTasks(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "list_tasks", s.board.ListTasks)
}
func (s *service) ListEpics(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "list_epics", s.board.ListEpics)
}
func (s *service) Sessions(ctx context.Context, a core.Actor, in core.ListInput) (core.SessionsPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "sessions", s.board.Sessions)
}
func (s *service) ListNotes(ctx context.Context, a core.Actor, in core.NotesInput) (core.NotesPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "list_notes", s.board.ListNotes)
}
func (s *service) Stats(ctx context.Context, a core.Actor, in core.StatsInput) (core.Stats, error) {
	return nativeActorCall(s, ctx, a, in, false, "stats", s.board.Stats)
}
func (s *service) ControlOverview(ctx context.Context, a core.Actor) (core.ControlOverview, error) {
	if err := automationOperationAllowed(ctx, "control_overview"); err != nil {
		return core.ControlOverview{}, err
	}
	return withAccess(s, ctx, false, a.SessionID, func(ctx context.Context, access access) (core.ControlOverview, error) {
		if a.Principal != access.principal {
			return core.ControlOverview{}, denied()
		}
		return s.board.ControlOverview(ctx, a)
	})
}
func (s *service) ControlSessions(ctx context.Context, a core.Actor, in core.ControlSessionsInput) (core.ControlSessionsPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "control_sessions", s.board.ControlSessions)
}
func (s *service) ControlTasks(ctx context.Context, a core.Actor, in core.ControlTasksInput) (core.TasksPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "control_tasks", s.board.ControlTasks)
}
func (s *service) RecentEvents(ctx context.Context, a core.Actor, in core.RecentEventsInput) (core.RecentEventsPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "recent_events", s.board.RecentEvents)
}
func (s *service) ListDecisions(ctx context.Context, a core.Actor, in core.DecisionsInput) (core.DecisionsPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "list_decisions", s.board.ListDecisions)
}
func (s *service) ListQuestions(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "list_questions", s.board.ListQuestions)
}

func (s *service) Similar(ctx context.Context, a core.Actor, in core.SimilarInput) (core.SimilarPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "similar", s.board.Similar)
}
func (s *service) ControlTeam(ctx context.Context, a core.Actor) (core.TeamView, error) {
	if err := automationOperationAllowed(ctx, "team"); err != nil {
		return core.TeamView{}, err
	}
	return withAccess(s, ctx, false, a.SessionID, func(ctx context.Context, access access) (core.TeamView, error) {
		if a.Principal != access.principal {
			return core.TeamView{}, denied()
		}
		return s.board.ControlTeam(ctx, a)
	})
}
