package worker

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This fixture uses a real SQLite board and domain operations. The typed
// authority replaces the registry checkpoint only; no attestation is fabricated.
type automationHTTPFixture struct {
	t      *testing.T
	s      *store.Store
	ctx    context.Context
	a      core.Actor
	token  core.TokenView
	h      core.ManagedHost
	p      core.AutomationPolicy
	serial int
}

func (f *automationHTTPFixture) id() string { f.serial++; return randomID("http_") }
func (f *automationHTTPFixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}
func (f *automationHTTPFixture) page(id string) core.WorkflowPage {
	f.t.Helper()
	p, err := f.s.WorkflowGet(f.ctx, f.a, core.WorkflowReadInput{WorkflowID: id})
	f.check(err)
	return p
}
func (f *automationHTTPFixture) tick(id string) core.WorkflowPage {
	f.t.Helper()
	p := f.page(id)
	_, err := f.s.AutomationTick(f.ctx, core.WorkflowAdvanceInput{WorkflowID: id, ExpectedVersion: p.Workflow.Version, ControlEpoch: p.Workflow.ControlEpoch})
	f.check(err)
	return f.page(id)
}
func (f *automationHTTPFixture) poll(epoch int64) core.ManagedPollResult {
	f.t.Helper()
	p, err := f.s.ManagedPoll(f.ctx, f.a, core.ManagedPollInput{RequestID: f.id(), HostID: f.h.ID, HostSecret: strings.Repeat("a", 64), WorkerID: "http-worker", WorkerEpoch: epoch, WorkerCapabilities: map[string]int{"automation": 1}})
	f.check(err)
	return p
}
func (f *automationHTTPFixture) report(p core.ManagedPollResult, phase, uuid string) core.ManagedResult {
	f.t.Helper()
	r, err := f.s.ManagedReport(f.ctx, f.a, core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: f.id(), HostID: f.h.ID, HostSecret: strings.Repeat("a", 64), WorkerID: "http-worker", WorkerEpoch: p.WorkerEpoch}, RunID: p.Run.ID, Generation: p.Run.Generation, CommandID: p.Command.ID, DispatchID: p.Command.DispatchID, Phase: phase, ProviderSessionID: uuid})
	f.check(err)
	return r
}
func newAutomationHTTPFixture(t *testing.T) *automationHTTPFixture {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(filepath.Join(t.TempDir(), "board.db"), core.Config{Project: "http-fixture"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	f := &automationHTTPFixture{t: t, s: s, ctx: ctx}
	f.token, err = s.CreateToken(ctx, core.TokenInput{Owner: "fixture", Machine: "fixture", TTL: time.Hour})
	f.check(err)
	principal, err := s.Authenticate(ctx, f.token.Secret)
	f.check(err)
	view, err := s.Describe(ctx, principal, core.OpenSession{RequestID: f.id(), Role: "human", RunID: "human", Machine: "fixture"})
	f.check(err)
	f.a = core.Actor{Principal: principal, SessionID: view.SessionID}
	f.ctx = store.WithManagedPrincipals(ctx, map[string]store.ManagedPrincipalGrant{principal.ID: {Epoch: 0, LegacyTokenID: principal.ID}})
	f.ctx = store.WithAutomationAuthority(f.ctx, store.AutomationAuthority{ProjectID: "p_00000000000000000000000000000001", Writable: true, Principals: map[string]store.AutomationPrincipal{principal.ID: {Principal: principal, Epoch: 0, LegacyTokenID: principal.ID, ExpiresAt: f.token.ExpiresAt}}})
	roles := map[string][]string{}
	bindings := map[string]core.AutomationRoleBinding{}
	for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
		p, e := s.ProfileCreate(f.ctx, f.a, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: core.ProfileSettings{Name: role, Role: role, Description: "Isolated HTTP regression", Instructions: "Use exact evidence", Provider: "openai"}})
		f.check(e)
		roles[role] = []string{"codex"}
		bindings[role] = core.AutomationRoleBinding{Provider: "codex", ProfileID: p.ID, ProfileVersion: p.Version}
	}
	checks := []core.AutomationCheckPresetRef{{PresetID: "unit", LocalVersion: 1, LocalDigest: strings.Repeat("b", 64), Kind: "test", Required: true, Isolation: "trusted_owner_process"}}
	expires := time.Now().Add(time.Hour)
	f.h, err = s.ManagedHostEnroll(f.ctx, f.a, core.ManagedHostEnrollInput{RequestID: f.id(), Name: "HTTP fixture", HostSecret: strings.Repeat("a", 64), Controllers: []string{principal.ID}, Workspaces: []string{"test"}, Providers: []string{"codex"}, Automation: &core.AutomationHostConsent{Version: 1, Enabled: true, ExpiresAt: expires, Scopes: map[string]core.AutomationWorkspaceScope{"test": {Version: 1, ExpiresAt: expires, Roles: roles, Checks: checks, MaxStepSeconds: 600, MaxWorkflowSeconds: 3600, MaxNativeInvocations: 16}}}})
	f.check(err)
	f.ctx = store.WithAutomationPolicyApproval(f.ctx, store.AutomationPolicyApproval{PrincipalID: principal.ID, SessionID: view.SessionID, HostID: f.h.ID, ConsentVersion: 1, Source: "local", HostProof: strings.Repeat("a", 64)})
	f.p, err = s.AutomationPolicyCreate(f.ctx, f.a, core.AutomationPolicyCreateInput{RequestID: f.id(), AutomationPolicySettings: core.AutomationPolicySettings{Name: "HTTP pipeline", HostID: f.h.ID, HostConsentVersion: 1, Workspace: "test", RoleBindings: bindings, IntakePrincipals: []string{principal.ID}, ControlPrincipals: []string{principal.ID}, ClientIntake: true, Checks: checks, GitPolicy: "local_commits", ExpiresAt: expires, Limits: core.AutomationLimits{PlanTasks: 1, NativeInvocations: 16, ContinuationPerStep: 1, StepSeconds: 600, WorkflowSeconds: 3600, GrantSeconds: 3600}}})
	f.check(err)
	f.poll(0)
	return f
}
