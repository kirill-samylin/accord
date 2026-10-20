package transport

import (
	"agent-board/internal/core"
	"bytes"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/url"
)

func strictCoordination(op operation) operation {
	run := op.run
	op.run = func(ctx context.Context, p core.Principal, session string, data []byte) (any, error) {
		var fields map[string]json.RawMessage
		if err := decodeObject(data, &fields); err != nil {
			return nil, err
		}
		for _, v := range fields {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return nil, core.Fail("validation", "null fields are not accepted")
			}
		}
		return run(ctx, p, session, data)
	}
	return op
}
func coordinationOperations(s Service) []operation {
	return []operation{
		strictCoordination(bind("coordination", "Explain routing, dependencies and verification.", true, s.Coordination)),
		strictCoordination(bind("assign", "Assign unleased work to a pinned profile or role queue.", false, s.Assign)),
		strictCoordination(bind("check_create", "Create a check against the current completed result atomically.", false, s.CheckCreate)),
	}
}
func addCoordinationResource(server *mcp.Server, s Service) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord coordination", MIMEType: "application/json", URITemplate: "accord://coordination{?session_id,task_key}"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		p, ok := ctx.Value(principalKey{}).(core.Principal)
		if !ok {
			return nil, core.Fail("unauthorized", "authentication required")
		}
		u, err := url.Parse(req.Params.URI)
		if err != nil || u.Scheme != "accord" || u.Host != "coordination" || u.Path != "" || u.User != nil || u.Fragment != "" {
			return nil, core.Fail("validation", "invalid coordination URI")
		}
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil || len(q) != 2 || len(q["session_id"]) != 1 || q.Get("session_id") == "" || len(q["task_key"]) != 1 || q.Get("task_key") == "" {
			return nil, core.Fail("validation", "session_id and task_key required exactly once")
		}
		out, err := s.Coordination(ctx, core.Actor{Principal: p, SessionID: q.Get("session_id")}, core.CoordinationInput{TaskKey: q.Get("task_key")})
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
	})
}
