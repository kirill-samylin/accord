package bridge

import (
	"agent-board/internal/automationipc"
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"sync"
	"testing"
	"time"
)

func TestWorkflowToolScopesAndStrictCheckIdentifiers(t *testing.T) {
	regular := &bridge{}
	managed := &bridge{managedRunID: "run"}
	child := &bridge{managedRunID: "run", executionCapabilityID: "cap"}
	if !regular.allowedTool("workflow_submit") || managed.allowedTool("workflow_submit") || managed.allowedTool("workflow_command") {
		t.Fatal("intake scope")
	}
	for _, name := range []string{"workflow_submit", "workflow_command", "create", "note", "events", "release", "link"} {
		if child.allowedTool(name) {
			t.Fatal("child scope", name)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"request_id":"a","preset_id":"b","command":"x"}`, `{"request_id":"a","request_id":"b","preset_id":"b"}`, `{"request_id":"a","preset_id":null}`} {
		if _, _, err := checkFields([]byte(raw)); err == nil {
			t.Fatal("invalid input", raw)
		}
	}
}
func TestWorkflowCheckIPCInjectsOwnershipAndResourceCannotStart(t *testing.T) {
	directory, err := os.MkdirTemp("/tmp", "accord-ipc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	binding := automationipc.Binding{WorkflowID: "wf", StepID: "step", RunID: "run", ControlEpoch: 3, Generation: 4}
	var mu sync.Mutex
	var requests []automationipc.Request
	server, err := automationipc.Listen(directory, binding, func(ctx context.Context, in automationipc.Request) (automationipc.Response, error) {
		mu.Lock()
		requests = append(requests, in)
		mu.Unlock()
		return automationipc.Response{Status: "running", CheckID: "check"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	cfg := server.Config()
	now := time.Now()
	b := &bridge{automation: &cfg, executionCapabilityID: "cap", session: core.SessionView{SessionID: "own"}, task: &core.Lease{SessionID: "own", Generation: 7}, taskKey: "TASK001", sessionDeadline: now.Add(time.Minute), taskDeadline: now.Add(time.Minute)}
	if _, err = b.checkCall(context.Background(), "start", "stable", "unit"); err != nil {
		t.Fatal(err)
	}
	_, err = b.readCheck(context.Background(), &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "accord://workflow-check?request_id=stable&preset_id=unit"}})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := append([]automationipc.Request(nil), requests...)
	mu.Unlock()
	if len(got) != 2 || got[0].Operation != "start" || got[1].Operation != "status" {
		t.Fatal("resource started process", got)
	}
	for _, in := range got {
		if in.SessionID != "own" || in.TaskKey != "TASK001" || in.TaskGeneration != 7 || in.RunID != "run" || in.ControlEpoch != 3 {
			t.Fatal("ownership changed", in)
		}
	}
	b.quarantined = true
	if _, err = b.checkCall(context.Background(), "start", "other", "unit"); err == nil {
		t.Fatal("quarantine bypass")
	}
	raw, _ := json.Marshal(got)
	if len(raw) == 0 {
		t.Fatal("no observations")
	}
}
