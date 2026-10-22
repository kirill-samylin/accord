package bridge

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
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

func TestDedupResourcesMirrorTemplatesAndBindReadsToWorkerSession(t *testing.T) {
	f := setup(t, 0)
	task := f.task()
	title := "Экспорт задач JSONL + CSV"
	task, err := f.store.Update(context.Background(), f.admin(), core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: fmtID(), TaskKey: task.Key}, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := f.client.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"accord://team{?session_id}":                                0,
		"accord://similar{?session_id,title,description,type,role}": 0,
	}
	for _, template := range templates.ResourceTemplates {
		if _, exists := want[template.URITemplate]; exists {
			want[template.URITemplate]++
		}
	}
	for template, count := range want {
		if count != 1 {
			t.Fatalf("dedup resource template %q mirrored %d times", template, count)
		}
	}
	encodedTitle := strings.ReplaceAll(url.QueryEscape(task.Title), "+", "%20")
	uris := []string{"accord://team", "accord://similar?title=" + encodedTitle}
	for _, uri := range uris {
		if _, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri}); err == nil {
			t.Fatal("resource read accepted without describe", uri)
		}
	}
	session := f.describe()
	for _, uri := range uris {
		separator := "?"
		if uri != "accord://team" {
			separator = "&"
		}
		for _, candidate := range []string{uri, uri + separator + "session_id=" + session.SessionID} {
			result, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: candidate})
			if err != nil || len(result.Contents) != 1 || result.Contents[0].URI != candidate {
				t.Fatalf("bound dedup read %q: %v %v", candidate, result, err)
			}
			if uri == "accord://team" {
				var team core.TeamView
				if err := json.Unmarshal([]byte(result.Contents[0].Text), &team); err != nil || team.UnclaimedTotal != 1 || len(team.Unclaimed) != 1 || team.Unclaimed[0].Key != task.Key {
					t.Fatalf("team resource contents: %v %v", team, err)
				}
				bound := false
				for _, member := range team.Members {
					for _, agent := range member.Agents {
						bound = bound || agent.SessionID == session.SessionID
					}
				}
				if !bound {
					t.Fatal("team omitted bound worker participation")
				}
			} else {
				var page core.SimilarPage
				if err := json.Unmarshal([]byte(result.Contents[0].Text), &page); err != nil || len(page.Similar) != 1 || page.Similar[0].Key != task.Key {
					t.Fatalf("similar resource contents: %v %v", page, err)
				}
			}
		}
		for _, query := range []string{"session_id=foreign", "session_id=" + session.SessionID + "&session_id=foreign"} {
			if _, err := f.client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri + separator + query}); err == nil {
				t.Fatal("dedup resource escaped worker session", uri)
			}
		}
	}
	tools, err := f.client.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 12 {
		t.Fatal("dedup resources changed tool budget", err)
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
