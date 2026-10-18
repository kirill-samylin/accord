package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This is an in-process integration test against real SQLite and the official
// SDK, not a process/E2E or an actual Claude Code/Codex acceptance claim.
func TestSDKLeaseWorkflowAndRevocation(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "workflow-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp", HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	serial := 0
	invoke := func(name, session string, input any) *mcp.CallToolResult {
		t.Helper()
		encoded, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if name != "describe" {
			fields["session_id"], _ = json.Marshal(session)
		}
		if name != "get" && name != "events" {
			serial++
			fields["request_id"], _ = json.Marshal(fmt.Sprintf("workflow-%d", serial))
		}
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: fields})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return result
	}
	open := func(role string) core.SessionView {
		return toolValue[core.SessionView](t, invoke("describe", "", core.OpenSession{Role: role, RunID: "workflow-" + role, Machine: "test-machine"}))
	}
	coordinator := open("coordinator")
	epic := toolValue[core.Task](t, invoke("create", coordinator.SessionID, core.CreateInput{Type: "epic", Title: "Workflow", Description: "Transport integration"}))
	task := toolValue[core.Task](t, invoke("create", coordinator.SessionID, core.CreateInput{Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Develop", Description: "Verify all operations"}))
	other := toolValue[core.Task](t, invoke("create", coordinator.SessionID, core.CreateInput{Type: "task", EpicKey: epic.Key, Role: "architect", Title: "Related task", Description: "Link source"}))
	developer := open("developer")
	claim := toolValue[core.Claim](t, invoke("next", developer.SessionID, core.NextInput{TaskKey: task.Key}))
	if claim.Lease == nil {
		t.Fatal("next returned no lease")
	}
	lease := core.LeaseInput{TaskKey: task.Key, Generation: claim.Lease.Generation}
	title := "Updated"
	toolValue[core.Task](t, invoke("update", developer.SessionID, core.UpdateInput{LeaseInput: lease, Title: &title}))
	toolValue[core.Task](t, invoke("comment", developer.SessionID, core.CommentInput{LeaseInput: lease, Text: "Progress"}))
	toolValue[core.SessionView](t, invoke("heartbeat", developer.SessionID, core.HeartbeatInput{TaskKey: task.Key, Generation: lease.Generation}))
	toolValue[core.Task](t, invoke("link", developer.SessionID, core.LinkInput{From: other.Key, To: task.Key, Type: "related", Generation: lease.Generation}))
	lease.Reason = "Pausing transport test"
	lease.Branch = "codex/test"
	toolValue[core.Task](t, invoke("release", developer.SessionID, lease))
	claim = toolValue[core.Claim](t, invoke("next", developer.SessionID, core.NextInput{TaskKey: task.Key}))
	if claim.Lease == nil || claim.Lease.Generation <= lease.Generation {
		t.Fatal("generation did not advance")
	}
	done := toolValue[core.Task](t, invoke("transition", developer.SessionID, core.TransitionInput{LeaseInput: core.LeaseInput{TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &core.Evidence{Commit: "1111111111111111111111111111111111111111", RunSummary: "in-process integration"}}))
	if done.State != "done" {
		t.Fatal("transition failed")
	}
	page := toolValue[core.EventsPage](t, invoke("events", developer.SessionID, core.EventsInput{Limit: 1}))
	if !page.HasMore || len(page.Events) != 1 {
		t.Fatal("event page missing")
	}
	principal, err := f.store.Authenticate(ctx, f.token)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.RevokeToken(ctx, principal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := clientSession.ListTools(ctx, nil); err == nil {
		t.Fatal("revoked token could list tools")
	}
}
