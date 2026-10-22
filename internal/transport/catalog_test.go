package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func catalogTask(t *testing.T, f fixture, session core.SessionView) core.Task {
	t.Helper()
	status, data := request(t, f, "POST", "/api/v1/tasks", session.SessionID,
		`{"request_id":"catalog-epic","type":"epic","title":"Catalog epic","description":"Synthetic"}`)
	epic := parseResult[core.Task](t, status, data)
	body, _ := json.Marshal(core.CreateInput{RequestID: "catalog-task", Type: "task", EpicKey: epic.Key,
		Role: "coordinator", Title: "Catalog task", Description: "Synthetic"})
	status, data = request(t, f, "POST", "/api/v1/tasks", session.SessionID, string(body))
	return parseResult[core.Task](t, status, data)
}

func TestCatalogRESTAndNoteResourceShareAuthenticatedState(t *testing.T) {
	f := newFixture(t)
	board := openSession(t, f, "coordinator", "catalog-open")
	task := catalogTask(t, f, board)
	body, _ := json.Marshal(core.NoteInput{RequestID: "catalog-note", Action: "create", ScopeType: "role",
		ScopeID: "coordinator", Text: "Synthetic durable note", SourceTaskKey: task.Key})
	status, data := request(t, f, "POST", "/api/v1/notes", board.SessionID, string(body))
	note := parseResult[core.Note](t, status, data)
	status, data = request(t, f, "POST", "/api/v1/notes", board.SessionID, string(body))
	if replay := parseResult[core.Note](t, status, data); replay.Key != note.Key {
		t.Fatal("note replay created a second key")
	}
	status, data = request(t, f, "GET", "/api/v1/tasks?role=coordinator&limit=1", board.SessionID, "")
	page := parseResult[core.TasksPage](t, status, data)
	if len(page.Tasks) != 1 || page.Tasks[0].Key != task.Key {
		t.Fatalf("task list: %s", data)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "notes-integration", Version: "1"}, nil)
	client, err := consumer.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp",
		HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	templates, err := client.ListResourceTemplates(ctx, nil)
	if err != nil || len(templates.ResourceTemplates) != 6 {
		t.Fatalf("resource catalog: %v %v", templates, err)
	}
	uri := "accord://notes?" + url.Values{"session_id": {board.SessionID}, "scope_type": {"role"}, "scope_id": {"coordinator"}}.Encode()
	result, err := client.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(result.Contents) != 1 {
		t.Fatalf("read notes: %v %v", result, err)
	}
	var notes core.NotesPage
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &notes); err != nil || len(notes.Notes) != 1 || notes.Notes[0].Key != note.Key {
		t.Fatalf("notes resource differs from REST: %v %v", notes, err)
	}
	for _, badURI := range []string{"accord://notes", "accord://notes?session_id=foreign", uri + "&limit=2&limit=3"} {
		if _, err := client.ReadResource(ctx, &mcp.ReadResourceParams{URI: badURI}); err == nil {
			t.Fatalf("unauthorized/invalid resource accepted: %s", badURI)
		}
	}
	status, data = request(t, f, "POST", "/api/v1/notes", board.SessionID,
		`{"request_id":"catalog-delete","action":"delete","key":"`+note.Key+`","reason":"Synthetic cleanup"}`)
	deleted := parseResult[core.Note](t, status, data)
	if deleted.DeletedAt == nil || deleted.AuthorSession != board.SessionID {
		t.Fatal("deletion lost provenance")
	}
	status, data = request(t, f, "GET", "/api/v1/notes", board.SessionID, "")
	if len(parseResult[core.NotesPage](t, status, data).Notes) != 0 {
		t.Fatal("deleted note visible by default")
	}
}

func TestReadQueriesRejectMalformedAndUnrecognizedInput(t *testing.T) {
	f := newFixture(t)
	board := openSession(t, f, "coordinator", "query-open")
	for _, path := range []string{
		"/api/v1/tasks?limit=0", "/api/v1/tasks?limit=201", "/api/v1/tasks?limit=1&limit=2",
		"/api/v1/tasks?owner=%ZZ", "/api/v1/tasks?include_closed=1", "/api/v1/tasks?unknown=yes",
		"/api/v1/notes?include_closed=true", "/api/v1/notes?scope_id=", "/api/v1/notes?include_deleted=1",
		"/api/v1/stats?from=yesterday", "/api/v1/stats?token=secret", "/api/v1/stats?group_by=role&group_by=epic",
	} {
		status, data := request(t, f, "GET", path, board.SessionID, "")
		if status != http.StatusBadRequest {
			t.Errorf("%s: HTTP %d %s", path, status, data)
		}
	}
	for _, path := range []string{"/api/v1/tasks", "/api/v1/epics", "/api/v1/sessions", "/api/v1/notes", "/api/v1/stats"} {
		status, _ := request(t, f, "GET", path, "", "")
		if status != http.StatusNotFound {
			t.Errorf("missing board session on %s returned %d", path, status)
		}
	}
}
