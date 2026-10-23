package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"agent-board/internal/core"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ProtocolVersion = "2026-07-28"

type principalKey struct{}

func mcpHandler(ops []operation, service Service, externalHostPolicy bool, designs DesignReader) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "agent-board", Version: "0.1.0-pilot"}, &mcp.ServerOptions{SupportedProtocolVersions: []string{ProtocolVersion, "2025-11-25"}})
	for _, op := range ops {
		schema := *op.schema
		schema.Properties = make(map[string]*jsonschema.Schema, len(op.schema.Properties)+1)
		for k, v := range op.schema.Properties {
			schema.Properties[k] = v
		}
		schema.Required = append([]string(nil), op.schema.Required...)
		if op.name != "describe" {
			schema.Properties["session_id"] = &jsonschema.Schema{Type: "string", MinLength: intPointer(1)}
			schema.Required = append(schema.Required, "session_id")
		}
		server.AddTool(&mcp.Tool{Name: op.name, Description: op.description, InputSchema: &schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: op.readOnly}},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				principal, ok := ctx.Value(principalKey{}).(core.Principal)
				if !ok {
					return toolResult(nil, core.Fail("UNAUTHORIZED", "authentication is required"))
				}
				body := req.Params.Arguments
				session := ""
				if op.name != "describe" {
					var fields map[string]json.RawMessage
					if err := decodeObject(body, &fields); err != nil {
						return toolResult(nil, err)
					}
					if err := json.Unmarshal(fields["session_id"], &session); err != nil {
						return toolResult(nil, core.Fail("NO_SESSION", "board session is required"))
					}
					delete(fields, "session_id")
					body, _ = json.Marshal(fields)
				}
				value, err := op.run(ctx, principal, session, body)
				return toolResult(value, err)
			})
	}
	addNoteResource(server, service)
	addProfileResources(server, service)
	addCoordinationResource(server, service)
	addDedupResources(server, service)
	addAutomationResources(server, service)
	if designs != nil {
		addDesignResources(server, designs)
	}
	// The SDK's localhost guard rejects a TLS proxy's advertised Host. Disable
	// it only behind NewHandlerWithOptions' validated exact authority policy.
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: MaxBodyBytes, DisableLocalhostProtection: externalHostPolicy})
	return http.NewCrossOriginProtection().Handler(handler)
}

func decodeObject(data []byte, out *map[string]json.RawMessage) error {
	if err := decode(data, out); err != nil {
		return err
	}
	if *out == nil {
		return core.Fail("VALIDATION", "JSON object is required")
	}
	return nil
}

func toolResult(value any, err error) (*mcp.CallToolResult, error) {
	failed := err != nil
	if failed {
		value, _ = publicError(err)
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return &mcp.CallToolResult{IsError: failed, Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}, StructuredContent: value}, nil
}

func intPointer(value int) *int { return &value }
