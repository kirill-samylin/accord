package bridge

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func localTool(tool *mcp.Tool) (*mcp.Tool, error) {
	data, err := json.Marshal(tool)
	if err != nil {
		return nil, err
	}
	var copied mcp.Tool
	if err = json.Unmarshal(data, &copied); err != nil {
		return nil, err
	}
	var schema map[string]json.RawMessage
	data, err = json.Marshal(copied.InputSchema)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &schema); err != nil || schema == nil {
		return nil, errors.New("invalid tool schema")
	}
	if copied.Name != "describe" {
		var required []string
		if raw := schema["required"]; len(raw) > 0 {
			if err = json.Unmarshal(raw, &required); err != nil {
				return nil, err
			}
		}
		filtered := make([]string, 0, len(required))
		for _, name := range required {
			if name != "session_id" {
				filtered = append(filtered, name)
			}
		}
		schema["required"], err = json.Marshal(filtered)
		if err != nil {
			return nil, err
		}
	}
	copied.InputSchema = schema
	return &copied, nil
}

func resultJSON(result *mcp.CallToolResult) (json.RawMessage, error) {
	if result == nil {
		return nil, errors.New("missing tool result")
	}
	if result.StructuredContent != nil {
		return json.Marshal(result.StructuredContent)
	}
	if len(result.Content) != 1 {
		return nil, errors.New("missing JSON result")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !json.Valid([]byte(text.Text)) {
		return nil, errors.New("invalid JSON result")
	}
	return json.RawMessage(text.Text), nil
}

func stringField(fields map[string]json.RawMessage, name string) string {
	var value string
	_ = json.Unmarshal(fields[name], &value)
	return value
}

func errorCode(raw json.RawMessage) string {
	var value struct {
		Code  string `json:"code"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	if value.Error.Code != "" {
		return value.Error.Code
	}
	return value.Code
}
