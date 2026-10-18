package bridge

import (
	"context"
	"encoding/json"
	"testing"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNotesResourceUsesOnlyWorkerSession(t *testing.T) {
	f := setup(t, 0)
	task := f.task()
	session := f.describe()
	note := value[core.Note](t, f.invoke("note", map[string]string{"request_id": "resource-note", "action": "create",
		"scope_type": "role", "scope_id": "developer", "text": "Synthetic note", "source_task_key": task.Key}))
	uri := "accord://notes?scope_type=role&scope_id=developer"
	result, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(result.Contents) != 1 || result.Contents[0].URI != uri {
		t.Fatalf("worker resource read: %v %v", result, err)
	}
	var page core.NotesPage
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &page); err != nil || len(page.Notes) != 1 || page.Notes[0].Key != note.Key {
		t.Fatalf("worker resource contents: %v %v", page, err)
	}
	for _, candidate := range []string{uri + "&session_id=foreign", uri + "&session_id=" + session.SessionID + "&session_id=foreign"} {
		if _, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: candidate}); err == nil {
			t.Fatal("resource escaped worker session")
		}
	}
}
