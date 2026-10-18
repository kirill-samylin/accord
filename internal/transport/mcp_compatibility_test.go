package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCP20251125InitializeListAndCall(t *testing.T) {
	f := newFixture(t)
	const legacy = "2025-11-25"
	rpc := func(body string, initialized bool) (int, []byte) {
		t.Helper()
		req, err := http.NewRequest("POST", f.server.URL+"/mcp", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+f.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if initialized {
			req.Header.Set("MCP-Protocol-Version", legacy)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, data
	}
	type reply struct {
		Error  json.RawMessage `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	call := func(body string, initialized bool) json.RawMessage {
		t.Helper()
		status, data := rpc(body, initialized)
		var result reply
		if status != http.StatusOK || json.Unmarshal(data, &result) != nil || len(result.Error) != 0 || len(result.Result) == 0 {
			t.Fatalf("legacy MCP failed: HTTP %d %s", status, data)
		}
		return result.Result
	}
	result := call(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"legacy-integration","version":"1"}}}`, false)
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(result, &init); err != nil || init.ProtocolVersion != legacy {
		t.Fatalf("legacy version not negotiated: %s", result)
	}
	status, data := rpc(`{"jsonrpc":"2.0","method":"notifications/initialized"}`, true)
	if status != http.StatusAccepted {
		t.Fatalf("initialized notification: HTTP %d %s", status, data)
	}
	result = call(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, true)
	var listed struct {
		Tools []struct{ Name string } `json:"tools"`
	}
	if err := json.Unmarshal(result, &listed); err != nil || len(listed.Tools) != 12 {
		t.Fatalf("legacy tools/list: %s", result)
	}
	result = call(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"describe","arguments":{"request_id":"legacy-open","role":"coordinator","run_id":"legacy-run","machine":"test-machine"}}}`, true)
	var described struct {
		IsError           bool             `json:"isError"`
		StructuredContent core.SessionView `json:"structuredContent"`
	}
	if err := json.Unmarshal(result, &described); err != nil || described.IsError || described.StructuredContent.SessionID == "" {
		t.Fatalf("legacy tools/call: %s", result)
	}
	// The board session opened over the legacy MCP contract is real shared state.
	body, _ := json.Marshal(map[string]string{"request_id": "legacy-epic", "type": "epic", "title": "Legacy MCP"})
	status, data = request(t, f, "POST", "/api/v1/tasks", described.StructuredContent.SessionID, string(body))
	if got := parseResult[core.Task](t, status, data); got.Type != "epic" || got.Title != "Legacy MCP" {
		t.Fatal("legacy board session did not authorize REST mutation")
	}
}

func TestSDKLegacyProtocolTools(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "legacy-sdk-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: f.server.URL + "/mcp", HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true,
	}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.InitializeResult().ProtocolVersion != "2025-11-25" {
		t.Fatal("SDK did not negotiate legacy protocol")
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != 12 {
		t.Fatalf("legacy SDK list: %v", err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe", Arguments: core.OpenSession{RequestID: "legacy-sdk-open", Role: "coordinator", RunID: "legacy-sdk", Machine: "test-machine"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := toolValue[core.SessionView](t, result); got.SessionID == "" {
		t.Fatal("legacy SDK call did not open board session")
	}
}
