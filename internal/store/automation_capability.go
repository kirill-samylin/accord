package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/subtle"
	"strings"
	"time"
)

func revokeWorkflowCapabilities(b *board, id string) {
	for _, c := range b.Automation.Capabilities {
		if c.WorkflowID == id {
			c.State = "revoked"
		}
	}
}
func executionCurrent(ctx context.Context, b *board, c *executionCapabilityRecord, now time.Time) error {
	if c == nil || c.State != "active" || !c.ExpiresAt.After(now) {
		return core.Fail("unauthorized", "Execution capability недействительна")
	}
	w, err := workflowFor(b, c.WorkflowID)
	if err != nil {
		return err
	}
	if w.ControlEpoch != c.ControlEpoch || w.State != "running" || w.CurrentStepID != c.StepID || !w.DeadlineAt.After(now) {
		return core.Fail("forbidden", "Workflow step остановлен или заменён")
	}
	if err = authorizeWorkflow(ctx, b, w, now); err != nil {
		return err
	}
	st := b.Automation.Steps[c.StepID]
	r := b.ManagedRuns[c.RunID]
	h := b.ManagedHosts[c.HostID]
	if st == nil || r == nil || h == nil || st.ManagedRunID != r.ID || st.ManagedGeneration != c.Generation || r.Generation != c.Generation || r.State != "starting" && r.State != "running" || h.Host.WorkerEpoch != c.WorkerEpoch || !h.Host.LeaseExpiresAt.After(now) || !st.DeadlineAt.After(now) {
		return core.Fail("forbidden", "Execution fencing потерян")
	}
	return nil
}
func (s *Store) ExecutionCapabilityRegister(ctx context.Context, a core.Actor, in core.ExecutionCapabilityRegisterInput) (core.ExecutionCapability, error) {
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	proof := in
	proof.HostSecret = secret
	check := func(b *board, now time.Time) error {
		_, err := automationHuman(ctx, b, a, now)
		if err != nil {
			return err
		}
		h, err := managedWorker(b, a, proof.ManagedPollInput, managedEpoch(ctx))
		if err != nil {
			return err
		}
		w, err := workflowFor(b, in.WorkflowID)
		if err != nil {
			return err
		}
		if err = authorizeWorkflow(ctx, b, w, now); err != nil {
			return err
		}
		st := b.Automation.Steps[in.StepID]
		c := b.ManagedCommands[in.CommandID]
		r := b.ManagedRuns[in.RunID]
		if st == nil || r == nil || c == nil || w.State != "running" || w.CurrentStepID != st.ID || st.WorkflowID != w.ID || w.ControlEpoch != in.ControlEpoch || st.ManagedRunID != r.ID || r.Automation == nil || r.Generation != in.Generation || st.ManagedGeneration != in.Generation || r.HostID != h.Host.ID || h.Host.WorkerEpoch != in.WorkerEpoch || h.WorkerID != in.WorkerID || !h.Host.LeaseExpiresAt.After(now) || c.Command.RunID != r.ID || c.Command.Generation != in.Generation || c.Command.DispatchID != in.DispatchID || c.WorkerID != in.WorkerID || c.WorkerEpoch != in.WorkerEpoch || (c.Command.Status != "accepted" && c.Command.Status != "applied") {
			return core.Fail("forbidden", "Capability требует текущий accepted attempt")
		}
		return nil
	}
	return automationMutate(s, ctx, a, in.RequestID, "execution_capability_register", in, check, func(b *board, now time.Time) (core.ExecutionCapability, error) {
		if !strings.HasPrefix(in.CapabilityID, "ec_") || len(in.CapabilityID) != 35 || !digestValid(in.Digest) {
			return core.ExecutionCapability{}, core.Fail("validation", "Нужны fixed capability ID/digest")
		}
		active := 0
		for _, c := range b.Automation.Capabilities {
			if c.Digest == in.Digest {
				return core.ExecutionCapability{}, core.Fail("conflict", "Capability secret повторно использован")
			}
			if c.RunID == in.RunID && c.Generation == in.Generation {
				return core.ExecutionCapability{}, core.Fail("conflict", "Capability попытки уже зарегистрирована")
			}
			if c.State == "active" && c.ExpiresAt.After(now) {
				active++
			}
		}
		if active >= 128 || len(b.Automation.Capabilities) >= 4096 {
			return core.ExecutionCapability{}, core.Fail("capacity", "Достигнут предел capabilities")
		}
		st := b.Automation.Steps[in.StepID]
		w := b.Automation.Workflows[in.WorkflowID]
		expiry := st.DeadlineAt
		for _, v := range []time.Time{w.Grant.ExpiresAt, w.DeadlineAt} {
			if v.Before(expiry) {
				expiry = v
			}
		}
		c := &executionCapabilityRecord{Digest: in.Digest, ExecutionCapability: core.ExecutionCapability{ID: in.CapabilityID, HostID: in.HostID, WorkflowID: in.WorkflowID, StepID: in.StepID, ControlEpoch: in.ControlEpoch, RunID: in.RunID, Generation: in.Generation, WorkerEpoch: in.WorkerEpoch, Role: st.Kind, ProfileID: st.Profile.ID, ProfileVersion: st.Profile.Version, State: "active", Operations: []string{"describe", "heartbeat", "next", "get", "update", "transition", "workflow_get", "workflow_report", "workflow_plan", "workflow_question", "workflow_run_check"}, CreatedAt: now, ExpiresAt: expiry}}
		b.Automation.Capabilities[c.ID] = c
		return c.ExecutionCapability, nil
	})
}
func (s *Store) AuthenticateExecutionCapability(ctx context.Context, token string) (core.ExecutionIdentity, error) {
	var zero core.ExecutionIdentity
	if !strings.HasPrefix(token, core.ExecutionTokenPrefix) || len(token) != len(core.ExecutionTokenPrefix)+64 || !digestValid(strings.TrimPrefix(token, core.ExecutionTokenPrefix)) {
		return zero, core.Fail("unauthorized", "Invalid execution credential")
	}
	digest := managedHash(token)
	return transact(s, ctx, func(b *board, now time.Time) (core.ExecutionIdentity, error) {
		for _, c := range b.Automation.Capabilities {
			if subtle.ConstantTimeCompare([]byte(c.Digest), []byte(digest)) != 1 {
				continue
			}
			if err := executionCurrent(ctx, b, c, now); err != nil {
				return zero, err
			}
			w := b.Automation.Workflows[c.WorkflowID]
			auth, _ := AutomationAuthorityFromContext(ctx)
			p := auth.Principals[w.Grant.HostOwnerID].Principal
			return core.ExecutionIdentity{Principal: p, Capability: c.ExecutionCapability}, nil
		}
		return zero, core.Fail("unauthorized", "Invalid execution credential")
	})
}
func executionGuard(ctx context.Context, b *board, a core.Actor, method string, input any, now time.Time) error {
	ident, bound := ExecutionIdentityFromContext(ctx)
	if !bound {
		if method == "describe" {
			if in, ok := input.(core.OpenSession); ok && in.ManagedRunID != "" {
				if r := b.ManagedRuns[in.ManagedRunID]; r != nil && r.Automation != nil {
					return core.Fail("forbidden", "Automated run требует scoped execution credential")
				}
			}
		}
		if s := b.Sessions[a.SessionID]; s != nil && s.ExecutionCapabilityID != "" {
			return core.Fail("forbidden", "Scoped session требует свой credential")
		}
		return nil
	}
	c := b.Automation.Capabilities[ident.Capability.ID]
	if err := executionCurrent(ctx, b, c, now); err != nil {
		return err
	}
	if !managedContains(c.Operations, method) {
		return core.Fail("forbidden", "Операция не входит в execution scope")
	}
	st := b.Automation.Steps[c.StepID]
	if method == "next" {
		r := b.ManagedRuns[c.RunID]
		t := b.Tasks[st.TaskKey]
		if r == nil || t == nil || r.Automation == nil || r.Automation.TaskGeneration < 1 {
			return core.Fail("forbidden", "Нет expected task generation")
		}
		expected := r.Automation.TaskGeneration
		if t.Generation != expected-1 && (t.Generation != expected || t.Lease == nil || t.Lease.SessionID != a.SessionID) {
			return core.Fail("lease_lost", "Task lease потеряна; текущая capability не захватывает новую generation")
		}
	}
	if method == "describe" {
		in, ok := input.(core.OpenSession)
		if !ok || in.ManagedRunID != c.RunID || in.ManagedGeneration != c.Generation || in.Role != c.Role || in.ProfileID != c.ProfileID || in.ProfileVersion != c.ProfileVersion || in.RunID != c.RunID {
			return core.Fail("forbidden", "Describe binding не совпадает с capability")
		}
		return nil
	}
	sess := b.Sessions[a.SessionID]
	if sess == nil || sess.ExecutionCapabilityID != c.ID || sess.WorkflowID != c.WorkflowID || sess.WorkflowStepID != c.StepID || sess.WorkflowControlEpoch != c.ControlEpoch {
		return core.Fail("forbidden", "Session/capability binding не совпадает")
	}
	taskKey := ""
	switch in := input.(type) {
	case core.NextInput:
		taskKey = in.TaskKey
	case core.GetInput:
		taskKey = in.TaskKey
	case core.UpdateInput:
		taskKey = in.TaskKey
	case core.TransitionInput:
		taskKey = in.TaskKey
	case core.HeartbeatInput:
		taskKey = in.TaskKey
	}
	allowedTarget := false
	if method == "get" {
		if own := b.Tasks[st.TaskKey]; own != nil && own.Review != nil {
			allowedTarget = taskKey == own.Review.TargetKey
		}
	}
	if taskKey != "" && taskKey != st.TaskKey && !allowedTarget || method == "next" && taskKey == "" {
		return core.Fail("forbidden", "Разрешена только назначенная workflow задача")
	}
	return nil
}
