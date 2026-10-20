package projects

import (
	"agent-board/internal/core"
	"context"
)

func (s *service) Next(ctx context.Context, a core.Actor, in core.NextInput) (core.Claim, error) {
	return actorCall(s, ctx, a, in, true, s.board.Next)
}
func (s *service) Create(ctx context.Context, a core.Actor, in core.CreateInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Create)
}
func (s *service) Update(ctx context.Context, a core.Actor, in core.UpdateInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Update)
}
func (s *service) Transition(ctx context.Context, a core.Actor, in core.TransitionInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Transition)
}
func (s *service) Release(ctx context.Context, a core.Actor, in core.LeaseInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Release)
}
func (s *service) Comment(ctx context.Context, a core.Actor, in core.CommentInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Comment)
}
func (s *service) Link(ctx context.Context, a core.Actor, in core.LinkInput) (core.Task, error) {
	return actorCall(s, ctx, a, in, true, s.board.Link)
}
func (s *service) Heartbeat(ctx context.Context, a core.Actor, in core.HeartbeatInput) (core.SessionView, error) {
	return actorCall(s, ctx, a, in, true, s.board.Heartbeat)
}
func (s *service) Note(ctx context.Context, a core.Actor, in core.NoteInput) (core.Note, error) {
	return actorCall(s, ctx, a, in, true, s.board.Note)
}
