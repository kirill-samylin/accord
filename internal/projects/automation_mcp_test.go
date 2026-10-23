package projects

import (
	"agent-board/internal/client"
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAutomationMCPIntakeAndResourceShareRESTReceipt(t *testing.T) {
	f := setup(t)
	p := f.create(t, f.who, "mcp intake")
	server := httptest.NewServer(f.handler)
	defer server.Close()
	origin, err := client.Origin(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sdk := mcp.NewClient(&mcp.Implementation{Name: "integration", Version: "1"}, nil)
	connection, err := sdk.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp/p/" + p.ID, HTTPClient: client.HTTP(origin, f.token.Secret, 5*time.Second), DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	tools, err := connection.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, tool := range tools.Tools {
		found[tool.Name] = true
	}
	for _, name := range []string{"workflow_submit", "workflow_command", "workflow_plan", "workflow_question"} {
		if !found[name] {
			t.Fatal("missing tool", name)
		}
	}
	if found["automation_policy_create"] || found["workflow_attest"] {
		t.Fatal("owner tool exposed to model")
	}
	open, err := connection.CallTool(ctx, &mcp.CallToolParams{Name: "describe", Arguments: core.OpenSession{RequestID: "native", Role: "developer", RunID: "client", Machine: "machine"}})
	if err != nil || open.IsError {
		t.Fatal("describe", err, open)
	}
	raw, _ := json.Marshal(open.StructuredContent)
	var sid core.SessionView
	if json.Unmarshal(raw, &sid) != nil {
		t.Fatal("session response")
	}
	input := map[string]any{"session_id": sid.SessionID, "request_id": "same-requirement", "text": "Implement feature"}
	result, err := connection.CallTool(ctx, &mcp.CallToolParams{Name: "workflow_submit", Arguments: input})
	if err != nil || result.IsError {
		t.Fatal("submit", err, result)
	}
	raw, _ = json.Marshal(result.StructuredContent)
	var workflow core.WorkflowPage
	if json.Unmarshal(raw, &workflow) != nil || workflow.Workflow.Grant != nil {
		t.Fatal("unexpected workflow")
	}
	replay := automationREST[core.WorkflowPage](t, f, p.ID, f.token.Secret, sid.SessionID, "automation/submit", core.WorkflowSubmitInput{RequestID: "same-requirement", Text: "Implement feature"})
	if replay.Workflow.ID != workflow.Workflow.ID {
		t.Fatal("REST/MCP duplicated intake")
	}
	resource, err := connection.ReadResource(ctx, &mcp.ReadResourceParams{URI: "accord://workflow?session_id=" + sid.SessionID + "&workflow_id=" + workflow.Workflow.ID})
	if err != nil || len(resource.Contents) != 1 {
		t.Fatal("resource", err)
	}
	var read core.WorkflowPage
	if json.Unmarshal([]byte(resource.Contents[0].Text), &read) != nil || read.Workflow.ID != workflow.Workflow.ID {
		t.Fatal("resource mismatch")
	}
	_, err = connection.ReadResource(ctx, &mcp.ReadResourceParams{URI: "accord://workflow?session_id=" + sid.SessionID + "&workflow_id=" + workflow.Workflow.ID + "&command=run"})
	if err == nil {
		t.Fatal("unknown resource fields accepted")
	}
}
