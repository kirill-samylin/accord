package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type automationReceiptFixture struct {
	f      *webFixture
	ctx    context.Context
	cookie *http.Cookie
	a      core.Actor
	host   core.ManagedHost
	policy core.AutomationPolicy
	create url.Values
	poll   core.ManagedPollInput
}

func automationReceiptSetup(t *testing.T, scopeTTL time.Duration) *automationReceiptFixture {
	t.Helper()
	f, ctx, cookie, a := automationBrowserSetup(t)
	ctx = store.WithManagedPrincipals(ctx, map[string]store.ManagedPrincipalGrant{a.Principal.ID: {Epoch: 0, LegacyTokenID: a.Principal.ID}})
	expiry := f.now.Add(24 * time.Hour)
	scopeExpiry := f.now.Add(scopeTTL)
	check := core.AutomationCheckPresetRef{PresetID: "unit", LocalVersion: 1, LocalDigest: strings.Repeat("a", 64), Kind: "test", Required: true, Isolation: "trusted_owner_process"}
	scope := core.AutomationWorkspaceScope{Version: 1, ExpiresAt: scopeExpiry, Roles: map[string][]string{}, Checks: []core.AutomationCheckPresetRef{check}, MaxStepSeconds: 1800, MaxWorkflowSeconds: 28800, MaxNativeInvocations: 40}
	bindings := map[string]core.AutomationRoleBinding{}
	for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
		scope.Roles[role] = []string{"codex"}
		p, e := f.board.ProfileCreate(ctx, a, core.ProfileCreateInput{RequestID: "profile-" + role, ProfileSettings: core.ProfileSettings{Name: role, Role: role, Provider: "unconfigured"}})
		if e != nil {
			t.Fatal(e)
		}
		bindings[role] = core.AutomationRoleBinding{Provider: "codex", ProfileID: p.ID, ProfileVersion: p.Version}
	}
	host, e := f.board.ManagedHostEnroll(ctx, a, core.ManagedHostEnrollInput{RequestID: "enroll", Name: "Receipt test", HostSecret: strings.Repeat("b", 64), Workspaces: []string{"main"}, Providers: []string{"codex"}, Automation: &core.AutomationHostConsent{Enabled: true, Version: 1, ExpiresAt: expiry, Scopes: map[string]core.AutomationWorkspaceScope{"main": scope}}})
	if e != nil {
		t.Fatal(e)
	}
	settings := core.AutomationPolicySettings{Name: "Receipt policy", HostID: host.ID, HostConsentVersion: 1, Workspace: "main", RoleBindings: bindings, IntakePrincipals: []string{a.Principal.ID}, ControlPrincipals: []string{a.Principal.ID}, Checks: []core.AutomationCheckPresetRef{check}, GitPolicy: "local_commits", Limits: core.AutomationLimits{PlanTasks: 8, FixPerTask: 2, FixTotal: 2, NativeInvocations: 40, ContinuationPerStep: 1, StepSeconds: 1800, WorkflowSeconds: 28800, GrantSeconds: 86400}, ExpiresAt: scopeExpiry}
	page := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	raw, _ := json.Marshal(settings)
	fields := jsonBrowserFields(page, "automation:policy-create", url.Values{"settings": {string(raw)}})
	created := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/policies/create", fields, cookie), 200)
	detail := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", created.Redirect, nil, cookie), 200).Page
	return &automationReceiptFixture{f: f, ctx: ctx, cookie: cookie, a: a, host: host, policy: detail.AutomationPolicy.Policy, create: fields, poll: core.ManagedPollInput{HostID: host.ID, HostSecret: strings.Repeat("b", 64), WorkerID: "receipt-worker", WorkerCapabilities: map[string]int{"automation": 1}}}
}
func (m *automationReceiptFixture) page(t *testing.T, id string) browserPage {
	t.Helper()
	return decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation/workflows/"+id, nil, m.cookie), 200).Page
}
func (m *automationReceiptFixture) submit(t *testing.T) core.WorkflowPage {
	t.Helper()
	page := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation", nil, m.cookie), 200).Page
	fields := jsonBrowserFields(page, "automation:submit", url.Values{"policy_id": {m.policy.ID}, "policy_version": {strconv.FormatInt(m.policy.Version, 10)}, "text": {"Verify receipt authority"}})
	created := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/submit", fields, m.cookie), 200)
	detail := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", created.Redirect, nil, m.cookie), 200).Page
	return *detail.Workflow
}
func (m *automationReceiptFixture) command(t *testing.T, w core.Workflow, kind string) core.WorkflowPage {
	t.Helper()
	p, e := m.f.board.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: kind + "-" + w.ID, WorkflowID: w.ID, Version: w.Version, Kind: kind})
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func (m *automationReceiptFixture) updatePolicy(t *testing.T) {
	t.Helper()
	p := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation/policies/"+m.policy.ID, nil, m.cookie), 200).Page
	settings := m.policy.AutomationPolicySettings
	settings.Name += " revised"
	raw, _ := json.Marshal(settings)
	fields := jsonBrowserFields(p, "automation:policy-update", url.Values{"settings": {string(raw)}, "policy_id": {m.policy.ID}, "version": {strconv.FormatInt(m.policy.Version, 10)}})
	out := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/policies/update", fields, m.cookie), 200)
	p = decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", out.Redirect, nil, m.cookie), 200).Page
	m.policy = p.AutomationPolicy.Policy
}
func (m *automationReceiptFixture) question(t *testing.T, w core.Workflow) core.WorkflowPage {
	t.Helper()
	poll := m.poll
	poll.RequestID = "poll-enroll"
	if _, e := m.f.board.ManagedPoll(m.ctx, m.a, poll); e != nil {
		t.Fatal(e)
	}
	if _, e := m.f.board.AutomationTick(m.ctx, core.WorkflowAdvanceInput{WorkflowID: w.ID, ExpectedVersion: w.Version, ControlEpoch: w.ControlEpoch}); e != nil {
		t.Fatal(e)
	}
	poll.RequestID = "poll-run"
	dispatch, e := m.f.board.ManagedPoll(m.ctx, m.a, poll)
	if e != nil || dispatch.Command == nil {
		t.Fatal("dispatch", e)
	}
	report := core.ManagedReportInput{ManagedPollInput: m.poll, CommandID: dispatch.Command.ID, DispatchID: dispatch.Command.DispatchID, RunID: dispatch.Run.ID, Generation: dispatch.Run.Generation, Phase: "accepted"}
	report.WorkerEpoch = dispatch.WorkerEpoch
	report.RequestID = "accepted"
	if _, e = m.f.board.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	raw := core.ExecutionTokenPrefix + strings.Repeat("c", 64)
	digest := sha256.Sum256([]byte(raw))
	binding := dispatch.Run.Automation
	_, e = m.f.board.ExecutionCapabilityRegister(m.ctx, m.a, core.ExecutionCapabilityRegisterInput{ManagedPollInput: core.ManagedPollInput{RequestID: "register", HostID: m.host.ID, HostSecret: m.poll.HostSecret, WorkerID: m.poll.WorkerID, WorkerEpoch: dispatch.WorkerEpoch}, CommandID: dispatch.Command.ID, DispatchID: dispatch.Command.DispatchID, RunID: dispatch.Run.ID, Generation: dispatch.Run.Generation, WorkflowID: w.ID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, CapabilityID: "ec_" + strings.Repeat("c", 32), Digest: hex.EncodeToString(digest[:])})
	if e != nil {
		t.Fatal(e)
	}
	identity, e := m.f.board.AuthenticateExecutionCapability(m.ctx, raw)
	if e != nil {
		t.Fatal(e)
	}
	child := store.WithExecutionCapability(m.ctx, identity)
	session, e := m.f.board.Describe(child, m.a.Principal, core.OpenSession{RequestID: "child-describe", RunID: dispatch.Run.ID, Machine: m.a.Principal.Machine, Role: "analyst", ProfileID: dispatch.Run.Profile.ID, ProfileVersion: dispatch.Run.Profile.Version, ManagedRunID: dispatch.Run.ID, ManagedGeneration: dispatch.Run.Generation})
	if e != nil {
		t.Fatal(e)
	}
	actor := core.Actor{Principal: m.a.Principal, SessionID: session.SessionID}
	claim, e := m.f.board.Next(child, actor, core.NextInput{RequestID: "child-claim", TaskKey: dispatch.Run.TaskKey})
	if e != nil || claim.Task == nil {
		t.Fatal("claim", e)
	}
	report.RequestID = "started"
	report.Phase = "started"
	report.ProviderSessionID = "11111111-1234-1234-1234-123456789012"
	if _, e = m.f.board.ManagedReport(m.ctx, m.a, report); e != nil {
		t.Fatal(e)
	}
	current := m.page(t, w.ID)
	asked, e := m.f.board.WorkflowQuestion(child, actor, core.WorkflowQuestionInput{RequestID: "ask", WorkflowID: w.ID, Version: current.Workflow.Workflow.Version, TaskGeneration: claim.Lease.Generation, Question: "Choose behavior", Options: []string{"A", "B"}})
	if e != nil {
		t.Fatal(e)
	}
	stopPoll := m.poll
	stopPoll.WorkerEpoch = dispatch.WorkerEpoch
	stopPoll.RequestID = "poll-stop"
	stopped, e := m.f.board.ManagedPoll(m.ctx, m.a, stopPoll)
	if e != nil || stopped.Command == nil || stopped.Command.Kind != "stop" {
		t.Fatal("stop", e)
	}
	stopReport := core.ManagedReportInput{ManagedPollInput: stopPoll, CommandID: stopped.Command.ID, DispatchID: stopped.Command.DispatchID, RunID: stopped.Run.ID, Generation: stopped.Run.Generation, Phase: "accepted"}
	stopReport.RequestID = "stop-accepted"
	if _, e = m.f.board.ManagedReport(m.ctx, m.a, stopReport); e != nil {
		t.Fatal(e)
	}
	stopReport.RequestID = "stop-finished"
	stopReport.Phase = "stopped"
	if _, e = m.f.board.ManagedReport(m.ctx, m.a, stopReport); e != nil {
		t.Fatal(e)
	}
	return asked
}
