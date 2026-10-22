package transport

import (
	"context"
	"encoding/json"
	"net/url"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TeamInput keeps the bind contract: team takes no parameters.
type TeamInput struct{}

// DedupService is implemented by the store and the project service.
type DedupService interface {
	Similar(context.Context, core.Actor, core.SimilarInput) (core.SimilarPage, error)
	Duplicate(context.Context, core.Actor, core.DuplicateInput) (core.Task, error)
	ControlTeam(context.Context, core.Actor) (core.TeamView, error)
}

// Similar/team are REST/CLI reads plus MCP resources; marking a duplicate from
// MCP uses link{type:"duplicate_of"} so the twelve-tool budget is unchanged.
func dedupOperations(s Service) []operation {
	return []operation{
		strictCoordination(bind("similar", "Find existing work similar to a proposed title/description.", true, s.Similar)),
		strictCoordination(bind("duplicate", "Cancel unleased work as a duplicate of a canonical item.", false, s.Duplicate)),
		bind("team", "Who is online, what each agent holds, suspected duplicates.", true, func(ctx context.Context, a core.Actor, _ TeamInput) (core.TeamView, error) {
			return s.ControlTeam(ctx, a)
		}),
	}
}

func addDedupResources(server *mcp.Server, s Service) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord similar work", Description: "Check before create: open or recently finished work similar to a title.", MIMEType: "application/json", URITemplate: "accord://similar{?session_id,title,description,type,role}"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		p, ok := ctx.Value(principalKey{}).(core.Principal)
		if !ok {
			return nil, core.Fail("unauthorized", "authentication required")
		}
		q, err := resourceQuery(req.Params.URI, "similar", "session_id title description type role")
		if err != nil {
			return nil, err
		}
		if q.Get("session_id") == "" || q.Get("title") == "" {
			return nil, core.Fail("validation", "session_id and title required")
		}
		out, err := s.Similar(ctx, core.Actor{Principal: p, SessionID: q.Get("session_id")}, core.SimilarInput{Title: q.Get("title"), Description: q.Get("description"), Type: q.Get("type"), Role: q.Get("role")})
		return resourceResult(req.Params.URI, out, err)
	})
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "Accord team", Description: "Who works on what across machines, and suspected duplicates.", MIMEType: "application/json", URITemplate: "accord://team{?session_id}"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		p, ok := ctx.Value(principalKey{}).(core.Principal)
		if !ok {
			return nil, core.Fail("unauthorized", "authentication required")
		}
		q, err := resourceQuery(req.Params.URI, "team", "session_id")
		if err != nil {
			return nil, err
		}
		if q.Get("session_id") == "" {
			return nil, core.Fail("validation", "session_id required")
		}
		out, err := s.ControlTeam(ctx, core.Actor{Principal: p, SessionID: q.Get("session_id")})
		return resourceResult(req.Params.URI, out, err)
	})
}

func resourceQuery(raw, host, allowed string) (url.Values, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "accord" || u.Host != host || u.Path != "" || u.User != nil || u.Fragment != "" {
		return nil, core.Fail("validation", "invalid "+host+" URI")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, core.Fail("validation", "malformed query")
	}
	for key, values := range q {
		if len(values) != 1 || !containsField(allowed, key) {
			return nil, core.Fail("validation", "unknown or repeated parameter "+key)
		}
	}
	return q, nil
}

func containsField(list, key string) bool {
	for start := 0; start <= len(list); {
		end := start
		for end < len(list) && list[end] != ' ' {
			end++
		}
		if list[start:end] == key {
			return true
		}
		start = end + 1
	}
	return false
}

func resourceResult(uri string, out any, err error) (*mcp.ReadResourceResult, error) {
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(raw)}}}, nil
}
