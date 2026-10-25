package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestControlPagesRoutingAndActualDTO(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	question := f.question(t, token)
	cookie := f.login(t, token)
	cases := map[string]string{"/web/": "overview", "/web/overview": "overview", "/web/?view=tasks": "board", "/web/?view=questions": "board", "/web/tasks": "board", "/web/questions": "board", "/web/agents": "agents", "/web/projects": "projects", "/web/settings": "settings", "/web/activity": "activity", "/web/stats": "stats", "/web/tasks/" + question.Key: "task", "/web/history": "history"}
	for path, name := range cases {
		p := f.page(t, cookie, path)
		if f.response.name != name {
			t.Fatalf("%s got %s", path, f.response.name)
		}
		if path == "/web/" && (p.Overview == nil || p.Overview.TasksTotal != 1 || p.Overview.Project != "fixture" || len(p.Activity) == 0) {
			t.Fatalf("overview %+v", p)
		}
		if path == "/web/stats" && (p.Stats == nil || p.Stats.GroupBy != "role" || p.Draft["days"] != "7") {
			t.Fatalf("stats %+v", p)
		}
		if path == "/web/settings" && (p.Session.Project != "fixture" || p.Session.SessionTTLSeconds != 60 || len(p.Session.Rules) == 0) {
			t.Fatalf("settings %+v", p)
		}
	}
	humans := f.page(t, cookie, "/web/agents?kind=human")
	if len(humans.Agents) != 2 {
		t.Fatalf("human sessions %+v", humans.Agents)
	}
	id := humans.Agents[0].SessionID
	detail := f.page(t, cookie, "/web/agents/"+id)
	if detail.Agent == nil || detail.Agent.SessionID != id || detail.Agent.Status != "active" {
		t.Fatalf("detail %+v", detail)
	}
	missing := request(f.s, "GET", "/web/agents/"+strings.Repeat("a", 64), nil, cookie)
	if missing.Code != 404 {
		t.Fatalf("missing session %d", missing.Code)
	}
	stats := f.page(t, cookie, "/web/stats?days=30&group_by=owner")
	if stats.Stats.GroupBy != "owner" || stats.Stats.To.Sub(stats.Stats.From) != 30*24*time.Hour {
		t.Fatal("stats period")
	}
}

func TestControlJSONReadsDoNotCreateActionsOrRenew(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	task := f.question(t, token)
	cookie := f.login(t, token)
	p := f.page(t, cookie, "/web/tasks/"+task.Key)
	if w := f.post(cookie, "/web/tasks/"+task.Key+"/claim", p, "claim", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	b, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := f.board.Get(context.Background(), b.actor(), core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	var actions int
	if err = f.s.db.QueryRow(`SELECT count(*) FROM web_actions`).Scan(&actions); err != nil {
		t.Fatal(err)
	}
	var latest int64
	for i := 0; i < 3; i++ {
		w := request(f.s, "GET", "/web/data/overview", nil, cookie)
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("data %d %s", w.Code, w.Body)
		}
		var overview core.ControlOverview
		if err = json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
			t.Fatal(err)
		}
		if overview.LeasedTasks != 1 || overview.ActiveHumanSessions != 2 {
			t.Fatalf("actual overview %+v", overview)
		}
		if i > 0 && overview.LatestEventCursor != latest {
			t.Fatal("poll emitted event")
		}
		latest = overview.LatestEventCursor
		for _, secret := range []string{token.Secret, token.ID, b.CSRF, b.Hash, "action_id", "principal_id"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("unexpected identity/action data %q", secret)
			}
		}
	}
	var afterActions int
	if err = f.s.db.QueryRow(`SELECT count(*) FROM web_actions`).Scan(&afterActions); err != nil {
		t.Fatal(err)
	}
	after, err := f.board.Get(context.Background(), b.actor(), core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	if afterActions != actions || !after.Lease.ExpiresAt.Equal(claimed.Lease.ExpiresAt) || after.Generation != claimed.Generation {
		t.Fatal("poll changed action/session/lease")
	}
	loaded, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Session.SessionID != b.Session.SessionID || !loaded.Session.ExpiresAt.Equal(b.Session.ExpiresAt) {
		t.Fatal("poll continued session")
	}
	f.now = f.now.Add(2 * time.Minute)
	expired := request(f.s, "GET", "/web/data/overview", nil, cookie)
	if expired.Code != 409 || !strings.Contains(expired.Body.String(), `"code":"LEASE_LOST"`) || !strings.HasPrefix(expired.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("stale board %d %s", expired.Code, expired.Body)
	}
	if err = f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	revoked := request(f.s, "GET", "/web/data/overview", nil, cookie)
	if revoked.Code != 401 || !strings.Contains(revoked.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Fatalf("revoked %d %s", revoked.Code, revoked.Body)
	}
	anonymous := request(f.s, "GET", "/web/data/overview", nil, nil)
	if anonymous.Code != http.StatusUnauthorized || strings.Contains(anonymous.Body.String(), "<html") {
		t.Fatal("anonymous data rendered login")
	}
}

