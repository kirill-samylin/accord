package store

import (
	"agent-board/internal/core"
	"context"
	"time"
)

func managedSessionRequested(in core.OpenSession) bool {
	return in.ManagedRunID != "" || in.ManagedGeneration != 0
}

// Only an executing, worker-fenced run may bind its authoritative immutable
// snapshot. This exception never accepts an arbitrary historical profile and
// does not change assignment eligibility or transfer an existing task lease.
func managedSessionProfile(ctx context.Context, b *board, p core.Principal, in core.OpenSession, now time.Time) (*core.ProfileSnapshot, error) {
	if in.ManagedRunID == "" || in.ManagedGeneration < 1 || in.RunID != in.ManagedRunID {
		return nil, core.Fail("validation", "Нужны точные managed run_id/generation")
	}
	r, err := managedRun(b, in.ManagedRunID)
	if err != nil {
		return nil, err
	}
	h, err := managedHost(b, r.HostID)
	if err != nil {
		return nil, err
	}
	if h.Host.OwnerPrincipalID != p.ID || h.Host.OwnerAccessEpoch != managedEpoch(ctx) || h.Host.State != "active" {
		return nil, core.Fail("forbidden", "Managed сессия требует текущего владельца компьютера")
	}
	if r.Generation != in.ManagedGeneration || r.State != "starting" && r.State != "running" || !h.Host.LeaseExpiresAt.After(now) || h.WorkerID == "" {
		return nil, core.Fail("conflict", "Managed попытка не исполняется под действующей worker lease")
	}
	dispatched := false
	for _, c := range b.ManagedCommands {
		v := c.Command
		if v.RunID == r.ID && v.Generation == r.Generation && v.DispatchID != "" && (v.Kind == "start" || v.Kind == "resume" || v.Kind == "clarify") && (v.Status == "accepted" || v.Status == "applied") && c.WorkerID == h.WorkerID && c.WorkerEpoch == h.Host.WorkerEpoch {
			dispatched = true
			break
		}
	}
	if !dispatched {
		return nil, core.Fail("conflict", "Нет подтверждённой execution-команды текущей попытки")
	}
	saved := r.Profile
	if saved == nil || in.ProfileID != saved.ID || in.ProfileVersion != saved.Version || in.Role != saved.Role {
		return nil, core.Fail("validation", "Профиль и роль должны совпадать с immutable managed snapshot")
	}
	copy := *saved
	return &copy, nil
}
