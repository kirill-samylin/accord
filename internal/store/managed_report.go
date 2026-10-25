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
	r := b.ManagedRuns[in.RunID]
	if r == nil {
		return nil, nil, nil, managedMissingRun(ctx, b, in.RunID)
	}
	c := b.ManagedCommands[in.CommandID]
	if c == nil || r.HostID != h.Host.ID || c.Command.RunID != r.ID {
		return nil, nil, nil, core.Fail("conflict", "Чужой или устаревший dispatch")
	}
	if c.Command.DispatchID == "" || c.Command.DispatchID != in.DispatchID {
		return nil, nil, nil, core.Fail("stale_dispatch", "Чужой или устаревший dispatch")
	}
	if c.Command.Generation != in.Generation {
		return nil, nil, nil, core.Fail("stale_generation", "Устаревшая generation запуска")
	}
	if (in.Phase == "accepted" || in.Phase == "started") && !managedCommandAllowed(ctx, b, h, c, now) && !managedLateReceipt(ctx, b, h, c, in, now) {
		return nil, nil, nil, core.Fail("forbidden", "Инициатор команды потерял разрешение управления")
	}
	return h, r, c, nil
}

// managedMissingRun distinguishes a run moved to the history archive
// (RUN_ARCHIVED) from one that never existed here (NOT_FOUND); both are
// definitive for a worker's pending report and map to HTTP 404.
func managedMissingRun(ctx context.Context, b *board, id string) error {
	var n int
	if b.tx != nil && b.tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_runs WHERE key=?`, automationArchivePrefix+id).Scan(&n) == nil && n > 0 {
		return core.Fail("run_archived", "Запуск перенесён в архив истории")
	}
	return core.Fail("not_found", "Запуск не найден")
}

// managedLateReceipt admits the receipt or start of a command dispatched before
// a routine workflow interruption (pause, cancel, question, timeout) when the
// reporter is still the live owner of that dispatch. The report is recorded so
// the native session and invocation are accounted for; the run stays stopping
// and its owned stop stays pending. Lost initiator or project authority is not
// an interruption and still refuses the report.
func managedLateReceipt(ctx context.Context, b *board, h *managedHostRecord, c *managedCommandRecord, in core.ManagedReportInput, now time.Time) bool {
	return c.Command.DispatchID != "" && h.WorkerID == in.WorkerID && h.Host.WorkerEpoch == in.WorkerEpoch && c.WorkerID == in.WorkerID && c.WorkerEpoch == in.WorkerEpoch && h.Host.LeaseExpiresAt.After(now) && managedWorkflowInterrupted(ctx, b, h, c, now)
}
func managedReportCheck(ctx context.Context, b *board, a core.Actor, in core.ManagedReportInput, now time.Time) (*managedHostRecord, *core.ManagedRun, *managedCommandRecord, error) {
	h, r, c, err := managedReportAuthorize(ctx, b, a, in, now)
	if err != nil {
		return nil, nil, nil, err
	}
	if r.Generation != in.Generation {
		return nil, nil, nil, core.Fail("stale_generation", "Устаревшая generation запуска")
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
		// The current lease holder reporting its own local stop (worker deadline,
		// `worker recover`) is not a takeover; it keeps lease, worker ID and epoch.
		liveOwner := h.WorkerID == in.WorkerID && h.Host.WorkerEpoch == in.WorkerEpoch && h.Host.LeaseExpiresAt.After(now)
		terminal := c.Command.Status == "failed" || c.Command.Status == "cancelled" || c.Command.Status == "delivery_unknown"
		if terminal && in.Phase != "recovered_stopped" {
			return out, core.Fail("command_terminal", "Команда уже terminal")
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
			if r.Automation != nil {
				for _, other := range b.ManagedRuns {
					if other.ID != r.ID && other.Provider == r.Provider && other.ProviderSessionID == in.ProviderSessionID && other.Automation != nil {
						return out, core.Fail("rule_violation", "Automation роли требуют отдельные native conversations")
					}
				}
			}
			r.ProviderSessionID = in.ProviderSessionID
			automationRunStarted(b, r, now)
			if !managedCommandAllowed(ctx, b, h, c, now) {
				// Late start after an interruption: never resume work.
				r.State = "stopping"
				r.LastError = managedInterruptedMessage
			} else if r.State != "stopping" {
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
				// Transient: retried after the previous lease lapses.
				return out, core.Fail("lease_held", "Прежний worker ещё владеет lease")
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
			if !liveOwner {
				// A restarted incarnation reconciles; the next poll re-acquires.
				h.WorkerID = ""
				h.Host.LeaseExpiresAt = now
			}
		default:
			return out, core.Fail("validation", "Неизвестная фаза worker report")
		}
		if in.Phase == "completed" || in.Phase == "stopped" || in.Phase == "failed" || in.Phase == "recovered_stopped" {
			automationRunTerminal(b, r, now)
			if err := managedCloseSessions(b, r, h, in.Phase, now); err != nil {
				return out, err
			}
			c.TerminalPhase = in.Phase
			c.TerminalAt = &now
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
		} else if in.Phase == "recovered_stopped" && liveOwner {
			// The live owner keeps its current lease (still valid for the
			// following run_exit proof) but a local stop never extends it: a
			// worker that is shutting down must not delay its restart.
			h.Host.LastSeenAt = now
		}
		emit(b, now, "managed_report_"+in.Phase, "", a.SessionID)
		return core.ManagedResult{Run: *r, Command: c.Command}, nil
	})
}