func TestControlUnknownAndInvalidQueriesRetainDraft(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "operator"))
	paths := []string{"/web/data/overview?fake=1", "/web/agents?fake=1", "/web/agents?state=running", "/web/agents?kind=bot", "/web/agents?kind=human&role=analyst", "/web/agents?limit=101", "/web/agents?cursor=bad", "/web/agents/bad", "/web/activity?session_id=bad", "/web/activity?type=fake", "/web/activity?cursor=-1", "/web/activity?limit=0", "/web/stats?days=8", "/web/stats?group_by=model", "/web/tasks?layout=drag", "/web/tasks?state=fake", "/web/tasks?q=" + url.QueryEscape(strings.Repeat("я", 201)), "/web/tasks?q=a&q=b", "/web/questions?q=a", "/web/?view=fake", "/web/projects?url=evil", "/web/settings?token=bad", "/web/tasks?q=%XX"}
	for _, path := range paths {
		w := request(f.s, "GET", path, nil, cookie)
		if w.Code != 400 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body)
		}
	}
	w := request(f.s, "GET", "/web/tasks?owner=retained&layout=bad", nil, cookie)
	if w.Code != 400 || f.response.last().Draft["owner"] != "retained" || f.response.last().Draft["layout"] != "bad" {
		t.Fatal("invalid filters lost")
	}
}

func TestControlWholeProjectTaskSearchAndBoundCursor(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	a := f.native(t, token)
	epic, err := f.board.Create(context.Background(), a, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Search"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 102; i++ {
		title := "General " + strconv.Itoa(i)
		description := "context"
		if i >= 100 {
			title = "СКРЫТЫЙ <script>"
			description = "FindMe"
		}
		_, err = f.board.Create(context.Background(), a, core.CreateInput{RequestID: "task-" + strconv.Itoa(i), Type: "task", EpicKey: epic.Key, Role: "analyst", Title: title, Description: description})
		if err != nil {
			t.Fatal(err)
		}
	}
	cookie := f.login(t, token)
	p := f.page(t, cookie, "/web/tasks?q="+url.QueryEscape("скрытый")+"&layout=board")
	if len(p.Tasks) != 2 || p.Query != "скрытый" || p.Draft["layout"] != "board" {
		t.Fatalf("search %+v", p.Tasks)
	}
	all := f.page(t, cookie, "/web/tasks?q=General")
	if len(all.Tasks) != 100 || all.NextCursor != "" {
		t.Fatal("query count")
	}
	all = f.page(t, cookie, "/web/tasks")
	if all.NextCursor == "" {
		t.Fatal("expected whole-project cursor")
	}
	mismatch := request(f.s, "GET", "/web/tasks?q=changed&cursor="+url.QueryEscape(all.NextCursor), nil, cookie)
	if mismatch.Code != 400 {
		t.Fatal("query cursor not bound")
	}
}
