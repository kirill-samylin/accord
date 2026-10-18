package bridge

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestConfiguredRoleAndPreciseEvidenceThroughSDKWorker(t *testing.T) {
	f := setup(t, 50*time.Millisecond)
	active, err := f.store.Config(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if err = json.Unmarshal(active.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	role := cfg.Roles["developer"]
	role.EvidenceOnDone = []string{"samples", "summary"}
	role.EvidenceSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"samples":{"type":"integer","minimum":0}}}`)
	cfg.Roles["copy-editor"] = role
	cfg.MCP.ToolsByRole = map[string][]string{}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.ApplyConfig(context.Background(), raw, active.ConfigVersion, "integration-admin"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.SetTokenScopes(context.Background(), core.ScopesInput{ID: f.token.ID, Roles: []string{"copy-editor", "coordinator"}, ExpectedVersion: 1, Actor: "integration-admin"}); err != nil {
		t.Fatal(err)
	}
	view := value[core.SessionView](t, f.invoke("describe", core.OpenSession{RequestID: "custom-role", Role: "copy-editor", RunID: "custom-run", Machine: "test-machine"}))
	if view.Role != "copy-editor" {
		t.Fatal("role identity not preserved")
	}
	actor := f.admin()
	epic, err := f.store.Create(context.Background(), actor, core.CreateInput{RequestID: fmtID(), Type: "epic", Title: "custom epic"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := f.store.Create(context.Background(), actor, core.CreateInput{RequestID: fmtID(), Type: "task", EpicKey: epic.Key, Role: "copy-editor", Title: "custom work", Description: "custom evidence through native bridge"})
	if err != nil {
		t.Fatal(err)
	}
	claim := value[core.Claim](t, f.invoke("next", core.NextInput{RequestID: "custom-claim", TaskKey: task.Key}))
	done := value[core.Task](t, f.invoke("transition", map[string]any{"request_id": "custom-done", "task_key": task.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": map[string]any{"samples": json.Number("9007199254740993"), "summary": "native SDK checked"}}))
	count, present, err := done.Evidence.Integer("samples")
	if err != nil || !present || count != 9007199254740993 || done.State != "done" {
		t.Fatal(done, err)
	}
	invalid := f.invoke("create", map[string]any{"request_id": "unknown-top", "type": "task", "epic_key": epic.Key, "role": "copy-editor", "title": "x", "description": "x", "unsolicited": true})
	if !invalid.IsError {
		t.Fatal("top-level DTO accepted unknown property")
	}
	tools, err := f.client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "transition" {
			schema, err := json.Marshal(tool.InputSchema)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(schema), `"evidence":{"additionalProperties":true,"type":"object"}`) && !strings.Contains(string(schema), `"evidence":{"type":"object","additionalProperties":true}`) {
				var root map[string]any
				if err = json.Unmarshal(schema, &root); err != nil {
					t.Fatal(err)
				}
				props := root["properties"].(map[string]any)
				evidence := props["evidence"].(map[string]any)
				if evidence["type"] != "object" {
					t.Fatal("evidence schema is not object", string(schema))
				}
			}
		}
	}
}
