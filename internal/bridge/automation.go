package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (b *bridge) allowedTool(name string) bool {
	if b.managedRunID != "" && (name == "workflow_submit" || name == "workflow_command") {
		return false
	}
	if b.executionCapabilityID == "" {
		return true
	}
	switch name {
	case "describe", "heartbeat", "next", "get", "update", "transition", "workflow_plan", "workflow_question":
		return true
	}
	return false
}
func (b *bridge) addCheckTool() {
	if b.automation == nil {
		return
	}
	b.server.AddTool(&mcp.Tool{Name: "workflow_run_check", Description: "Start one owner-approved verification preset for your live leased workflow step. Retry with the same request_id and preset_id; poll accord://workflow-check, or repeat this exact request_id and preset_id, for actual completion. Never treat queued/running as passed.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"request_id": map[string]any{"type": "string", "minLength": 1, "maxLength": 128}, "preset_id": map[string]any{"type": "string", "minLength": 1, "maxLength": 128}}, "required": []string{"request_id", "preset_id"}}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		request, preset, e := checkFields(req.Params.Arguments)
		if e != nil {
			return failure("VALIDATION", "Only request_id and preset_id are accepted."), nil
		}
		value, e := b.checkCall(ctx, "start", request, preset)
		if e != nil {
			return failure("CHECK_UNAVAILABLE", "Check status is uncertain; read status with the same IDs before retrying."), nil
		}
		raw, _ := json.Marshal(value)
		return &mcp.CallToolResult{IsError: value.Error != "", StructuredContent: value, Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}}, nil
	})
	b.server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Workflow verification status", MIMEType: "application/json", URITemplate: "accord://workflow-check{?request_id,preset_id,session_id}"}, b.readCheck)
}
func checkFields(raw []byte) (string, string, error) {
	if transport.ValidateJSONFields(raw) != nil {
		return "", "", errors.New("invalid fields")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 2 {
		return "", "", errors.New("invalid fields")
	}
	for k := range fields {
		if k != "request_id" && k != "preset_id" {
			return "", "", errors.New("unknown field")
		}
	}
	request, preset := stringField(fields, "request_id"), stringField(fields, "preset_id")
	if request == "" || preset == "" || len(request) > 128 || len(preset) > 128 {
		return "", "", errors.New("invalid identifier")
	}
	return request, preset, nil
}
func (b *bridge) checkCall(ctx context.Context, operation, request, preset string) (automationipc.Response, error) {
	b.mu.Lock()
	if b.automation == nil || b.executionCapabilityID == "" || b.quarantined || b.session.SessionID == "" || b.task == nil || !b.live(time.Now()) {
		b.mu.Unlock()
		return automationipc.Response{}, errors.New("no live workflow attempt")
	}
	config := *b.automation
	in := automationipc.Request{Operation: operation, RequestID: request, PresetID: preset, SessionID: b.session.SessionID, TaskKey: b.taskKey, TaskGeneration: b.task.Generation}
	b.mu.Unlock()
	// Local checks may wait for reservation receipts; they must never starve lease heartbeats.
	return automationipc.Call(ctx, config, in)
}
func (b *bridge) readCheck(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, e := url.Parse(req.Params.URI)
	if e != nil || u.Scheme != "accord" || u.Host != "workflow-check" || u.Path != "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("invalid check URI")
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, errors.New("invalid check query")
	}
	for k, v := range q {
		if len(v) != 1 || (k != "request_id" && k != "preset_id" && k != "session_id") {
			return nil, errors.New("invalid check fields")
		}
	}
	b.mu.Lock()
	sid := b.session.SessionID
	b.mu.Unlock()
	if q.Has("session_id") && q.Get("session_id") != sid {
		return nil, errors.New("SESSION_MISMATCH")
	}
	raw, _ := json.Marshal(map[string]string{"request_id": q.Get("request_id"), "preset_id": q.Get("preset_id")})
	request, preset, e := checkFields(raw)
	if e != nil {
		return nil, e
	}
	result, e := b.checkCall(ctx, "status", request, preset)
	if e != nil {
		return nil, errors.New("local check status unavailable")
	}
	raw, e = json.Marshal(result)
	if e != nil {
		return nil, e
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
}
