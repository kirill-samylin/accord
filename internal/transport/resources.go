package transport

import (
	"context"
	"encoding/json"
	"net/url"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// A resource read is bound to the bearer principal AND live board session.
// A session ID in a resource URI never grants authentication on its own.
func addNoteResource(server *mcp.Server, service Service) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name: "Accord notes", MIMEType: "application/json",
		Description: "Read one notes page for a live board session; reads do not renew leases.",
		URITemplate: "accord://notes{?session_id,scope_type,scope_id,limit,cursor,include_deleted}",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		principal, ok := ctx.Value(principalKey{}).(core.Principal)
		if !ok {
			return nil, core.Fail("UNAUTHORIZED", "authentication is required")
		}
		u, err := url.Parse(request.Params.URI)
		if err != nil || u.Scheme != "accord" || u.Host != "notes" || u.Path != "" || u.User != nil || u.Fragment != "" {
			return nil, core.Fail("VALIDATION", "invalid notes resource URI")
		}
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil || len(q["session_id"]) != 1 || q.Get("session_id") == "" {
			return nil, core.Fail("NO_SESSION", "board session is required")
		}
		session := q.Get("session_id")
		q.Del("session_id")
		body, err := readQuery("notes", q.Encode())
		if err != nil {
			return nil, err
		}
		var input core.NotesInput
		if err := decode(body, &input); err != nil {
			return nil, err
		}
		page, err := service.ListNotes(ctx, core.Actor{Principal: principal, SessionID: session}, input)
		if err != nil {
			// Never expose storage errors (which may contain private paths) via RPC.
			public, _ := publicError(err)
			return nil, &public.Error
		}
		data, err := json.Marshal(page)
		if err != nil {
			return nil, core.Fail("INTERNAL", "cannot encode notes")
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{
			{URI: request.Params.URI, MIMEType: "application/json", Text: string(data)},
		}}, nil
	})
}
