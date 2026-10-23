package store

import (
	"agent-board/internal/core"
	"context"
	"strconv"
	"strings"
	"time"
)

func attestCheck(ctx context.Context, b *board, a core.Actor, in core.WorkflowAttestInput, now time.Time) (*core.Workflow, *core.WorkflowStep, *core.Task, error) {
	terminal := automationTerminalAttest(in.Kind)
	sess, _, err := automationActor(ctx, b, a, now, !terminal)
	if err != nil {
		return nil, nil, nil, err
	}
	if sess.Role != "human" {
		return nil, nil, nil, core.Fail("forbidden", "Worker требует owner human session")
	}
	if _, bound := ExecutionIdentityFromContext(ctx); bound {
		return nil, nil, nil, core.Fail("forbidden", "Child не сообщает worker proof")
	}
	h, err := managedWorker(b, a, in.ManagedPollInput, managedEpoch(ctx))
	if err != nil {
		return nil, nil, nil, err
	}
	r := b.ManagedRuns[in.RunID]
	st := b.Automation.Steps[in.StepID]
	w := b.Automation.Workflows[in.WorkflowID]
	c := b.ManagedCommands[in.CommandID]
	if r == nil || st == nil || w == nil || c == nil || r.HostID != h.Host.ID || r.Automation == nil || r.Automation.StepID != st.ID || st.WorkflowID != w.ID || st.ManagedRunID != r.ID || st.ManagedGeneration != in.Generation || r.Generation != in.Generation || in.ControlEpoch != r.Automation.ControlEpoch || st.TaskKey != in.TaskKey || c.Command.RunID != r.ID || c.Command.Generation != in.Generation || c.Command.DispatchID != in.DispatchID || c.WorkerID != in.WorkerID || c.WorkerEpoch != in.WorkerEpoch || h.WorkerID != in.WorkerID || h.Host.WorkerEpoch != in.WorkerEpoch || !h.Host.LeaseExpiresAt.After(now) {
		return nil, nil, nil, core.Fail("forbidden", "Worker attestation binding устарел")
	}
	if in.Kind == "run_exit" && !automationExitReportMatches(r, c) {
		return nil, nil, nil, core.Fail("conflict", "run_exit требует подтверждённый terminal report этой execution command")
	}
	if r.ProviderSessionID != "" && in.NativeSessionID != r.ProviderSessionID || !managedSessionID.MatchString(in.NativeSessionID) {
		return nil, nil, nil, core.Fail("forbidden", "Native conversation не совпадает")
	}
	t := b.Tasks[st.TaskKey]
	if t == nil || t.Generation != in.TaskGeneration || r.Automation.TaskGeneration < 1 || in.TaskGeneration > r.Automation.TaskGeneration || in.TaskGeneration < r.Automation.TaskGeneration-1 {
		return nil, nil, nil, core.Fail("evidence_stale", "Task generation изменилась")
	}
	if !terminal {
		if w.State != "running" || w.ControlEpoch != in.ControlEpoch || !st.DeadlineAt.After(now) {
			return nil, nil, nil, core.Fail("forbidden", "Workflow остановлен")
		}
		if err = authorizeWorkflow(ctx, b, w, now); err != nil {
			return nil, nil, nil, err
		}
		if st.Kind != "tester" && st.Kind != "developer" {
			return nil, nil, nil, core.Fail("forbidden", "Эта роль не запускает checks")
		}
		if t.Lease == nil {
			return nil, nil, nil, core.Fail("lease_lost", "Для check нужен native task lease")
		}
		sess := b.Sessions[t.Lease.SessionID]
		if sess == nil || sess.ManagedRunID != r.ID || sess.ManagedGeneration != in.Generation || sess.WorkflowID != w.ID || sess.WorkflowStepID != st.ID || sess.ExecutionCapabilityID == "" {
			return nil, nil, nil, core.Fail("forbidden", "Check требует собственного native holder")
		}
	}
	return w, st, t, nil
}
func observationsValid(o core.WorkflowObservations) error {
	if o.ActualHead != "" && !validCommit(o.ActualHead) || o.TargetSHA != "" && !validCommit(o.TargetSHA) || o.ManifestSHA256 != "" && !digestValid(o.ManifestSHA256) || o.ReportSHA256 != "" && !digestValid(o.ReportSHA256) || o.ChecksPassed < 0 || o.ChecksFailed < 0 || o.ChecksSkipped < 0 || o.ChecksPassed+o.ChecksFailed+o.ChecksSkipped > 10000000 || len(o.ArtifactRefs) > 32 {
		return core.Fail("validation", "Некорректные bounded observations")
	}
	for _, ref := range o.ArtifactRefs {
		hash := strings.TrimPrefix(ref, "sha256:")
		if hash == ref {
			hash = strings.TrimPrefix(ref, "artifact:")
		}
		if hash == ref || !digestValid(hash) {
			return core.Fail("validation", "Некорректный artifact reference")
		}
	}
	if o.ReportKind != "" && o.ReportKind != "junit" && o.ReportKind != "tap" && o.ReportKind != "exit" {
		return core.Fail("validation", "Неизвестный report kind")
	}
	return nil
}
func (s *Store) WorkflowAttest(ctx context.Context, a core.Actor, in core.WorkflowAttestInput) (core.WorkflowAttestResult, error) {
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	proof := in
	proof.HostSecret = secret
	return automationMutate(s, ctx, a, in.RequestID, "workflow_attest", in, func(b *board, now time.Time) error { _, _, _, err := attestCheck(ctx, b, a, proof, now); return err }, func(b *board, now time.Time) (core.WorkflowAttestResult, error) {
		var out core.WorkflowAttestResult
		w, st, _, err := attestCheck(ctx, b, a, proof, now)
		if err != nil {
			return out, err
		}
		if err = observationsValid(in.Observations); err != nil {
			return out, err
		}
		if len(b.Automation.Attestations) >= 32768 {
			return out, core.Fail("capacity", "Достигнут предел attestation history")
		}
		key := in.RunID + "/" + in.StepID + "/" + strconv.FormatInt(in.Generation, 10) + "/" + in.CheckRequestID
		if in.Kind == "run_exit" {
			key += "/exit"
		} else if !bounded(in.CheckRequestID, 128) {
			return out, core.Fail("validation", "Нужен bounded check request_id")
		}
		record := b.Automation.Attestations[key]
		if in.Kind == "run_exit" {
			if record != nil {
				return out, core.Fail("conflict", "Exit proof уже записан")
			}
			record = &core.WorkflowAttestation{ID: key, Kind: "run_exit", Status: "completed"}
			b.Automation.Attestations[key] = record
		} else {
			presetFound := false
			for _, preset := range workflowTaskChecks(b, w, b.Tasks[st.TaskKey]) {
				if preset.PresetID == in.PresetID && preset.LocalVersion == in.PresetVersion && preset.LocalDigest == in.PresetDigest {
					presetFound = true
				}
			}
			if !presetFound {
				return out, core.Fail("forbidden", "Check preset не разрешён этим task")
			}
			if record != nil && (record.PresetID != in.PresetID || record.PresetVersion != in.PresetVersion || record.PresetDigest != in.PresetDigest || record.NativeSessionID != in.NativeSessionID || record.TaskGeneration != in.TaskGeneration) {
				return out, core.Fail("conflict", "Check intent tuple неизменяем")
			}
			switch in.Kind {
			case "check_reserved":
				if record != nil {
					return out, core.Fail("conflict", "Check уже зарезервирован")
				}
				count := 0
				for _, existing := range b.Automation.Attestations {
					if existing.StepID == st.ID && existing.Kind == "check" {
						count++
					}
				}
				if count >= 64 {
					return out, core.Fail("capacity", "Предел checks per step")
				}
				record = &core.WorkflowAttestation{ID: key, Kind: "check", Status: "reserved"}
				b.Automation.Attestations[key] = record
			case "check_started":
				if record == nil || record.Status != "reserved" {
					return out, core.Fail("conflict", "Нужна check reservation")
				}
				record.Status = "running"
			case "check_finished":
				if record == nil || record.Status != "running" {
					return out, core.Fail("conflict", "Нет наблюдаемого check start")
				}
				if in.Observations.CheckExitCode == nil {
					return out, core.Fail("evidence_missing", "Нужен реальный check exit")
				}
				record.Status = "completed"
			case "check_unknown":
				if record == nil {
					return out, core.Fail("conflict", "Нет check intent")
				}
				record.Status = "delivery_unknown"
				w.State = "needs_recovery"
				w.ReasonCode = "check_ownership_uncertain"
				w.ControlEpoch++
				revokeWorkflowCapabilities(b, w.ID)
				workflowTouch(w, now)
			default:
				return out, core.Fail("validation", "Неизвестный attestation kind")
			}
		}
		if record.CreatedAt.IsZero() {
			for _, prior := range b.Automation.Attestations {
				if prior.WorkflowID == w.ID && prior.Sequence >= record.Sequence {
					record.Sequence = prior.Sequence + 1
				}
			}
			if record.Sequence == 0 {
				record.Sequence = 1
			}
			record.CreatedAt = now
		}
		record.UpdatedAt = now
		record.WorkflowID = w.ID
		record.StepID = st.ID
		record.ControlEpoch = in.ControlEpoch
		record.RunID = in.RunID
		record.Generation = in.Generation
		record.TaskKey = in.TaskKey
		record.TaskGeneration = in.TaskGeneration
		record.NativeSessionID = in.NativeSessionID
		record.CheckRequestID = in.CheckRequestID
		record.PresetID = in.PresetID
		record.PresetVersion = in.PresetVersion
		record.PresetDigest = in.PresetDigest
		record.Observations = in.Observations
		workflowTouch(w, now)
		out.Attestation = *record
		out.ControlEpoch = w.ControlEpoch
		out.MaxRuntimeSeconds = int64(st.DeadlineAt.Sub(now).Seconds())
		if out.MaxRuntimeSeconds < 0 {
			out.MaxRuntimeSeconds = 0
		}
		out.ReservationTTLSeconds = 30
		return out, nil
	})
}

func automationTerminalAttest(kind string) bool {
	return kind == "run_exit" || kind == "check_finished" || kind == "check_unknown"
}

// An applied start receipt only proves turn submission. Exit attestation needs
// the exact durable terminal report, including a distinct owned stop command.
func automationExitReportMatches(r *core.ManagedRun, c *managedCommandRecord) bool {
	if c.TerminalAt == nil || c.Command.RunID != r.ID || c.Command.Generation != r.Generation {
		return false
	}
	switch c.TerminalPhase {
	case "completed":
		return r.State == "completed" && c.Command.Kind != "stop" && c.Command.Status == "applied"
	case "stopped":
		return r.State == "stopped" && c.Command.Kind == "stop" && c.Command.Status == "applied"
	case "failed":
		return r.State == "failed" && c.Command.Status == "failed"
	case "recovered_stopped":
		return r.State == "stopped" && c.Command.Status == "cancelled"
	}
	return false
}
