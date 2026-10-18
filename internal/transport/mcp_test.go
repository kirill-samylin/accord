package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type authenticatedTransport struct{ token string }

func (a authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(clone)
}
func toolValue[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	if result.IsError {
		encoded, _ := json.Marshal(result.Content)
		t.Fatalf("tool error: %s", encoded)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSDKStatelessToolsShareRESTState(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp", HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.InitializeResult().ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocol %s", session.InitializeResult().ProtocolVersion)
	}
	before, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Tools) != 12 {
		t.Fatalf("tool count %d", len(before.Tools))
	}
	for _, tool := range before.Tools {
		data, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		_, has := schema.Properties["session_id"]
		if has != (tool.Name != "describe") {
			t.Fatalf("session schema: %s", tool.Name)
		}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe", Arguments: core.OpenSession{RequestID: "mcp-open", Role: "coordinator", RunID: "sdk", Machine: "test-machine"}})
	if err != nil {
		t.Fatal(err)
	}
	boardSession := toolValue[core.SessionView](t, result)
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "create", Arguments: map[string]string{"session_id": boardSession.SessionID, "request_id": "mcp-epic", "type": "epic", "title": "SDK epic", "description": "shared storage"}})
	if err != nil {
		t.Fatal(err)
	}
	epic := toolValue[core.Task](t, result)
	status, data := request(t, f, "GET", "/api/v1/tasks/"+epic.Key, boardSession.SessionID, "")
	if parseResult[core.Task](t, status, data).Title != "SDK epic" {
		t.Fatal("REST and MCP diverged")
	}
	after, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("describe changed tools/list")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get", Arguments: map[string]string{"session_id": boardSession.SessionID, "task_key": epic.Key, "principal": "forged"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("unknown fields were accepted")
	}
}
