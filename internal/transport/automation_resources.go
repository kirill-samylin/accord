package transport

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func addAutomationResources(server *mcp.Server, service Service) {
	a, ok := service.(AutomationService)
	if !ok || !a.AutomationAvailable() {
		return
	}
	for _, kind := range []string{"workflows", "workflow", "workflow-report"} {
		server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord " + kind, MIMEType: "application/json", URITemplate: "accord://" + kind + "{?session_id,workflow_id,state,cursor,limit}"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			p, ok := ctx.Value(principalKey{}).(core.Principal)
			if !ok {
				return nil, core.Fail("UNAUTHORIZED", "authentication required")
			}
			u, e := url.Parse(req.Params.URI)
			if e != nil || u.Scheme != "accord" || u.Host != kind || u.Path != "" || u.User != nil || u.Fragment != "" {
				return nil, core.Fail("VALIDATION", "invalid workflow URI")
			}
			q, e := url.ParseQuery(u.RawQuery)
			if e != nil {
				return nil, core.Fail("VALIDATION", "invalid workflow query")
			}
			for k, v := range q {
				if len(v) != 1 {
					return nil, core.Fail("VALIDATION", "repeated query field")
				}
				switch k {
				case "session_id", "workflow_id", "state", "cursor", "limit":
				default:
					return nil, core.Fail("VALIDATION", "unknown query field")
				}
			}
			if q.Get("session_id") == "" {
				return nil, core.Fail("NO_SESSION", "board session required")
			}
			in := core.WorkflowReadInput{WorkflowID: q.Get("workflow_id"), State: q.Get("state"), Cursor: q.Get("cursor")}
			if q.Has("limit") {
				in.Limit, e = strconv.Atoi(q.Get("limit"))
				if e != nil {
					return nil, core.Fail("VALIDATION", "invalid limit")
				}
			}
			actor := core.Actor{Principal: p, SessionID: q.Get("session_id")}
			var value any
			switch kind {
			case "workflows":
				value, e = a.WorkflowList(ctx, actor, in)
			case "workflow":
				value, e = a.WorkflowGet(ctx, actor, in)
			case "workflow-report":
				value, e = a.WorkflowReport(ctx, actor, in)
			}
			if e != nil {
				return nil, e
			}
			raw, e := json.Marshal(value)
			if e != nil {
				return nil, e
			}
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
		})
	}
}
