package projects

import (
	"agent-board/internal/core"
	"context"
)

func (s *service) Profiles(ctx context.Context, a core.Actor, in core.ProfilesInput) (core.ProfilesPage, error) {
	return actorCall(s, ctx, a, in, false, s.board.Profiles)
}
func (s *service) Profile(ctx context.Context, a core.Actor, in core.ProfileInput) (core.ProfilePage, error) {
	return actorCall(s, ctx, a, in, false, s.board.Profile)
}
func (s *service) ProfileCreate(ctx context.Context, a core.Actor, in core.ProfileCreateInput) (core.AgentProfile, error) {
	return actorCall(s, ctx, a, in, true, s.board.ProfileCreate)
}
func (s *service) ProfileUpdate(ctx context.Context, a core.Actor, in core.ProfileUpdateInput) (core.AgentProfile, error) {
	return actorCall(s, ctx, a, in, true, s.board.ProfileUpdate)
}
func (s *service) ProfileArchive(ctx context.Context, a core.Actor, in core.ProfileArchiveInput) (core.AgentProfile, error) {
	return actorCall(s, ctx, a, in, true, s.board.ProfileArchive)
}
