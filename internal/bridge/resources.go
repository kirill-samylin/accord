package bridge

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (b *bridge) mirrorResources(ctx context.Context) error {
	if b.upstream.InitializeResult().Capabilities.Resources == nil {
		return nil // Older, eleven-tool servers have no resources capability.
	}
	count := 0
	for template, err := range b.upstream.ResourceTemplates(ctx, nil) {
		if err != nil {
			return err
		}
		count++
		if count > 32 {
			return errors.New("too many resource templates")
		}
		if template.URITemplate != "accord://notes{?session_id,scope_type,scope_id,limit,cursor,include_deleted}" {
			continue
		}
		copy := *template
		copy.Description = "Read notes for this worker's board session; session_id is optional and cannot select another worker."
		b.server.AddResourceTemplate(&copy, b.readNotes)
	}
	return nil
}

func (b *bridge) readNotes(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session.SessionID == "" {
		return nil, errors.New("NO_SESSION: call describe first")
	}
	u, err := url.Parse(request.Params.URI)
	if err != nil || u.Scheme != "accord" || u.Host != "notes" || u.Path != "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("invalid notes resource URI")
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, errors.New("invalid notes query")
	}
	if ids, present := query["session_id"]; present && (len(ids) != 1 || ids[0] != b.session.SessionID) {
		return nil, errors.New("SESSION_MISMATCH: resource belongs to this worker only")
	}
	if !b.quarantined {
		if !b.live(time.Now()) {
			b.quarantine()
		} else if time.Since(b.lastHeartbeat) >= b.interval {
			if err := b.heartbeat(ctx); err != nil {
				b.quarantine()
			}
		}
	}
	query.Set("session_id", b.session.SessionID)
	u.RawQuery = query.Encode()
	reading, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := b.upstream.ReadResource(reading, &mcp.ReadResourceParams{URI: u.String()})
	if err != nil {
		return nil, errors.New("cannot read notes; verify session and connectivity")
	}
	// The consumer requested a worker-local URI, without its injected SID.
	for _, content := range result.Contents {
		content.URI = request.Params.URI
	}
	return result, nil
}
