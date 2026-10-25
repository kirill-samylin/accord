package store

import (
	"agent-board/internal/core"
	"context"
	"sort"
	"time"
)

func managedWorker(b *board, a core.Actor, in core.ManagedPollInput, epoch int64) (*managedHostRecord, error) {
	h, err := managedAllowed(b, a, in.HostID, "", true)
	if err != nil {
		return nil, err
	}
	if !managedProof(h, in.HostSecret) || h.Host.OwnerAccessEpoch != epoch {
		return nil, core.Fail("unauthorized", "Локальный host proof/epoch недействителен")
	}
	if h.Host.State != "active" {
		return nil, core.Fail("forbidden", "Компьютер отключён владельцем")
	}
	if !bounded(in.WorkerID, 128) {
		return nil, core.Fail("validation", "Нужен worker_id")
	}
	return h, nil
}

const managedInterruptedMessage = "Workflow остановлен после dispatch (пауза, отмена, вопрос или срок); ожидается подтверждённая остановка"

func managedPending(b *board, host string) []*managedCommandRecord {
	out := []*managedCommandRecord{}
	for _, c := range b.ManagedCommands {
		r := b.ManagedRuns[c.Command.RunID]
		// A needs_recovery run is resolved only by the owner's local `worker
		// recover` (recovered_stopped settles its commands); re-delivering its
		// stop first on every poll would starve every other run on the host.
		if r != nil && r.HostID == host && r.State != "needs_recovery" && (c.Command.Status == "queued" || c.Command.Status == "accepted" && r.Generation == c.Command.Generation || c.Command.Status == "applied" && managedActive(r.State) && r.Generation == c.Command.Generation) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, z := out[i].Command, out[j].Command
		if (a.Kind == "stop") != (z.Kind == "stop") {
			return a.Kind == "stop"
		}
		if a.RunID == z.RunID {
			return a.Sequence < z.Sequence
		}
		return a.CreatedAt.Before(z.CreatedAt)
	})
	return out
}
func managedBusy(b *board, host string) bool {
	for _, r := range b.ManagedRuns {
		if r.HostID == host && managedActive(r.State) {
			return true
		}
	}
	return false
}

