package transport

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/url"
)

func addProfileResources(server *mcp.Server, service Service) {
	for _, kind := range []string{"profiles", "profile"} {
		template := "accord://profiles{?session_id,state}"
		if kind == "profile" {
			template = "accord://profile{?session_id,profile_id,version}"
		}
		server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord " + kind, MIMEType: "application/json", URITemplate: template}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			p, ok := ctx.Value(principalKey{}).(core.Principal)
			if !ok {
				return nil, core.Fail("unauthorized", "authentication required")
			}
			u, err := url.Parse(req.Params.URI)
			if err != nil || u.Scheme != "accord" || u.Host != kind || u.Path != "" || u.User != nil || u.Fragment != "" {
				return nil, core.Fail("validation", "invalid profile URI")
			}
			q, err := url.ParseQuery(u.RawQuery)
			if err != nil || len(q["session_id"]) != 1 || q.Get("session_id") == "" {
				return nil, core.Fail("NO_SESSION", "board session required")
			}
			a := core.Actor{Principal: p, SessionID: q.Get("session_id")}
			q.Del("session_id")
			body, err := readQuery(kind, q.Encode())
			if err != nil {
				return nil, err
			}
			var result any
			if kind == "profiles" {
				var in core.ProfilesInput
				if err = decode(body, &in); err == nil {
					result, err = service.Profiles(ctx, a, in)
				}
			} else {
				var in core.ProfileInput
				if err = decode(body, &in); err == nil {
					result, err = service.Profile(ctx, a, in)
				}
			}
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(result)
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
		})
	}
}
