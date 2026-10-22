package store

import (
	"agent-board/internal/core"
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var managedSessionID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var managedBearer = regexp.MustCompile(`(?i)(bearer\s+|(?:api[_-]?key|token|secret)\s*[:=]\s*)[^\s,;]+`)

func managedClean(s string) string {
	s = managedBearer.ReplaceAllString(s, "[redacted]")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
}
func managedOutput(r *core.ManagedRun, in core.ManagedReportInput) error {
	bytes := 0
	for _, line := range in.Output {
		bytes += len(line)
		if len(line) > 4096 || !utf8.ValidString(line) {
			return core.Fail("validation", "Событие ограничено4КиБ UTF8")
		}
	}
	if bytes > 16384 || len(in.Output) > 64 || len(in.Error) > 4096 || !utf8.ValidString(in.Error) {
		return core.Fail("validation", "Отчёт превышает лимит")
	}
	for _, line := range in.Output {
		r.Output = append(r.Output, managedClean(line))
	}
	bytes = 0
	for _, line := range r.Output {
		bytes += len(line)
	}
	for bytes > 65536 && len(r.Output) > 0 {
		bytes -= len(r.Output[0])
		r.Output = r.Output[1:]
		r.OutputTruncated = true
	}
	return nil
}
func managedReportAuthorize(ctx context.Context, b *board, a core.Actor, in core.ManagedReportInput, now time.Time) (*managedHostRecord, *core.ManagedRun, *managedCommandRecord, error) {
	h, err := managedWorker(b, a, in.ManagedPollInput, managedEpoch(ctx))
	if err != nil {
		return nil, nil, nil, err
	}
	r, err := managedRun(b, in.RunID)
	if err != nil {
		return nil, nil, nil, err
	}
	c := b.ManagedCommands[in.CommandID]
	if c == nil || r.HostID != h.Host.ID || c.Command.RunID != r.ID || c.Command.DispatchID == "" || c.Command.DispatchID != in.DispatchID || c.Command.Generation != in.Generation {
		return nil, nil, nil, core.Fail("conflict", "Чужой или устаревший dispatch")
	}
	if (in.Phase == "accepted" || in.Phase == "started") && !managedCommandAllowed(ctx, b, h, c, now) {
		return nil, nil, nil, core.Fail("forbidden", "Инициатор команды потерял разрешение управления")
	}
	return h, r, c, nil
}
func managedReportCheck(ctx context.Context, b *board, a core.Actor, in core.ManagedReportInput, now time.Time) (*managedHostRecord, *core.ManagedRun, *managedCommandRecord, error) {
	h, r, c, err := managedReportAuthorize(ctx, b, a, in, now)
	if err != nil {
		return nil, nil, nil, err
	}
	if r.Generation != in.Generation {
		return nil, nil, nil, core.Fail("conflict", "Устаревшая generation запуска")
	}
	if in.Phase == "recovered_stopped" || in.Phase == "delivery_unknown" {
		if !bounded(in.WorkerID, 128) {
			return nil, nil, nil, core.Fail("validation", "Нужен worker ID")
		}
		return h, r, c, nil
	}
	if h.WorkerID != in.WorkerID || h.Host.WorkerEpoch != in.WorkerEpoch || c.WorkerID != in.WorkerID || c.WorkerEpoch != in.WorkerEpoch || !h.Host.LeaseExpiresAt.After(now) {
		return nil, nil, nil, core.Fail("conflict", "worker ownership lease потеряна")
	}
	return h, r, c, nil
}
func (s *Store) ManagedReport(ctx context.Context, a core.Actor, in core.ManagedReportInput) (core.ManagedResult, error) {
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	proof := in
	proof.HostSecret = secret
	return managedMutate(s, ctx, a, in.RequestID, "report", in, func(b *board, now time.Time) error {
		_, _, _, err := managedReportAuthorize(ctx, b, a, proof, now)
		return err
	}, func(b *board, now time.Time) (core.ManagedResult, error) {
		var out core.ManagedResult
		h, r, c, err := managedReportCheck(ctx, b, a, proof, now)
		if err != nil {
			return out, err
		}
		if err = managedOutput(r, in); err != nil {
			return out, err
		}
		terminal := c.Command.Status == "failed" || c.Command.Status == "cancelled" || c.Command.Status == "delivery_unknown"
		if terminal && in.Phase != "recovered_stopped" {
			return out, core.Fail("conflict", "Команда уже terminal")
		}
		switch in.Phase {
		case "accepted":
			if c.Command.Status != "queued" {
				return out, core.Fail("conflict", "receipt уже подтверждён")
			}
			c.Command.Status = "accepted"
			c.Command.AcceptedAt = &now
		case "started":
			if c.Command.Status != "accepted" || c.Command.Kind == "stop" || r.Generation != in.Generation {
				return out, core.Fail("conflict", "Запуск не соответствует accepted command")
			}
			if !managedSessionID.MatchString(in.ProviderSessionID) || r.ProviderSessionID != "" && r.ProviderSessionID != in.ProviderSessionID {
				return out, core.Fail("conflict", "Native session ID не совпадает")
			}
			r.ProviderSessionID = in.ProviderSessionID
			if r.State != "stopping" {
				r.State = "running"
			}
			c.Command.Status = "applied"
			c.Command.AppliedAt = &now
		case "completed":
			if c.Command.Status != "applied" || c.Command.Kind == "stop" || r.Generation != in.Generation {
				return out, core.Fail("conflict", "Нет подтверждённого запущенного turn")
			}
			if r.State != "running" && r.State != "stopping" {
				return out, core.Fail("conflict", "Turn уже завершён")
			}
			if r.State != "stopping" {
				r.State = "completed"
			}
		case "stopped":
			if c.Command.Kind != "stop" || c.Command.Status != "accepted" {
				return out, core.Fail("conflict", "Нет accepted stop")
			}
			r.State = "stopped"
			c.Command.Status = "applied"
			c.Command.AppliedAt = &now
			c.Command.Forced = in.Forced
			for _, old := range b.ManagedCommands {
				if old.Command.RunID == r.ID && old.Command.ID != c.Command.ID && (old.Command.Status == "accepted" || old.Command.Status == "queued" && old.Command.DispatchID != "") {
					old.Command.Status = "cancelled"
				}
			}
		case "failed":
			if c.Command.Status != "accepted" && c.Command.Status != "applied" {
				return out, core.Fail("conflict", "Нет принятой команды")
			}
			c.Command.Status = "failed"
			c.Command.Error = managedClean(in.Error)
			r.State = "failed"
			r.LastError = c.Command.Error
		case "delivery_unknown":
			c.Command.Status = "delivery_unknown"
			c.Command.Error = managedClean(in.Error)
			r.State = "needs_recovery"
			r.LastError = c.Command.Error
		case "recovered_stopped":
			if h.Host.LeaseExpiresAt.After(now) && h.WorkerID != in.WorkerID {
				return out, core.Fail("conflict", "Прежний worker ещё владеет lease")
			}
			r.State = "stopped"
			c.Command.Status = "cancelled"
			c.Command.Forced = c.Command.Forced || in.Forced
			// One local tree exit reconciles this execution attempt, including a
			// concurrent stop receipt. Undispatched clarification remains mailbox.
			for _, peer := range b.ManagedCommands {
				if peer.Command.RunID == r.ID && peer.Command.Generation == in.Generation && peer.Command.ID != c.Command.ID && (peer.Command.Status == "accepted" || peer.Command.Status == "queued" && (peer.Command.DispatchID != "" || peer.Command.Kind != "clarify")) {
					peer.Command.Status = "cancelled"
				}
			}
			h.WorkerID = ""
			h.Host.LeaseExpiresAt = now
		default:
			return out, core.Fail("validation", "Неизвестная фаза worker report")
		}
		if in.Phase == "started" || in.Phase == "failed" {
			for _, related := range b.ManagedCommands {
				if related.Command.ID != c.Command.ID && related.Command.DispatchID == c.Command.DispatchID && related.Command.Status == "accepted" {
					related.Command.Status = c.Command.Status
					related.Command.AppliedAt = c.Command.AppliedAt
					related.Command.Error = c.Command.Error
				}
			}
		}
		r.Version++
		r.UpdatedAt = now
		if in.Phase != "recovered_stopped" && in.Phase != "delivery_unknown" {
			h.Host.LastSeenAt = now
			h.Host.LeaseExpiresAt = now.Add(30 * time.Second)
		}
		emit(b, now, "managed_report_"+in.Phase, "", a.SessionID)
		return core.ManagedResult{Run: *r, Command: c.Command}, nil
	})
}
