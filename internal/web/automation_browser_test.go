package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The test supplies registry authority directly; production receives it only from
// projects.Service while registry/legacy write guards are held.
func automationBrowserRequest(f *webFixture, ctx context.Context, method, path string, fields url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	body := ""
	if fields != nil {
		body = fields.Encode()
	}
	r := httptest.NewRequest(method, "https://accord.test"+path, strings.NewReader(body)).WithContext(ctx)
	r.Header.Set("Accept", "application/json")
	if fields != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "https://accord.test")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	return w
}
func automationBrowserSetup(t *testing.T) (*webFixture, context.Context, *http.Cookie, core.Actor) {
	t.Helper()
	f := fixture(t)
	token := f.token(t, "automation-owner")
	cookie := f.login(t, token)
	principal, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	ctx := store.WithAutomationAuthority(context.Background(), store.AutomationAuthority{ProjectID: "synthetic-project", Writable: true, Principals: map[string]store.AutomationPrincipal{principal.ID: {Principal: principal, Epoch: 0, LegacyTokenID: principal.ID, ExpiresAt: f.now.Add(time.Hour)}}})
	p := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if p.Automation == nil || p.Managed == nil {
		t.Fatal("missing actual page data")
	}
	return f, ctx, cookie, core.Actor{Principal: principal, SessionID: p.Session.SessionID}
}
func TestAutomationBrowserDraftExactReceiptAndDistinctTeamCreates(t *testing.T) {
	f, ctx, cookie, _ := automationBrowserSetup(t)
	page := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	ids := map[string]bool{}
	for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
		id := page.Actions["automation:team:"+role]
		if len(id) != 64 || ids[id] {
			t.Fatal("team must have four independent create actions")
		}
		ids[id] = true
		fields := url.Values{"csrf": {page.CSRF}, "action_id": {id}, "name": {role}, "role": {role}, "description": {"team"}, "instructions": {"Bound work only"}, "provider": {"unconfigured"}, "model": {""}}
		first := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/profiles", fields, cookie), 200)
		again := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/profiles", fields, cookie), 200)
		if first.Redirect == "" || first.Redirect != again.Redirect {
			t.Fatal("profile retry created a different result")
		}
	}
	page = decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if len(page.Profiles) != 4 {
		t.Fatalf("partial setup lost profiles: %d", len(page.Profiles))
	}
	fields := jsonBrowserFields(page, "automation:submit", url.Values{"text": {"Implement a page"}})
	bad := cloneAutomationValues(fields)
	bad.Set("csrf", "wrong")
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", bad, cookie), 403)
	first := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 200)
	again := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 200)
	if first.Redirect == "" || first.Redirect != again.Redirect {
		t.Fatal("draft retry created another workflow")
	}
	detail := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", first.Redirect, nil, cookie), 200).Page
	if detail.Workflow == nil || detail.Workflow.Workflow.Grant != nil || detail.Workflow.Workflow.State != "awaiting_authorization" || len(detail.Workflow.Steps) != 0 || !detail.Workflow.CanControl {
		t.Fatal("draft was dispatched or cannot be explicitly authorized")
	}
	changed := cloneAutomationValues(fields)
	changed.Set("text", "Different requirement")
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", changed, cookie), 409)
	list := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if len(list.Automation.Workflows) != 1 || len(list.Managed.Runs) != 0 {
		t.Fatal("browser retry dispatched hidden work")
	}
}
func TestAutomationBrowserPolicyApprovalAndRevokedHostCachedReceipt(t *testing.T) {
	f, ctx, cookie, a := automationBrowserSetup(t)
	roles := []string{"analyst", "developer", "reviewer", "tester"}
	check := core.AutomationCheckPresetRef{PresetID: "tests", LocalVersion: 1, LocalDigest: strings.Repeat("a", 64), Kind: "test", Required: true, Isolation: "trusted_owner_process"}
	expires := f.now.UTC().Add(time.Hour)
	scope := core.AutomationWorkspaceScope{Version: 1, ExpiresAt: expires, Roles: map[string][]string{}, Checks: []core.AutomationCheckPresetRef{check}, MaxStepSeconds: 1800, MaxWorkflowSeconds: 28800, MaxNativeInvocations: 40}
	bindings := map[string]core.AutomationRoleBinding{}
	for _, role := range roles {
		scope.Roles[role] = []string{"codex"}
		p, err := f.board.ProfileCreate(ctx, a, core.ProfileCreateInput{RequestID: "profile-" + role, ProfileSettings: core.ProfileSettings{Name: role, Role: role, Provider: "unconfigured"}})
		if err != nil {
			t.Fatal(err)
		}
		bindings[role] = core.AutomationRoleBinding{Provider: "codex", ProfileID: p.ID, ProfileVersion: p.Version}
	}
	h, err := f.board.ManagedHostEnroll(ctx, a, core.ManagedHostEnrollInput{RequestID: "local-enroll", Name: "synthetic-host", HostSecret: strings.Repeat("b", 64), Workspaces: []string{"main"}, Providers: []string{"codex"}, Controllers: []string{}, Automation: &core.AutomationHostConsent{Enabled: true, Version: 1, ExpiresAt: expires, Scopes: map[string]core.AutomationWorkspaceScope{"main": scope}}})
	if err != nil {
		t.Fatal(err)
	}
	settings := core.AutomationPolicySettings{Name: "Team", HostID: h.ID, HostConsentVersion: 1, Workspace: "main", RoleBindings: bindings, IntakePrincipals: []string{a.Principal.ID}, ControlPrincipals: []string{a.Principal.ID}, Checks: []core.AutomationCheckPresetRef{check}, GitPolicy: "local_commits", Limits: core.AutomationLimits{PlanTasks: 8, FixPerTask: 2, FixTotal: 2, NativeInvocations: 40, ContinuationPerStep: 1, StepSeconds: 1800, WorkflowSeconds: 28800, GrantSeconds: 86400}, ExpiresAt: expires}
	if _, err = f.board.AutomationPolicyCreate(ctx, a, core.AutomationPolicyCreateInput{RequestID: "untrusted-native", AutomationPolicySettings: settings}); err == nil {
		t.Fatal("human session alone created automation policy")
	}
	page := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	raw, _ := json.Marshal(settings)
	fields := jsonBrowserFields(page, "automation:policy-create", url.Values{"settings": {string(raw)}})
	first := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/policies/create", fields, cookie), 200)
	replay := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/policies/create", fields, cookie), 200)
	if first.Redirect == "" || first.Redirect != replay.Redirect {
		t.Fatal("policy retry changed receipt")
	}
	detail := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", first.Redirect, nil, cookie), 200).Page
	if detail.AutomationPolicy == nil || detail.AutomationPolicy.Policy.IssuerPrincipalID != a.Principal.ID || len(detail.AutomationPolicy.Revisions) != 1 {
		t.Fatal("browser policy missing immutable issuer/revision")
	}
	policyID := detail.AutomationPolicy.Policy.ID
	page = decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	invalid := jsonBrowserFields(page, "automation:submit", url.Values{"text": {"Rejected once"}, "policy_id": {policyID}, "policy_version": {"2"}})
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", invalid, cookie), 409)
	settings.Name = "Team revision two"
	updatedSettings, _ := json.Marshal(settings)
	update := jsonBrowserFields(detail, "automation:policy-update", url.Values{"settings": {string(updatedSettings)}, "policy_id": {policyID}, "version": {"1"}})
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/policies/update", update, cookie), 200)
	// The formerly rejected request would now satisfy the version precondition.
	// A cached rejection must never turn into an unobserved new workflow.
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", invalid, cookie), 409)
	after := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if len(after.Automation.Workflows) != 0 {
		t.Fatal("cached rejection later executed hidden work")
	}
	if _, err = f.board.ManagedHostDisable(ctx, a, core.ManagedHostDisableInput{RequestID: "disable", HostID: h.ID, Version: h.Version}); err != nil {
		t.Fatal(err)
	}
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/policies/create", fields, cookie), 403)
}
