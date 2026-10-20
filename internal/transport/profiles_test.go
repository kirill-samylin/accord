package transport

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestProfileRESTStrictFieldsReplayAndMCPResourceBudget(t *testing.T) {
	f := newFixture(t)
	session := openSession(t, f, "human", "profiles-session")
	body := `{"request_id":"profile-create","name":"Developer","description":"","role":"developer","instructions":"Instructions","provider":"unconfigured","model":""}`
	code, raw := request(t, f, "POST", "/api/v1/profile_create", session.SessionID, body)
	profile := parseResult[core.AgentProfile](t, code, raw)
	code, raw = request(t, f, "POST", "/api/v1/profile_create", session.SessionID, body)
	if parseResult[core.AgentProfile](t, code, raw).ID != profile.ID {
		t.Fatal("duplicate create")
	}
	for _, bad := range []string{`{"request_id":"duplicate","name":"First","name":"Second"}`, `{"request_id":"unknown","env":{}}`, `{"request_id":"missing","name":"x","role":"developer","provider":"local"}`} {
		if status, _ := request(t, f, "POST", "/api/v1/profile_create", session.SessionID, bad); status != 400 {
			t.Fatal("invalid profile accepted", status)
		}
	}
	for _, query := range []string{"?state=active&state=all", "?unknown=yes"} {
		if status, _ := request(t, f, "GET", "/api/v1/profiles"+query, session.SessionID, ""); status != 400 {
			t.Fatal("invalid query accepted")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "profile-author-test", Version: "1"}, nil)
	conn, err := consumer.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp", HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tools, err := conn.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 12 {
		t.Fatal("MCP tool budget", err)
	}
	uri := "accord://profile?" + url.Values{"session_id": {session.SessionID}, "profile_id": {profile.ID}, "version": {"1"}}.Encode()
	resource, err := conn.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(resource.Contents) != 1 {
		t.Fatal("profile resource", err)
	}
	var detail core.ProfilePage
	if err = json.Unmarshal([]byte(resource.Contents[0].Text), &detail); err != nil || detail.Profile.ID != profile.ID || detail.Profile.Instructions != "Instructions" {
		t.Fatal("resource differs", err)
	}
	if _, err = conn.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri + "&session_id=foreign"}); err == nil {
		t.Fatal("foreign resource session accepted")
	}
}
