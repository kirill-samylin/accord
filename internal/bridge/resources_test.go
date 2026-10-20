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

func TestProfileResourcesUseWorkerSessionAndPreserveToolBudget(t *testing.T) {
	f := setup(t, 0)
	session := f.describe()
	principal, err := f.store.Authenticate(context.Background(), f.token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := f.store.ProfileCreate(context.Background(), core.Actor{Principal: principal, SessionID: session.SessionID}, core.ProfileCreateInput{RequestID: "profile", ProfileSettings: core.ProfileSettings{Name: "Worker", Role: "developer", Provider: "local", Instructions: "Saved preference"}})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := f.client.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, template := range templates.ResourceTemplates {
		if template.URITemplate == "accord://profiles{?session_id,state}" || template.URITemplate == "accord://profile{?session_id,profile_id,version}" {
			found++
		}
	}
	if found != 2 {
		t.Fatal("profile resources not mirrored")
	}
	uri := "accord://profile?profile_id=" + profile.ID + "&version=1"
	result, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(result.Contents) != 1 {
		t.Fatal(err)
	}
	var page core.ProfilePage
	if err = json.Unmarshal([]byte(result.Contents[0].Text), &page); err != nil || page.Profile.Instructions != profile.Instructions {
		t.Fatal("snapshot resource", err)
	}
	if _, err = f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri + "&session_id=foreign"}); err == nil {
		t.Fatal("resource escaped bridge session")
	}
	tools, err := f.client.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 12 {
		t.Fatal("expanded tool budget", err)
	}
}
