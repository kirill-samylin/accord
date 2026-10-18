package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func (s *Store) Release(ctx context.Context, a core.Actor, in core.LeaseInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "release", in, func(b *board, _ *core.Session, now time.Time) (core.Task, error) {
		t, err := owned(b, a, in.TaskKey, in.Generation)
		if err != nil {
			return core.Task{}, err
		}
		if !bounded(in.Reason, 4096) || len(in.Branch) > 512 {
			return core.Task{}, core.Fail("invalid_input", "release requires reason and bounded branch")
		}
		finishAttempt(t, now, in.Reason, in.Branch)
		t.Lease = nil
		t.State = "ready"
		t.ReadyAt = now
		t.Version++
		emit(b, now, "released", t.Key, a.SessionID)
		return viewTask(b, t), nil
	})
}
func (s *Store) Comment(ctx context.Context, a core.Actor, in core.CommentInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "comment", in, func(b *board, _ *core.Session, now time.Time) (core.Task, error) {
		t, err := owned(b, a, in.TaskKey, in.Generation)
		if err != nil {
			return core.Task{}, err
		}
		if !bounded(in.Text, 4096) {
			return core.Task{}, core.Fail("invalid_input", "comment must contain 1..4096 bytes")
		}
		if err = addTokens(t, in.TokensSpentDelta, in.Branch); err != nil {
			return core.Task{}, err
		}
		t.Comments = append(t.Comments, core.Comment{Text: in.Text, SessionID: a.SessionID, At: now})
		t.Version++
		s.renew(t, now)
		emit(b, now, "commented", t.Key, a.SessionID)
		if in.TokensSpentDelta != nil {
			delta := *in.TokensSpentDelta
			b.Events[len(b.Events)-1].Data = &core.EventData{TokensSpentDelta: &delta}
		}
		return viewTask(b, t), nil
	})
}
