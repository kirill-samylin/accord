package store

import (
	"agent-board/internal/core"
	"context"
	"testing"
	"time"
)

func TestAutomationCurrentAuthorityBeforeStableReplayAndChildBoundary(t *testing.T) {
	m := automationSetup(t)
	in := core.WorkflowSubmitInput{RequestID: "stable-requirement", PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: "Ship safely"}
	first, e := m.f.s.WorkflowSubmit(m.ctx, m.a, in)
	if e != nil {
		t.Fatal(e)
	}
	a2 := m.f.actor(t, "human")
	again, e := m.f.s.WorkflowSubmit(m.ctx, a2, in)
	if e != nil || first.Workflow.ID != again.Workflow.ID {
		t.Fatal("session-independent replay", e)
	}
	revoked := WithAutomationAuthority(m.ctx, AutomationAuthority{ProjectID: "test", Writable: true, Principals: map[string]AutomationPrincipal{}})
	if _, e = m.f.s.WorkflowSubmit(revoked, a2, in); e == nil {
		t.Fatal("revoked replay accepted")
	}
	id := first.Workflow.ID
	m.tick(t, id)
	attempt := m.attempt(t, id)
	if _, e = m.f.s.WorkflowSubmit(attempt.ctx, attempt.actor, core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: "recursive"}); e == nil {
		t.Fatal("child recursiveintake")
	}
	if _, e = m.f.s.Create(attempt.ctx, attempt.actor, core.CreateInput{RequestID: m.f.id(), Type: "epic", Title: "escalation"}); e == nil {
		t.Fatal("child admincreate")
	}
	if _, e = m.f.s.Describe(attempt.ctx, m.f.p, core.OpenSession{RequestID: m.f.id(), Role: "human", RunID: "unbound", Machine: m.f.p.Machine}); e == nil {
		t.Fatal("child unbound human")
	}
	if _, e = m.f.s.Next(attempt.ctx, attempt.actor, core.NextInput{RequestID: m.f.id()}); e == nil {
		t.Fatal("child arbitraryqueue")
	}
	if _, e = m.f.s.Get(attempt.ctx, attempt.actor, core.GetInput{TaskKey: m.f.epic.Key}); e == nil {
		t.Fatal("child foreign taskread")
	}
	if _, e = m.f.s.Get(m.ctx, attempt.actor, core.GetInput{TaskKey: attempt.claim.Task.Key}); e == nil {
		t.Fatal("owner bearer reusedscoped session")
	}
	page := m.page(t, id)
	paused, e := m.f.s.WorkflowCommand(m.ctx, m.a, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, Kind: "pause"})
	if e != nil {
		t.Fatal(e)
	}
	if paused.Workflow.State != "pausing" {
		t.Fatal("unproven stop shownpaused")
	}
	if _, e = m.f.s.Heartbeat(attempt.ctx, attempt.actor, core.HeartbeatInput{RequestID: m.f.id(), TaskKey: attempt.claim.Task.Key, Generation: attempt.claim.Lease.Generation}); e == nil {
		t.Fatal("paused child stillrenews")
	}
}
func TestAutomationPolicyCannotCrossWorkspaceConsentOrCreateWithoutHumanApproval(t *testing.T) {
	m := automationSetup(t)
	in := core.AutomationPolicyCreateInput{RequestID: m.f.id(), AutomationPolicySettings: m.p.AutomationPolicySettings}
	in.Name = "second"
	without := WithAutomationPolicyApproval(m.ctx, AutomationPolicyApproval{})
	if _, e := m.f.s.AutomationPolicyCreate(without, m.a, in); e == nil {
		t.Fatal("Describehuman becamepolicyapproval")
	}
	in.Workspace = "unconsented"
	if _, e := m.f.s.AutomationPolicyCreate(m.ctx, m.a, in); e == nil {
		t.Fatal("workspace crossproduct")
	}
	in.Workspace = "src"
	in.RoleBindings = map[string]core.AutomationRoleBinding{}
	for k, v := range m.p.RoleBindings {
		in.RoleBindings[k] = v
	}
	binding := in.RoleBindings["developer"]
	binding.Provider = "claude"
	in.RoleBindings["developer"] = binding
	if _, e := m.f.s.AutomationPolicyCreate(m.ctx, m.a, in); e == nil {
		t.Fatal("provider crossproduct")
	}
}
func TestAutomationExpiredReceiptDoesNotRenewOrEnableFreshMutation(t *testing.T) {
	m := automationSetup(t)
	in := core.WorkflowSubmitInput{RequestID: "receipt", Text: "draft"}
	first, e := m.f.s.WorkflowSubmit(m.ctx, m.a, in)
	if e != nil {
		t.Fatal(e)
	}
	m.f.clock.advance(2 * time.Hour)
	again, e := m.f.s.WorkflowSubmit(m.ctx, m.a, in)
	if e != nil || again.Workflow.ID != first.Workflow.ID {
		t.Fatal("expiredreceipt", e)
	}
	in.RequestID = "new"
	if _, e = m.f.s.WorkflowSubmit(m.ctx, m.a, in); e == nil {
		t.Fatal("expiredsession newmutation")
	}
	var expiry time.Time
	_, e = transact(m.f.s, m.ctx, func(b *board, now time.Time) (bool, error) {
		expiry = b.Sessions[m.a.SessionID].ExpiresAt
		return true, nil
	})
	if e != nil || expiry.After(m.f.clock.read()) {
		t.Fatal("receipt reanimated session")
	}
}
func TestAutomationLimitsPlanMandatoryTestsAndCapacityAtomicity(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	page := m.page(t, id)
	_, e := m.f.s.WorkflowPlan(a.ctx, a.actor, core.WorkflowPlanInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Tasks: []core.WorkflowPlanTask{{ClientKey: "one", Title: "Missing tests", Description: "implement", Acceptance: []string{"works"}}}})
	if e == nil {
		t.Fatal("vacuous QA plan")
	}
	_, e = m.f.s.WorkflowPlan(a.ctx, a.actor, core.WorkflowPlanInput{RequestID: m.f.id(), WorkflowID: id, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Tasks: []core.WorkflowPlanTask{{ClientKey: "one", Title: "Cycle", Description: "implement", Acceptance: []string{"works"}, RequiredCheckIDs: []string{"unit"}, Prerequisites: []string{"one"}}}})
	if e == nil {
		t.Fatal("cyclicplan")
	}
	if got := m.page(t, id); got.Workflow.PlanRevision != 0 {
		t.Fatal("rejectedplan partially committed")
	}
	// Host floor rejection is transactional: no partial epic/task/run is created.
	n := automationSetup(t)
	flow := n.submit(t)
	_, e = transact(n.f.s, n.ctx, func(b *board, now time.Time) (bool, error) {
		b.ManagedHosts[n.h.ID].Host.WorkerCapabilities = nil
		return true, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	blocked := n.tick(t, flow.Workflow.ID)
	if blocked.Workflow.State != "blocked" || blocked.Workflow.EpicKey != "" || len(blocked.Steps) != 0 {
		t.Fatal("partial incompatible launch")
	}
}
func TestAutomationViewerReadDoesNotPreserveWriterDelegation(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	read := WithAutomationAuthority(context.Background(), AutomationAuthority{ProjectID: "test", ReadPrincipals: map[string]AutomationPrincipal{m.f.p.ID: {Principal: m.f.p, ExpiresAt: m.f.clock.read().Add(time.Hour)}}, Principals: map[string]AutomationPrincipal{}})
	if _, e := m.f.s.WorkflowGet(read, m.a, core.WorkflowReadInput{WorkflowID: w.Workflow.ID}); e != nil {
		t.Fatal("viewerread", e)
	}
	if _, e := m.f.s.WorkflowSubmit(read, m.a, core.WorkflowSubmitInput{RequestID: m.f.id(), Text: "forbidden"}); e == nil {
		t.Fatal("viewerwrite")
	}
	_, e := m.f.s.AutomationTick(WithAutomationAuthority(read, AutomationAuthority{ProjectID: "test", Writable: true, Principals: map[string]AutomationPrincipal{}}), core.WorkflowAdvanceInput{WorkflowID: w.Workflow.ID, ExpectedVersion: w.Workflow.Version, ControlEpoch: w.Workflow.ControlEpoch})
	if e != nil {
		t.Fatal(e)
	}
	got := m.page(t, w.Workflow.ID)
	if got.Workflow.State != "paused" || len(got.Steps) != 0 {
		t.Fatal("viewergrant continued")
	}
}

func TestAutomationManualManagedChildCannotDraftWithoutPolicy(t *testing.T) {
	m := automationSetup(t)
	binding := m.p.RoleBindings["developer"]
	r, e := m.f.s.ManagedStart(m.ctx, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "src", Provider: "codex", ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion, Prompt: "manual work"})
	if e != nil {
		t.Fatal(e)
	}
	poll, e := m.f.s.ManagedPoll(m.ctx, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.f.s.ManagedReport(m.ctx, m.a, core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "fixture-worker", WorkerEpoch: poll.WorkerEpoch}, RunID: r.Run.ID, Generation: r.Run.Generation, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, Phase: "accepted"}); e != nil {
		t.Fatal(e)
	}
	sess, e := m.f.s.Describe(m.ctx, m.f.p, core.OpenSession{RequestID: m.f.id(), RunID: r.Run.ID, Machine: m.f.p.Machine, Role: "developer", ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion, ManagedRunID: r.Run.ID, ManagedGeneration: 1})
	if e != nil {
		t.Fatal(e)
	}
	_, e = m.f.s.WorkflowSubmit(m.ctx, core.Actor{Principal: m.f.p, SessionID: sess.SessionID}, core.WorkflowSubmitInput{RequestID: m.f.id(), Text: "recursive draft"})
	if e == nil {
		t.Fatal("manual managed child draftedworkflow")
	}
}
func TestAutomationUnboundNativeIntakeAndControlArePolicyScoped(t *testing.T) {
	m := automationSetup(t)
	agent := m.f.actor(t, "developer")
	w, e := m.f.s.WorkflowSubmit(m.ctx, agent, core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: "Native client request"})
	if e != nil {
		t.Fatal(e)
	}
	paused, e := m.f.s.WorkflowCommand(m.ctx, agent, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: w.Workflow.Version, Kind: "pause"})
	if e != nil || paused.Workflow.State != "paused" {
		t.Fatal("unboundpolicycontrol", e)
	}
	_, e = m.f.s.WorkflowCommand(m.ctx, agent, core.WorkflowCommandInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: paused.Workflow.Version, Kind: "resume", RefreshPolicyID: m.p.ID, RefreshPolicyVersion: m.p.Version})
	if e == nil {
		t.Fatal("native client escalatedtohumanregrant")
	}
}
func TestAutomationNoUnboundProfileSessionMayClaimWorkflowTask(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	w = m.tick(t, w.Workflow.ID)
	binding := m.p.RoleBindings["analyst"]
	view, e := m.f.s.Describe(m.ctx, m.f.p, core.OpenSession{RequestID: m.f.id(), RunID: "unbound", Machine: m.f.p.Machine, Role: "analyst", ProfileID: binding.ProfileID, ProfileVersion: binding.ProfileVersion})
	if e != nil {
		t.Fatal(e)
	}
	claim, e := m.f.s.Next(m.ctx, core.Actor{Principal: m.f.p, SessionID: view.SessionID}, core.NextInput{RequestID: m.f.id(), TaskKey: w.Steps[0].TaskKey})
	if e != nil {
		t.Fatal(e)
	}
	if claim.Task != nil {
		t.Fatal("unboundprofileclaimedscopedtask")
	}
}

