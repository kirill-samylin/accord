package projects

import (
	"agent-board/internal/client"
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// T022-R2-7a: the automation archive is readable, read-only, over the project
// REST namespace and an MCP resource, by project owners only (editors refused).
func TestAutomationArchiveRESTAndMCPAreReadOnlyForProjectManagers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "archive read")
	credential, who, err := f.r.CreateCredential(ctx, "editor", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	owner := f.describe(t, p.ID, f.token.Secret, "archive-owner", "human")
	flows := automationREST[core.AutomationArchivePage](t, f, p.ID, f.token.Secret, owner.SessionID, "automation/archive", core.AutomationArchiveInput{Kind: "workflows"})
	if flows.Kind != "workflows" || flows.Workflows == nil || len(flows.Workflows) != 0 || flows.NextCursor != "" {
		t.Fatalf("empty workflow archive page %+v", flows)
	}
	runs := automationREST[core.AutomationArchivePage](t, f, p.ID, f.token.Secret, owner.SessionID, "automation/archive", core.AutomationArchiveInput{Kind: "managed_runs", Limit: 32})
	if runs.Kind != "managed_runs" || runs.Runs == nil {
		t.Fatalf("empty run archive page %+v", runs)
	}
	path := "/api/v1/p/" + p.ID + "/automation/archive"
	if out := f.native(t, "POST", path, f.token.Secret, owner.SessionID, `{"kind":"tasks"}`); out.Code != 400 {
		t.Fatal("unknown archive kind", out.Code, out.Body.String())
	}
	if out := f.native(t, "POST", path+"?kind=workflows", f.token.Secret, owner.SessionID, `{"kind":"workflows"}`); out.Code != 400 {
		t.Fatal("query accepted", out.Code)
	}
	if out := f.native(t, "GET", path, f.token.Secret, owner.SessionID, ""); out.Code != 405 {
		t.Fatal("archive is POST-only", out.Code)
	}
	editor := f.describe(t, p.ID, credential.Secret, "archive-editor", "human")
	if out := f.native(t, "POST", path, credential.Secret, editor.SessionID, `{"kind":"workflows"}`); out.Code != 403 {
		t.Fatal("editor read the archive", out.Code, out.Body.String())
	}

	server := httptest.NewServer(f.handler)
	defer server.Close()
	origin, err := client.Origin(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	connect := func(secret string) *mcp.ClientSession {
		t.Helper()
		sdk := mcp.NewClient(&mcp.Implementation{Name: "archive", Version: "1"}, nil)
		c, err := sdk.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp/p/" + p.ID, HTTPClient: client.HTTP(origin, secret, 5*time.Second), DisableStandaloneSSE: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ownerMCP := connect(f.token.Secret)
	resource, err := ownerMCP.ReadResource(readCtx, &mcp.ReadResourceParams{URI: "accord://automation-archive?session_id=" + owner.SessionID + "&kind=managed_runs&limit=4"})
	if err != nil || len(resource.Contents) != 1 {
		t.Fatal("archive resource", err)
	}
	var page core.AutomationArchivePage
	if json.Unmarshal([]byte(resource.Contents[0].Text), &page) != nil || page.Kind != "managed_runs" {
		t.Fatal("archive resource page", resource.Contents[0].Text)
	}
	if _, err = ownerMCP.ReadResource(readCtx, &mcp.ReadResourceParams{URI: "accord://automation-archive?session_id=" + owner.SessionID + "&kind=workflows&workflow_id=x"}); err == nil {
		t.Fatal("unknown archive resource field accepted")
	}
	if _, err = connect(credential.Secret).ReadResource(readCtx, &mcp.ReadResourceParams{URI: "accord://automation-archive?session_id=" + editor.SessionID + "&kind=workflows"}); err == nil {
		t.Fatal("editor read the archive resource")
	}
}
