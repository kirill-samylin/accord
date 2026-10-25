package store

import (
	"agent-board/internal/core"
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

func managedContains(v []string, want string) bool {
	for _, x := range v {
		if x == want {
			return true
		}
	}
	return false
}
func managedSnapshot(b *board, in core.ManagedStartInput) (*core.ProfileSnapshot, error) {
	p, err := profileFor(b, in.ProfileID)
	if err != nil {
		return nil, err
	}
	if p.State != "active" || p.Version != in.ProfileVersion {
		return nil, core.Fail("conflict", "Профиль изменён или архивирован")
	}
	want := map[string]string{"codex": "openai", "claude": "anthropic"}[in.Provider]
	if want == "" || p.Provider != want && p.Provider != "unconfigured" {
		return nil, core.Fail("validation", "Поставщик профиля не совпадает с локальным CLI")
	}
	return &core.ProfileSnapshot{ID: p.ID, Version: p.Version, ProfileSettings: p.ProfileSettings}, nil
}
func managedNewCommand(b *board, r *core.ManagedRun, a core.Actor, kind, text string, now time.Time, epoch int64) (*managedCommandRecord, error) {
	var seq int64
	count, pending := 0, 0
	for _, c := range b.ManagedCommands {
		if c.Command.RunID != r.ID {
			continue
		}
		count++
		if c.Command.Sequence > seq {
			seq = c.Command.Sequence
		}
		if c.Command.Status == "queued" || c.Command.Status == "accepted" {
			pending++
		}
	}
	if kind != "stop" && (count >= 256 || pending >= 16) || kind == "stop" && count >= 257 {
		return nil, core.Fail("conflict", "Достигнут предел команд запуска")
	}
	id, err := managedID("mc_")
	if err != nil {
		return nil, err
	}
	c := &managedCommandRecord{CreatorEpoch: epoch, Command: core.ManagedCommand{ID: id, RunID: r.ID, Sequence: seq + 1, Generation: r.Generation, Kind: kind, Text: text, CreatedBy: a.Principal.ID, CreatedAt: now, Status: "queued"}}
	b.ManagedCommands[id] = c
	return c, nil
}
func (s *Store) ManagedStart(ctx context.Context, a core.Actor, in core.ManagedStartInput) (core.ManagedResult, error) {
	return managedMutate(s, ctx, a, in.RequestID, "start", in, func(b *board, _ time.Time) error {
		h, e := managedAllowed(b, a, in.HostID, "", false)
		if e == nil && h.Host.State != "active" {
			return core.Fail("forbidden", "Компьютер отключён")
		}
		return e
	}, func(b *board, now time.Time) (core.ManagedResult, error) {
		var out core.ManagedResult
		h := b.ManagedHosts[in.HostID]
		if !managedContains(h.Host.Workspaces, in.Workspace) || !managedContains(h.Host.Providers, in.Provider) {
			return out, core.Fail("forbidden", "Workspace/provider не разрешён локальным владельцем")
		}
		if err := textLimit(in.Prompt, 16384, "задание"); err != nil {
			return out, err
		}
		p, err := managedSnapshot(b, in)
		if err != nil {
			return out, err
		}
		if in.TaskKey != "" {
			t, err := taskFor(b, in.TaskKey)
			if err != nil {
				return out, err
			}
			if t.Role != p.Role {
				return out, core.Fail("validation", "Роль задачи не совпадает с профилем")
			}
		}
		active := 0
		for _, r := range b.ManagedRuns {
			if managedActive(r.State) {
				active++
				if r.HostID == in.HostID && r.Workspace == in.Workspace {
					return out, core.Fail("conflict", "Workspace уже занят или требует восстановления")
				}
			}
		}
		if b.Automation != nil {
			for _, reservation := range b.Automation.Reservations {
				if reservation.HostID == in.HostID && reservation.Workspace == in.Workspace {
					return out, core.Fail("conflict", "Workspace зарезервирован workflow")
				}
			}
		}
		if active+workflowReservedRuns(b) >= managedRunLiveLimit || active >= 8 {
			return out, core.Fail("conflict", "Достигнут предел активных и зарезервированных запусков")
		}
		if err := automationRotateHistory(b, now, a.SessionID); err != nil {
			return out, err
		}
		id, err := managedID("mr_")
		if err != nil {
			return out, err
		}
		r := &core.ManagedRun{ID: id, HostID: in.HostID, Version: 1, Provider: in.Provider, Workspace: in.Workspace, Prompt: in.Prompt, Profile: p, TaskKey: in.TaskKey, CreatedBy: a.Principal.ID, State: "queued", Generation: 1, CreatedAt: now, UpdatedAt: now, Output: []string{}}
		b.ManagedRuns[id] = r
		c, err := managedNewCommand(b, r, a, "start", in.Prompt, now, managedEpoch(ctx))
		if err != nil {
			return out, err
		}
		emit(b, now, "managed_start_queued", "", a.SessionID)
		return core.ManagedResult{Run: *r, Command: c.Command}, nil
	})
}
func (s *Store) ManagedCommand(ctx context.Context, a core.Actor, in core.ManagedCommandInput) (core.ManagedResult, error) {
	return managedMutate(s, ctx, a, in.RequestID, "command", in, func(b *board, _ time.Time) error { _, e := managedAllowed(b, a, "", in.RunID, false); return e }, func(b *board, now time.Time) (core.ManagedResult, error) {
		var out core.ManagedResult
		r := b.ManagedRuns[in.RunID]
		if r.Automation != nil {
			return out, core.Fail("forbidden", "Управляйте этим запуском через workflow controls")
		}
		h := b.ManagedHosts[r.HostID]
		if r.Version != in.Version {
			return out, core.Fail("conflict", "Версия запуска изменилась")
		}
		if h.Host.State != "active" && in.Kind != "stop" {
			return out, core.Fail("forbidden", "Компьютер отключён")
		}
		if len(in.Text) > 4096 || !utf8.ValidString(in.Text) {
			return out, core.Fail("validation", "Уточнение ограничено4КиБ")
		}
		switch in.Kind {
		case "stop":
			if !managedActive(r.State) {
				return out, core.Fail("conflict", "Процесс уже завершён")
			}
			for _, c := range b.ManagedCommands {
				if c.Command.RunID == r.ID && c.Command.Kind == "stop" && (c.Command.Status == "queued" || c.Command.Status == "accepted") {
					return out, core.Fail("conflict", "Остановка уже запрошена")
				}
			}
		case "resume":
			if r.State != "stopped" && r.State != "completed" {
				return out, core.Fail("conflict", "Сначала требуется подтверждённая остановка")
			}
			if r.ProviderSessionID == "" {
				return out, core.Fail("conflict", "resume_unavailable: нет native session ID")
			}
			for _, other := range b.ManagedRuns {
				if other.ID != r.ID && other.HostID == r.HostID && other.Workspace == r.Workspace && managedActive(other.State) {
					return out, core.Fail("conflict", "Workspace занят")
				}
			}
			r.Generation++
			r.State = "queued"
		case "clarify":
			if !bounded(in.Text, 4096) {
				return out, core.Fail("validation", "Введите уточнение")
			}
			if r.State == "needs_recovery" || r.State == "failed" || r.State == "stopping" {
				return out, core.Fail("conflict", "Запуск требует восстановления/остановки")
			}
		default:
			return out, core.Fail("validation", "Допустимы stop/resume/clarify")
		}
		if in.Kind == "stop" {
			undispatched := r.State == "queued"
			for _, c := range b.ManagedCommands {
				if c.Command.RunID == r.ID && c.Command.DispatchID != "" && (c.Command.Status == "queued" || c.Command.Status == "accepted") {
					undispatched = false
				}
			}
			if undispatched {
				for _, c := range b.ManagedCommands {
					if c.Command.RunID == r.ID && c.Command.Status == "queued" {
						c.Command.Status = "cancelled"
					}
				}
				r.State = "stopped"
			} else {
				r.State = "stopping"
			}
		}
		text := strings.TrimSpace(in.Text)
		if in.Kind == "resume" && text == "" {
			text = "Продолжай выполнение задания с учётом уже полученных уточнений."
		}
		c, err := managedNewCommand(b, r, a, in.Kind, text, now, managedEpoch(ctx))
		if err != nil {
			return out, err
		}
		if in.Kind == "stop" && r.State == "stopped" {
			c.Command.Status = "applied"
			c.Command.AppliedAt = &now
		}
		r.Version++
		r.UpdatedAt = now
		emit(b, now, "managed_"+in.Kind+"_queued", "", a.SessionID)
		return core.ManagedResult{Run: *r, Command: c.Command}, nil
	})
}
