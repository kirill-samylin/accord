package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client mistakes are definitive answers, not uncertainty about ownership: the
// bridge must report them and keep maintaining its own lease.
func TestClientValidationErrorsNeverQuarantine(t *testing.T) {
	f := setup(t, 0)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": task.Key}))
	for _, raw := range []string{`{"since_cursor":0,"limit":1e309}`, `{"since_cursor":0,"limit":1e20}`, `{"since_cursor":-1}`} {
		result := f.invoke("events", json.RawMessage(raw))
		body, _ := resultJSON(result)
		if !result.IsError || errorCode(body) != "VALIDATION" {
			t.Fatalf("%s: want VALIDATION, got %s", raw, body)
		}
	}
	result := f.invoke("comment", json.RawMessage(`{"request_id":"huge","task_key":"`+task.Key+`","generation":1e400,"text":"x"}`))
	if body, _ := resultJSON(result); !result.IsError || errorCode(body) != "VALIDATION" {
		t.Fatalf("mutation with infinite number: %s", body)
	}
	if f.stderr.String() != "" {
		t.Fatalf("validation error quarantined the bridge: %q", f.stderr.String())
	}
	value[core.Task](t, f.invoke("comment", map[string]any{"request_id": "after", "task_key": task.Key, "generation": claim.Lease.Generation, "text": "still owned"}))
}

// Another session's task, or a stale generation for a task this bridge does
// not track, is not loss of the bridge's own lease.
func TestForeignTaskLeaseErrorsDoNotQuarantine(t *testing.T) {
	f := setup(t, 0)
	own := f.task()
	foreign := f.task()
	unclaimed := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", map[string]string{"request_id": "next", "task_key": own.Key}))
	actor := f.admin()
	other, err := f.store.Describe(context.Background(), actor.Principal, core.OpenSession{RequestID: fmtID(), Role: "developer", RunID: "other-worker", Machine: "test-machine"})
	if err != nil {
		t.Fatal(err)
	}
	held, err := f.store.Next(context.Background(), core.Actor{Principal: actor.Principal, SessionID: other.SessionID}, core.NextInput{RequestID: fmtID(), TaskKey: foreign.Key})
	if err != nil || held.Lease == nil {
		t.Fatal(err)
	}
	calls := []struct {
		name  string
		input map[string]any
		code  string
	}{
		{"comment", map[string]any{"request_id": "c1", "task_key": foreign.Key, "generation": held.Lease.Generation, "text": "x"}, "LEASE_HELD"},
		{"update", map[string]any{"request_id": "u1", "task_key": foreign.Key, "generation": held.Lease.Generation, "title": "x"}, "LEASE_HELD"},
		{"link", map[string]any{"request_id": "l1", "from": own.Key, "to": foreign.Key, "type": "related", "generation": held.Lease.Generation}, "LEASE_HELD"},
		// No lease at all: the server still says LEASE_LOST, and the bridge
		// re-verifies its own lease with a heartbeat instead of quarantining.
		{"update", map[string]any{"request_id": "u2", "task_key": unclaimed.Key, "generation": 7, "title": "x"}, "LEASE_LOST"},
	}
	for _, call := range calls {
		result := f.invoke(call.name, call.input)
		body, _ := resultJSON(result)
		if !result.IsError || errorCode(body) != call.code {
			t.Fatalf("%s %v: want %s, got %s", call.name, call.input["task_key"], call.code, body)
		}
	}
	if f.stderr.String() != "" {
		t.Fatalf("foreign task error quarantined the bridge: %q", f.stderr.String())
	}
	value[core.Task](t, f.invoke("comment", map[string]any{"request_id": "own", "task_key": own.Key, "generation": claim.Lease.Generation, "text": "still owned"}))
}

func TestResourceDomainErrorsKeepTheirCode(t *testing.T) {
	f := setup(t, 0)
	f.describe()
	_, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "accord://coordination?task_key=T-999"})
	var wire *jsonrpc.Error
	if !errors.As(err, &wire) || wire.Code != mcp.CodeResourceNotFound || !strings.Contains(wire.Message, "NOT_FOUND") || !strings.Contains(string(wire.Data), `"code":"NOT_FOUND"`) {
		t.Fatalf("resource error lost its domain code: %#v", err)
	}
	_, err = f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "accord://notes?scope_type=role&scope_id=developer&limit=100000"})
	if !errors.As(err, &wire) || wire.Code != jsonrpc.CodeInvalidParams || !strings.Contains(string(wire.Data), `"code":"VALIDATION"`) {
		t.Fatalf("resource validation error: %#v", err)
	}
	if f.stderr.String() != "" {
		t.Fatal("resource error quarantined the bridge")
	}
}

func TestRejectedBeforeExecutionOnlyForPreHandlerCodes(t *testing.T) {
	for code, want := range map[int64]bool{jsonrpc.CodeParseError: true, jsonrpc.CodeInvalidRequest: true, jsonrpc.CodeMethodNotFound: true, jsonrpc.CodeInvalidParams: true, jsonrpc.CodeInternalError: false, 0: false} {
		if _, got := rejectedBeforeExecution(errors.Join(errors.New("wrapped"), &jsonrpc.Error{Code: code, Message: "m"})); got != want {
			t.Fatalf("code %d: definitive=%v", code, got)
		}
	}
	if _, got := rejectedBeforeExecution(errors.New("connection reset")); got {
		t.Fatal("transport failure treated as definitive")
	}
}
