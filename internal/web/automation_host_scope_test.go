package web

import (
	"agent-board/internal/core"
	"context"
	"net/url"
	"testing"
	"time"
)

func TestAutomationHostScopeBrowserReadIsExactAndReadOnly(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	summary := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation", nil, m.cookie), 200).Page
	if summary.Managed == nil || len(summary.Managed.Hosts) != 1 || !summary.Managed.Hosts[0].Automation.ScopesTruncated || len(summary.Managed.Hosts[0].Automation.Scopes) != 0 {
		t.Fatal("summary exposed full scope")
	}
	path := "/web/automation/host-scope?" + url.Values{"host_id": {m.host.ID}, "workspace": {"main"}}.Encode()
	exact := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", path, nil, m.cookie), 200).Page
	if exact.Managed == nil || len(exact.Managed.Hosts) != 1 || len(exact.Managed.Hosts[0].Automation.Scopes) != 1 || exact.Managed.Hosts[0].Automation.ScopesTruncated || len(exact.Managed.Hosts[0].Automation.Scopes["main"].Checks) != 1 {
		t.Fatal("scope read omitted authoritative selection")
	}
	if exact.Automation != nil || exact.Workflow != nil || len(exact.Actions) != 0 {
		t.Fatal("scope read loaded unrelated workflow data or issued actions")
	}
	for _, query := range []string{"host_id=" + m.host.ID, "host_id=" + m.host.ID + "&workspace=main&workspace=other", "host_id=" + m.host.ID + "&workspace=main&approval=true"} {
		decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation/host-scope?"+query, nil, m.cookie), 400)
	}
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation/host-scope?host_id="+m.host.ID+"&workspace=unknown", nil, m.cookie), 404)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/host-scope", url.Values{"csrf": {exact.CSRF}}, m.cookie), 404)
	after := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation", nil, m.cookie), 200).Page
	if len(after.Managed.Runs) != 0 || len(after.Automation.Workflows) != 0 || after.Managed.Hosts[0].Version != m.host.Version {
		t.Fatal("scope read changed workflow or host")
	}
}
func TestAutomationHostScopeBrowserViewerCannotAdoptControl(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	m.f.s.options.ProjectInfo = func(_ context.Context) (*core.ProjectView, error) { return &core.ProjectView{CanWrite: false}, nil }
	path := "/web/automation/host-scope?host_id=" + m.host.ID + "&workspace=main"
	out := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", path, nil, m.cookie), 200).Page
	if out.Managed == nil || out.Managed.Hosts[0].CanControl {
		t.Fatal("viewer received controller projection")
	}
}
