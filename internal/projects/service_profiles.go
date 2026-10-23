package projects

import (
	"agent-board/internal/core"
	"context"
)

func (s *service) Profiles(ctx context.Context, a core.Actor, in core.ProfilesInput) (core.ProfilesPage, error) {
	return nativeActorCall(s, ctx, a, in, false, "profiles", s.board.Profiles)
}
func (s *service) Profile(ctx context.Context, a core.Actor, in core.ProfileInput) (core.ProfilePage, error) {
	return nativeActorCall(s, ctx, a, in, false, "profile", s.board.Profile)
}
func (s *service) ProfileCreate(ctx context.Context, a core.Actor, in core.ProfileCreateInput) (core.AgentProfile, error) {
	return nativeActorCall(s, ctx, a, in, true, "profile_create", s.board.ProfileCreate)
}
func (s *service) ProfileUpdate(ctx context.Context, a core.Actor, in core.ProfileUpdateInput) (core.AgentProfile, error) {
	return nativeActorCall(s, ctx, a, in, true, "profile_update", s.board.ProfileUpdate)
}
func (s *service) ProfileArchive(ctx context.Context, a core.Actor, in core.ProfileArchiveInput) (core.AgentProfile, error) {
	return nativeActorCall(s, ctx, a, in, true, "profile_archive", s.board.ProfileArchive)
}
