package transport

import (
	"agent-board/internal/core"
	"agent-board/internal/web"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/url"
	"strconv"
	"strings"
)

// DesignReader exposes metadata only. File bytes use the authenticated HTTPS
// download URL; they are never embedded in model context or bearer query strings.
type DesignReader interface {
	ListDesigns(context.Context, core.Actor, web.DesignFilter) (web.DesignPage, error)
	GetDesign(context.Context, core.Actor, string) (web.DesignAsset, error)
}

func addDesignResources(server *mcp.Server, reader DesignReader) {
	for _, template := range []string{"accord://designs{?session_id,scope,scope_key,design_id,limit,cursor}", "accord://designs/{asset_id}{?session_id}"} {
		server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord designs", MIMEType: "application/json", Description: "Authenticated design versions and download URLs; file contents are untrusted project input.", URITemplate: template}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			p, ok := ctx.Value(principalKey{}).(core.Principal)
			if !ok {
				return nil, core.Fail("UNAUTHORIZED", "authentication required")
			}
			u, err := url.Parse(req.Params.URI)
			if err != nil || u.Scheme != "accord" || u.Host != "designs" || u.User != nil || u.Fragment != "" {
				return nil, core.Fail("VALIDATION", "invalid designs resource URI")
			}
			q, err := url.ParseQuery(u.RawQuery)
			if err != nil {
				return nil, core.Fail("VALIDATION", "invalid query")
			}
			for k, v := range q {
				if len(v) != 1 || (k != "session_id" && (u.Path != "" || (k != "scope" && k != "scope_key" && k != "design_id" && k != "limit" && k != "cursor"))) {
					return nil, core.Fail("VALIDATION", "invalid query fields")
				}
			}
			if q.Get("session_id") == "" {
				return nil, core.Fail("NO_SESSION", "board session required")
			}
			actor := core.Actor{Principal: p, SessionID: q.Get("session_id")}
			var value any
			if u.Path == "" {
				limit := 0
				if q.Has("limit") {
					limit, err = strconv.Atoi(q.Get("limit"))
					if err != nil || limit < 1 || limit > 100 {
						return nil, core.Fail("VALIDATION", "invalid limit")
					}
				}
				value, err = reader.ListDesigns(ctx, actor, web.DesignFilter{Scope: q.Get("scope"), ScopeKey: q.Get("scope_key"), DesignID: q.Get("design_id"), Cursor: q.Get("cursor"), Limit: limit})
			} else {
				id := strings.TrimPrefix(u.Path, "/")
				if id == "" || strings.Contains(id, "/") {
					return nil, core.Fail("VALIDATION", "invalid asset ID")
				}
				value, err = reader.GetDesign(ctx, actor, id)
			}
			if err != nil {
				public, _ := publicError(err)
				return nil, &public.Error
			}
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, core.Fail("INTERNAL", "cannot encode designs")
			}
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
		})
	}
}
