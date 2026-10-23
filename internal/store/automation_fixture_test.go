package store

import (
	"agent-board/internal/core"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

type automationFixture struct {
	f      *fixture
	a      core.Actor
	ctx    context.Context
	h      core.ManagedHost
	p      core.AutomationPolicy
	check  core.AutomationCheckPresetRef
	native int
}

func automationSetup(t *testing.T) *automationFixture {
	t.Helper()
	f := newFixture(t)
	a := f.actor(t, "human")
	ctx := WithAutomationAuthority(testContext, AutomationAuthority{ProjectID: "test", Writable: true, Principals: map[string]AutomationPrincipal{f.p.ID: {Principal: f.p, Epoch: 0, LegacyTokenID: f.p.ID, ExpiresAt: f.clock.read().Add(24 * time.Hour)}}})
	ctx = WithManagedPrincipals(ctx, map[string]ManagedPrincipalGrant{f.p.ID: {Epoch: 0, LegacyTokenID: f.p.ID}})
	bindings := map[string]core.AutomationRoleBinding{}
	roles := map[string][]string{}
	for _, role := range automationRoles {
		settings := settings()
		settings.Name = role
		settings.Role = role
		profile, err := f.s.ProfileCreate(ctx, f.coord, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: settings})
		if err != nil {
			t.Fatal(err)
		}
		bindings[role] = core.AutomationRoleBinding{Provider: "codex", ProfileID: profile.ID, ProfileVersion: profile.Version}
		roles[role] = []string{"codex"}
	}
	check := core.AutomationCheckPresetRef{PresetID: "unit", LocalVersion: 1, LocalDigest: strings.Repeat("a", 64), Kind: "test", Required: true, Isolation: "trusted_owner_process"}
	expiry := f.clock.read().Add(24 * time.Hour)
	consent := &core.AutomationHostConsent{Enabled: true, Version: 1, ExpiresAt: expiry, Scopes: map[string]core.AutomationWorkspaceScope{"src": {Version: 1, ExpiresAt: expiry, Roles: roles, Checks: []core.AutomationCheckPresetRef{check}, MaxStepSeconds: 3600, MaxWorkflowSeconds: 86400, MaxNativeInvocations: 128}}}
	h, err := f.s.ManagedHostEnroll(ctx, a, core.ManagedHostEnrollInput{RequestID: f.id(), HostSecret: managedTestSecret, Name: "Fixture", Workspaces: []string{"src"}, Providers: []string{"codex"}, Automation: consent})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithAutomationPolicyApproval(ctx, AutomationPolicyApproval{PrincipalID: f.p.ID, SessionID: a.SessionID, Epoch: 0, HostID: h.ID, ConsentVersion: 1, Source: "local", HostProof: managedTestSecret})
	p, err := f.s.AutomationPolicyCreate(ctx, a, core.AutomationPolicyCreateInput{RequestID: f.id(), AutomationPolicySettings: core.AutomationPolicySettings{Name: "Pipeline", HostID: h.ID, HostConsentVersion: 1, Workspace: "src", RoleBindings: bindings, IntakePrincipals: []string{f.p.ID}, ControlPrincipals: []string{f.p.ID}, ClientIntake: true, Checks: []core.AutomationCheckPresetRef{check}, GitPolicy: "local_commits", Limits: automationLimits(core.AutomationLimits{}), ExpiresAt: expiry}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.ManagedPoll(ctx, a, core.ManagedPollInput{RequestID: f.id(), HostID: h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerCapabilities: map[string]int{"automation": 1}}); err != nil {
		t.Fatal(err)
	}
	return &automationFixture{f: f, a: a, ctx: ctx, h: h, p: p, check: check}
}
func (m *automationFixture) submit(t *testing.T) core.WorkflowPage {
	t.Helper()
	v, e := m.f.s.WorkflowSubmit(m.ctx, m.a, core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: "Implement a verified change."})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func (m *automationFixture) page(t *testing.T, id string) core.WorkflowPage {
	t.Helper()
	v, e := m.f.s.WorkflowGet(m.ctx, m.a, core.WorkflowReadInput{WorkflowID: id})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func (m *automationFixture) tick(t *testing.T, id string) core.WorkflowPage {
	t.Helper()
	p := m.page(t, id)
	if _, e := m.f.s.AutomationTick(m.ctx, core.WorkflowAdvanceInput{WorkflowID: id, ExpectedVersion: p.Workflow.Version, ControlEpoch: p.Workflow.ControlEpoch}); e != nil {
		t.Fatal(e)
	}
	return m.page(t, id)
}

type automationAttempt struct {
	poll  core.ManagedPollResult
	actor core.Actor
	ctx   context.Context
	claim core.Claim
	uuid  string
}

func (m *automationFixture) attempt(t *testing.T, id string) automationAttempt {
	t.Helper()
	poll, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerCapabilities: map[string]int{"automation": 1}})
	if e != nil || poll.Command == nil {
		t.Fatalf("poll: %+v %v", poll, e)
	}
	m.native++
	uuid := fmt.Sprintf("%08x-1234-1234-1234-%012x", m.native, m.native)
	if poll.Run.ProviderSessionID != "" {
		uuid = poll.Run.ProviderSessionID
	}
	report := core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: poll.WorkerEpoch}, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, Phase: "accepted"}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	raw := core.ExecutionTokenPrefix + fmt.Sprintf("%064x", m.native)
	binding := poll.Run.Automation
	cap, e := m.f.s.ExecutionCapabilityRegister(m.ctx, m.a, core.ExecutionCapabilityRegisterInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: poll.WorkerEpoch}, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, RunID: poll.Run.ID, Generation: poll.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, CapabilityID: fmt.Sprintf("ec_%032x", m.native), Digest: managedHash(raw)})
	if e != nil {
		t.Fatal(e)
	}
	identity, e := m.f.s.AuthenticateExecutionCapability(m.ctx, raw)
	if e != nil || identity.Capability.ID != cap.ID {
		t.Fatal("authcap", e)
	}
	ctx := WithExecutionCapability(m.ctx, identity)
	open := core.OpenSession{RequestID: m.f.id(), RunID: poll.Run.ID, Machine: m.f.p.Machine, Role: poll.Run.Profile.Role, ProfileID: poll.Run.Profile.ID, ProfileVersion: poll.Run.Profile.Version, ManagedRunID: poll.Run.ID, ManagedGeneration: poll.Run.Generation}
	view, e := m.f.s.Describe(ctx, m.f.p, open)
	if e != nil {
		t.Fatal("describe", e)
	}
	again, e := m.f.s.Describe(ctx, m.f.p, open)
	if e != nil || again.SessionID != view.SessionID {
		t.Fatal("describe lostresponse", e)
	}
	actor := core.Actor{Principal: m.f.p, SessionID: view.SessionID}
	claim, e := m.f.s.Next(ctx, actor, core.NextInput{RequestID: m.f.id(), TaskKey: poll.Run.TaskKey})
	if e != nil || claim.Task == nil {
		t.Fatal("claim", e)
	}
	report.RequestID = m.f.id()
	report.Phase = "started"
	report.ProviderSessionID = uuid
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	return automationAttempt{poll, actor, ctx, claim, uuid}
}
func (m *automationFixture) finish(t *testing.T, a automationAttempt, evidence core.Evidence, outcome string) {
	t.Helper()
	_, err := m.f.s.Transition(a.ctx, a.actor, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: m.f.id(), TaskKey: a.claim.Task.Key, Generation: a.claim.Lease.Generation}, State: "done", Outcome: outcome, Evidence: &evidence})
	if err != nil {
		t.Fatal("finish", err)
	}
}
func (m *automationFixture) attest(t *testing.T, a automationAttempt, kind, checkID string, o core.WorkflowObservations) core.WorkflowAttestResult {
	t.Helper()
	binding := a.poll.Run.Automation
	v, e := m.f.s.WorkflowAttest(m.ctx, m.a, core.WorkflowAttestInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch}, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, TaskKey: a.claim.Task.Key, TaskGeneration: a.claim.Lease.Generation, NativeSessionID: a.uuid, Kind: kind, CheckRequestID: checkID, PresetID: m.check.PresetID, PresetVersion: m.check.LocalVersion, PresetDigest: m.check.LocalDigest, Observations: o})
	if e != nil {
		t.Fatal("attest", kind, e)
	}
	return v
}
func (m *automationFixture) exit(t *testing.T, a automationAttempt, head, target string) {
	t.Helper()
	_, e := m.f.s.ManagedReport(m.ctx, m.a, core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: a.poll.WorkerEpoch}, RunID: a.poll.Run.ID, Generation: a.poll.Run.Generation, CommandID: a.poll.Command.ID, DispatchID: a.poll.Command.DispatchID, Phase: "completed"})
	if e != nil {
		t.Fatal(e)
	}
	clean := true
	m.attest(t, a, "run_exit", "", core.WorkflowObservations{ActualHead: head, TargetSHA: target, SourceClean: &clean, ManifestSHA256: strings.Repeat("b", 64)})
}
func (m *automationFixture) plan(t *testing.T, id string, a automationAttempt) core.WorkflowPage {
	t.Helper()
	p := m.page(t, id)
	v, e := m.f.s.WorkflowPlan(a.ctx, a.actor, core.WorkflowPlanInput{RequestID: m.f.id(), WorkflowID: id, Version: p.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Tasks: []core.WorkflowPlanTask{{ClientKey: "one", Title: "Implement requirement", Description: "Write implementation and unit tests.", Acceptance: []string{"behavior verified"}, RequiredCheckIDs: []string{m.check.PresetID}}}})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