// managedUncertain reports dispatched work whose local fate is unknown after a
// lease lapse or takeover. needs_recovery runs are already known-uncertain and
// fenced on their own (no dispatch, workspace blocked, `worker recover`), so
// they do not make ownership of every other run on the host uncertain.
func managedUncertain(b *board, host string) bool {
	for _, c := range b.ManagedCommands {
		r := b.ManagedRuns[c.Command.RunID]
		if r != nil && r.HostID == host && managedActive(r.State) && r.State != "needs_recovery" && r.Generation == c.Command.Generation && c.Command.DispatchID != "" {
			return true
		}
	}
	return false
}
func (s *Store) ManagedPoll(ctx context.Context, a core.Actor, in core.ManagedPollInput) (core.ManagedPollResult, error) {
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	proof := in
	proof.HostSecret = secret
	return managedMutate(s, ctx, a, in.RequestID, "poll", in, func(b *board, _ time.Time) error { _, err := managedWorker(b, a, proof, managedEpoch(ctx)); return err }, func(b *board, now time.Time) (core.ManagedPollResult, error) {
		var out core.ManagedPollResult
		h := b.ManagedHosts[in.HostID]
		previousWorker := h.WorkerID
		if previousWorker == in.WorkerID && previousWorker != "" && !h.Host.LeaseExpiresAt.After(now) && managedUncertain(b, in.HostID) {
			return out, core.Fail("managed_uncertain", "Ownership lease истекла; нужна локальная сверка остановки")
		}
		if h.WorkerID != "" && h.WorkerID != in.WorkerID {
			if h.Host.LeaseExpiresAt.After(now) {
				// Transient: another worker still holds a live lease.
				return out, core.Fail("lease_held", "worker ownership требует terminal recovery, takeover запрещён")
			}
			if managedUncertain(b, in.HostID) {
				return out, core.Fail("managed_uncertain", "worker ownership требует terminal recovery, takeover запрещён")
			}
			h.Host.WorkerEpoch++
		}
		if h.WorkerID == "" {
			for _, old := range b.ManagedCommands {
				run := b.ManagedRuns[old.Command.RunID]
				if run != nil && run.HostID == h.Host.ID && managedActive(run.State) && run.State != "needs_recovery" && old.Command.DispatchID != "" && old.WorkerID != in.WorkerID && (old.Command.Status == "accepted" || old.Command.Status == "applied") {
					return out, core.Fail("managed_uncertain", "Сначала сверяйте все процессы прежнего worker")
				}
			}
			h.Host.WorkerEpoch++
		}
		if previousWorker == in.WorkerID && in.WorkerEpoch != 0 && in.WorkerEpoch != h.Host.WorkerEpoch {
			return out, core.Fail("conflict", "worker epoch устарел")
		}
		h.Host.WorkerCapabilities = in.WorkerCapabilities
		h.WorkerID = in.WorkerID
		h.Host.LastSeenAt = now
		h.Host.LeaseExpiresAt = now.Add(30 * time.Second)
		out.Host = managedHostView(h, a)
		out.WorkerEpoch = h.Host.WorkerEpoch
		out.LeaseExpiresAt = h.Host.LeaseExpiresAt
		for _, c := range managedPending(b, h.Host.ID) {
			r := b.ManagedRuns[c.Command.RunID]
			if !managedCommandAllowed(ctx, b, h, c, now) {
				if c.Command.DispatchID != "" {
					// A workflow pause/cancel/question/timeout (state or control
					// epoch change) only adds an owned stop: the dispatched command
					// keeps its status so its own terminal report stays acceptable.
					// Lost initiator authority still makes delivery unknown.
					message := "Инициатор потерял доступ после dispatch; остановка требует подтверждения"
					if managedWorkflowInterrupted(ctx, b, h, c, now) {
						message = managedInterruptedMessage
					} else {
						c.Command.Status = "delivery_unknown"
					}
					if r.State != "stopping" || r.LastError != message {
						r.State = "stopping"
						r.LastError = message
						r.Version++
						r.UpdatedAt = now
					}
					var stop *managedCommandRecord
					for _, existing := range b.ManagedCommands {
						if existing.Command.RunID == r.ID && existing.Command.Kind == "stop" && (existing.Command.Status == "queued" || existing.Command.Status == "accepted") && managedCommandAllowed(ctx, b, h, existing, now) {
							stop = existing
							break
						}
					}
					if stop != nil && stop.Command.Status == "accepted" {
						return out, nil
					}
					var err error
					if stop == nil {
						stop, err = managedNewCommand(b, r, a, "stop", "", now, managedEpoch(ctx))
					}
					if err != nil {
						return out, err
					}
					id := stop.Command.DispatchID
					if id == "" {
						id, err = managedID("md_")
					} else if stop.WorkerID != in.WorkerID || stop.WorkerEpoch != h.Host.WorkerEpoch {
						return out, core.Fail("managed_uncertain", "Прежний stop требует recovery")
					}
					if err != nil {
						return out, err
					}
					stop.Command.DispatchID = id
					stop.WorkerID = in.WorkerID
					stop.WorkerEpoch = h.Host.WorkerEpoch
					cmd, run := stop.Command, *r
					out.Command = &cmd
					out.Run = &run
					return out, nil
				}
				c.Command.Status = "cancelled"
				if c.Command.Kind == "start" || c.Command.Kind == "resume" {
					r.State = "stopped"
					r.Version++
					r.UpdatedAt = now
				}
				continue
			}
			if c.Command.Status == "accepted" || c.Command.Status == "applied" {
				continue
			}
			if c.Command.DispatchID != "" {
				if c.WorkerID != in.WorkerID || c.WorkerEpoch != h.Host.WorkerEpoch {
					return out, core.Fail("managed_uncertain", "Нужна сверка прежнего dispatch")
				}
			} else {
				if c.Command.Kind == "clarify" {
					if r.State != "completed" || !c.Command.CreatedAt.Before(r.UpdatedAt) {
						continue
					}
					r.Generation++
					c.Command.Generation = r.Generation
				}
				if c.Command.Kind != "stop" {
					busy := false
					for _, other := range b.ManagedCommands {
						if other.Command.RunID == r.ID && other.Command.ID != c.Command.ID && other.Command.Status == "accepted" && other.Command.Generation == r.Generation {
							busy = true
						}
					}
					if busy {
						continue
					}
				}
				id, err := managedID("md_")
				if err != nil {
					return out, err
				}
				c.Command.DispatchID = id
				c.WorkerID = in.WorkerID
				c.WorkerEpoch = h.Host.WorkerEpoch
				if c.Command.Kind == "start" || c.Command.Kind == "resume" || c.Command.Kind == "clarify" {
					r.State = "starting"
					r.Version++
					r.UpdatedAt = now
				}
				if c.Command.Kind == "resume" {
					text := ""
					for _, prev := range managedPending(b, h.Host.ID) {
						if prev.Command.RunID == r.ID && prev.Command.Kind == "clarify" && prev.Command.Sequence < c.Command.Sequence && prev.Command.Status == "queued" {
							text += prev.Command.Text + "\n"
							prev.Command.Status = "accepted"
							prev.Command.DispatchID = id
							prev.Command.Generation = r.Generation
							prev.Command.AcceptedAt = &now
							prev.WorkerID = in.WorkerID
							prev.WorkerEpoch = h.Host.WorkerEpoch
						}
					}
					c.Command.Text = text + c.Command.Text
				}
			}
			cmd, run := c.Command, *r
			out.Command = &cmd
			out.Run = &run
			break
		}
		return out, nil
	})
}