func TestAutomationOptionalZeroBudgetsAreNeverExpanded(t *testing.T) {
	v := automationLimits(core.AutomationLimits{PlanTasks: 1, NativeInvocations: 4, StepSeconds: 60, WorkflowSeconds: 600, GrantSeconds: 3600})
	if v.FixPerTask != 0 || v.FixTotal != 0 || v.ContinuationPerStep != 0 {
		t.Fatal("zero local retries silentlyexpanded")
	}
}

func TestAutomationDisabledClientIntakeRequiresTrustedBrowserProofEvenHumanRole(t *testing.T) {
	m := automationSetup(t)
	settings := m.p.AutomationPolicySettings
	settings.ClientIntake = false
	p, e := m.f.s.AutomationPolicyUpdate(m.ctx, m.a, core.AutomationPolicyUpdateInput{AutomationPolicyCreateInput: core.AutomationPolicyCreateInput{RequestID: m.f.id(), AutomationPolicySettings: settings}, PolicyID: m.p.ID, Version: m.p.Version})
	if e != nil {
		t.Fatal(e)
	}
	in := core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: p.ID, PolicyVersion: p.Version, Text: "Human role alone is not a browser channel"}
	permission := core.AutomationPermissionInput{PolicyID: p.ID, Kind: "submit"}
	if _, e = m.f.s.WorkflowSubmit(m.ctx, m.a, in); e == nil {
		t.Fatal("native Describehuman bypasseddisabledintake")
	}
	if e = m.f.s.AutomationPermission(m.ctx, m.a, permission); e == nil {
		t.Fatal("cachedpermissiontrustedhumanrole")
	}
	for _, approval := range []AutomationIntakeApproval{{PrincipalID: "other", SessionID: m.a.SessionID, ActionID: "issued"}, {PrincipalID: m.f.p.ID, SessionID: "other", ActionID: "issued"}, {PrincipalID: m.f.p.ID, SessionID: m.a.SessionID}} {
		ctx := WithAutomationIntakeApproval(m.ctx, approval)
		if _, e = m.f.s.WorkflowSubmit(ctx, m.a, in); e == nil {
			t.Fatal("invalid UIintakeproof accepted")
		}
	}
	ui := WithAutomationIntakeApproval(m.ctx, AutomationIntakeApproval{PrincipalID: m.f.p.ID, SessionID: m.a.SessionID, ActionID: "server-issued"})
	if e = m.f.s.AutomationPermission(ui, m.a, permission); e != nil {
		t.Fatal(e)
	}
	flow, e := m.f.s.WorkflowSubmit(ui, m.a, in)
	if e != nil || flow.Workflow.State != "queued" {
		t.Fatal("trustedUIintakedenied", e)
	}
}
