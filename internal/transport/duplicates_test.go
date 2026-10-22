package transport

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDuplicateRESTAndMCPResources(t *testing.T) {
	f := newFixture(t)
	board := openSession(t, f, "coordinator", "dup-open")
	status, data := request(t, f, "POST", "/api/v1/tasks", board.SessionID, `{"request_id":"e","type":"epic","title":"Panel"}`)
	epic := parseResult[core.Task](t, status, data)
	create := func(id, title string) core.Task {
		status, data := request(t, f, "POST", "/api/v1/tasks", board.SessionID, `{"request_id":"`+id+`","type":"task","epic_key":"`+epic.Key+`","role":"developer","title":"`+title+`","description":"x"}`)
		return parseResult[core.Task](t, status, data)
	}
	keep := create("a", "Экспорт задач в JSONL")
	dup := create("b", "Выгрузка задач в JSONL")
	if len(dup.Warnings) != 1 || dup.Warnings[0].Similar[0].Key != keep.Key {
		t.Fatalf("create warnings: %+v", dup.Warnings)
	}
	status, data = request(t, f, "POST", "/api/v1/tasks/similar", board.SessionID, `{"title":"экспорт задач"}`)
	if page := parseResult[core.SimilarPage](t, status, data); len(page.Similar) != 2 || page.Mode != "warn" {
		t.Fatalf("similar: %+v", page)
	}
	status, data = request(t, f, "GET", "/api/v1/team", board.SessionID, "")
	if team := parseResult[core.TeamView](t, status, data); team.DuplicatesTotal != 1 {
		t.Fatalf("team: %+v", team)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mcp.NewClient(&mcp.Implementation{Name: "dup", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp",
		HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	// MCP URI templates match %20, not "+", for spaces.
	uri := "accord://similar?session_id=" + board.SessionID + "&title=" + url.PathEscape("Экспорт задач")
	if result, err := client.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri}); err != nil || len(result.Contents) != 1 {
		t.Fatalf("similar resource: %v %v", result, err)
	}
	status, data = request(t, f, "POST", "/api/v1/tasks/"+dup.Key+"/duplicate", board.SessionID, `{"request_id":"d","canonical_key":"`+keep.Key+`"}`)
	if marked := parseResult[core.Task](t, status, data); marked.State != "cancelled" || marked.DuplicateOf != keep.Key {
		t.Fatalf("duplicate: %+v", marked)
	}
	status, _ = request(t, f, "POST", "/api/v1/tasks/"+keep.Key+"/duplicate", board.SessionID, `{"request_id":"e2","canonical_key":"`+dup.Key+`"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("cancelled canonical accepted: %d", status)
	}
}
