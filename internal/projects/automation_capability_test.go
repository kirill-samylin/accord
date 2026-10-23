package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/transport"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func automationREST[O any](t *testing.T, f *fixture, pid, secret, sid, operation string, input any) O {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out := f.native(t, "POST", "/api/v1/p/"+pid+"/"+operation, secret, sid, string(raw))
	if out.Code != 200 {
		t.Fatalf("%s: %d %s", operation, out.Code, out.Body.String())
	}
	var value O
	if err = json.Unmarshal(out.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAutomationRegistryDispatchCapabilityAndRevocation(t *testing.T) {
	t.Run("legacy", func(t *testing.T) { automationRegistryDispatch(t, false, false) })
	t.Run("catalog", func(t *testing.T) { automationRegistryDispatch(t, true, false) })
	t.Run("archive", func(t *testing.T) { automationRegistryDispatch(t, false, true) })
}
func automationRegistryDispatch(t *testing.T, catalog, archive bool) {
	f := setup(t)
	ctx := context.Background()
	if catalog {
		credential, who, err := f.r.CreateCredential(ctx, "owner", "machine", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		f.token, f.who = credential, who
	}
	p := f.create(t, f.who, "automation")
	sid := f.describe(t, p.ID, f.token.Secret, "human", "human").SessionID
	page := automationREST[core.ManagedPage](t, f, p.ID, f.token.Secret, sid, "managed/list", core.ManagedListInput{})
	roles := map[string][]string{}
	bindings := map[string]core.AutomationRoleBinding{}
	for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
		profile := automationREST[core.AgentProfile](t, f, p.ID, f.token.Secret, sid, "profile_create", core.ProfileCreateInput{RequestID: "profile-" + role, ProfileSettings: core.ProfileSettings{Name: role, Description: "Independent role", Role: role, Instructions: "Use genuine evidence", Provider: "openai", Model: ""}})
		roles[role] = []string{"codex"}
		bindings[role] = core.AutomationRoleBinding{Provider: "codex", ProfileID: profile.ID, ProfileVersion: profile.Version}
	}
	expiry := time.Now().UTC().Add(time.Hour)
	proof := strings.Repeat("a", 64)
	check := core.AutomationCheckPresetRef{PresetID: "unit", LocalVersion: 1, LocalDigest: strings.Repeat("b", 64), Kind: "test", Required: true, Isolation: "trusted_owner_process"}
	consent := &core.AutomationHostConsent{Enabled: true, Version: 1, ExpiresAt: expiry, Scopes: map[string]core.AutomationWorkspaceScope{"src": {Version: 1, ExpiresAt: expiry, Roles: roles, Checks: []core.AutomationCheckPresetRef{check}, MaxStepSeconds: 600, MaxWorkflowSeconds: 3600, MaxNativeInvocations: 16}}}
	host := automationREST[core.ManagedHost](t, f, p.ID, f.token.Secret, sid, "managed/hosts/enroll", core.ManagedHostEnrollInput{RequestID: "host", Name: "Test", HostSecret: proof, Controllers: []string{page.ActorPrincipalID}, Workspaces: []string{"src"}, Providers: []string{"codex"}, Automation: consent})
	policyInput := transport.AutomationPolicyLocalCreateInput{HostProof: proof, AutomationPolicyCreateInput: core.AutomationPolicyCreateInput{RequestID: "policy", AutomationPolicySettings: core.AutomationPolicySettings{Name: "Pipeline", HostID: host.ID, HostConsentVersion: 1, Workspace: "src", RoleBindings: bindings, IntakePrincipals: []string{page.ActorPrincipalID}, ControlPrincipals: []string{page.ActorPrincipalID}, ClientIntake: true, Checks: []core.AutomationCheckPresetRef{check}, GitPolicy: "local_commits", Limits: core.AutomationLimits{PlanTasks: 4, FixPerTask: 1, FixTotal: 2, NativeInvocations: 16, ContinuationPerStep: 1, StepSeconds: 600, WorkflowSeconds: 3600, GrantSeconds: 3600}, ExpiresAt: expiry}}}
	bad := policyInput
	bad.HostProof = strings.Repeat("c", 64)
	raw, _ := json.Marshal(bad)
	if out := f.native(t, "POST", "/api/v1/p/"+p.ID+"/automation/policies/create", f.token.Secret, sid, string(raw)); out.Code != 403 {
		t.Fatal("bad local proof", out.Code, out.Body.String())
	}
	policy := automationREST[core.AutomationPolicy](t, f, p.ID, f.token.Secret, sid, "automation/policies/create", policyInput)
	badRevoke := transport.AutomationPolicyLocalRevokeInput{AutomationPolicyRevokeInput: core.AutomationPolicyRevokeInput{RequestID: "bad-revoke", PolicyID: policy.ID, Version: policy.Version, Reason: "test"}, HostProof: strings.Repeat("f", 64), HostConsentVersion: 1}
	raw, _ = json.Marshal(badRevoke)
	if out := f.native(t, "POST", "/api/v1/p/"+p.ID+"/automation/policies/revoke", f.token.Secret, sid, string(raw)); out.Code != 403 {
		t.Fatal("bad revoke proof", out.Code, out.Body.String())
	}
	pollInput := core.ManagedPollInput{RequestID: "poll-initial", HostID: host.ID, HostSecret: proof, WorkerID: "worker", WorkerCapabilities: map[string]int{"automation": 1}}
	automationREST[core.ManagedPollResult](t, f, p.ID, f.token.Secret, sid, "managed/poll", pollInput)
	workflow := automationREST[core.WorkflowPage](t, f, p.ID, f.token.Secret, sid, "automation/submit", core.WorkflowSubmitInput{RequestID: "submit", PolicyID: policy.ID, PolicyVersion: policy.Version, Text: "Implement requirement with tests"})
	if _, err := f.r.automationCycle(ctx, 0, map[string]int{}); err != nil {
		t.Fatal(err)
	}
	pollInput.RequestID = "poll-start"
	poll := automationREST[core.ManagedPollResult](t, f, p.ID, f.token.Secret, sid, "managed/poll", pollInput)
	if poll.Command == nil || poll.Run == nil || poll.Run.Automation.WorkflowID != workflow.Workflow.ID {
		t.Fatal("scheduler did not dispatch workflow", poll)
	}
	pollInput.WorkerEpoch = poll.WorkerEpoch
	pollInput.RequestID = "accept"
	automationREST[core.ManagedRun](t, f, p.ID, f.token.Secret, sid, "managed/report", core.ManagedReportInput{ManagedPollInput: pollInput, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, Phase: "accepted"})
	token := core.ExecutionTokenPrefix + strings.Repeat("d", 64)
	digest := sha256.Sum256([]byte(token))
	binding := poll.Run.Automation
	pollInput.RequestID = "cap"
	automationREST[core.ExecutionCapability](t, f, p.ID, f.token.Secret, sid, "automation/capabilities/register", core.ExecutionCapabilityRegisterInput{ManagedPollInput: pollInput, CommandID: poll.Command.ID, DispatchID: poll.Command.DispatchID, RunID: poll.Run.ID, Generation: poll.Run.Generation, WorkflowID: binding.WorkflowID, StepID: binding.StepID, ControlEpoch: binding.ControlEpoch, CapabilityID: "ec_" + strings.Repeat("e", 32), Digest: hex.EncodeToString(digest[:])})
	authctx, _, autherr := f.r.nativeIdentity(ctx, token, p.ID, "/api/v1/session", true)
	if autherr != nil {
		t.Fatal("router cap", autherr)
	}
	runtime, autherr := f.r.runtime(p.ID)
	if autherr != nil {
		t.Fatal(autherr)
	}
	if _, autherr = runtime.service.Authenticate(authctx, token); autherr != nil {
		t.Fatal("service cap", autherr)
	}
	if archive {
		// Archiving must still deliver stop and accept an owned terminal report.
		if _, err := f.r.updateProject(ctx, f.who, p.ID, p.Name, "archived", p.Version); err != nil {
			t.Fatal(err)
		}
		if _, err := f.r.automationCycle(ctx, 0, map[string]int{}); err != nil {
			t.Fatal(err)
		}
		pollInput.RequestID = "archived-poll"
		stopped := automationREST[core.ManagedPollResult](t, f, p.ID, f.token.Secret, sid, "managed/poll", pollInput)
		if stopped.Command == nil || stopped.Command.Kind != "stop" {
			t.Fatal("archive did not deliver stop", stopped)
		}
		pollInput.RequestID = "stop-accepted"
		automationREST[core.ManagedResult](t, f, p.ID, f.token.Secret, sid, "managed/report", core.ManagedReportInput{ManagedPollInput: pollInput, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: stopped.Command.ID, DispatchID: stopped.Command.DispatchID, Phase: "accepted"})
		pollInput.RequestID = "stop-terminal"
		automationREST[core.ManagedResult](t, f, p.ID, f.token.Secret, sid, "managed/report", core.ManagedReportInput{ManagedPollInput: pollInput, RunID: poll.Run.ID, Generation: poll.Run.Generation, CommandID: stopped.Command.ID, DispatchID: stopped.Command.DispatchID, Phase: "stopped"})
		return
	}
	// Omitted run/profile fields are forced by the capability, not expanded.
	child := automationREST[core.SessionView](t, f, p.ID, token, "", "session", core.OpenSession{RequestID: "child"})
	if child.Role != "analyst" {
		t.Fatal("role not forced", child)
	}
	claim := automationREST[core.Claim](t, f, p.ID, token, child.SessionID, "tasks/next", core.NextInput{RequestID: "claim", TaskKey: poll.Run.TaskKey})
	if claim.Task == nil || claim.Task.Key != poll.Run.TaskKey {
		t.Fatal("wrong task claimed")
	}
	for _, op := range []string{"automation/list", "managed/list", "profile_create"} {
		out := f.native(t, "POST", "/api/v1/p/"+p.ID+"/"+op, token, child.SessionID, `{}`)
		if out.Code == 200 {
			t.Fatal("child privilege expanded", op)
		}
	}
	raw, _ = json.Marshal(core.WorkflowReadInput{WorkflowID: workflow.Workflow.ID})
	if out := f.native(t, "POST", "/api/v1/p/"+p.ID+"/automation/get", token, child.SessionID, string(raw)); out.Code != 200 {
		t.Fatal("own workflow denied", out.Code, out.Body.String())
	}

	// Revoke the underlying source credential while the execution token survives.
	var revokeErr error
	if catalog {
		revokeErr = f.r.RevokeCredential(ctx, f.token.ID)
	} else {
		revokeErr = f.store.RevokeToken(ctx, f.token.ID)
	}
	if revokeErr != nil {
		t.Fatal(revokeErr)
	}
	out := f.native(t, "GET", fmt.Sprintf("/api/v1/p/%s/tasks/%s", p.ID, poll.Run.TaskKey), token, child.SessionID, "")
	if out.Code == 200 {
		t.Fatal("capability survived owner credential revoke")
	}
}
