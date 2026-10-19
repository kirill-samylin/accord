package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/web"
)

func controlFixture() web.Page {
	p := fixture()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	p.Overview = &core.ControlOverview{ObservedAt: now, Project: p.Session.Project, TasksTotal: 432, TasksInProgress: 17, ActiveAgentSessions: 12}
	p.Agents = []core.ControlSession{{SessionID: strings.Repeat("a", 64), Role: "developer", Owner: p.Title, Machine: p.Title, RunID: p.Title, Status: "active", ExpiresAt: now, Task: p.Task}}
	p.Agent = &p.Agents[0]
	p.Activity = []core.Event{{Cursor: 99, Type: "task_done", SessionID: p.Agent.SessionID, TaskKey: p.Task.Key, Reason: p.Title, At: now, Data: &core.EventData{Owner: p.Title, Role: "developer"}}}
	p.Stats = &core.Stats{From: now.Add(-7 * 24 * time.Hour), To: now, IncompleteData: true, IncompleteEvents: 4, GroupBy: "owner", Totals: core.StatsMetrics{TokensSpent: 100, MissingTokens: 2, MissingTokensShare: .5}, Groups: []core.StatsGroup{{Key: p.Title}}}
	p.ObservedAt = now
	return p
}

func TestControlPagesEscapeAllExternalFacts(t *testing.T) {
	for _, name := range []string{"overview", "agents", "agent", "activity", "projects", "settings", "stats"} {
		t.Run(name, func(t *testing.T) {
			p := controlFixture()
			p.View = name
			p.Session.Project = p.Title
			p.Session.Rules = map[string]string{p.Title: p.Title}
			html := rendered(t, name, p)
			if strings.Contains(html, "<script>alert(1)</script>") {
				t.Fatal("unescaped control field")
			}
			if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt;") {
				t.Fatal("external text disappeared")
			}
			if !strings.Contains(html, `/web/static/control.js`) || !strings.Contains(html, `id="navigation"`) {
				t.Fatal("control shell missing")
			}
			if strings.Contains(html, "innerHTML") || strings.Contains(html, "https://") {
				t.Fatal("unexpected executable or remote content")
			}
		})
	}
}

func TestOverviewUsesCompleteCountsAndScopedFreshness(t *testing.T) {
	p := controlFixture()
	p.Tasks = []core.Task{*p.Task}
	html := strings.ReplaceAll(rendered(t, "overview", p), "\n", "")
	for _, text := range []string{`data-overview-value="tasks_total">432`, `data-overview-value="active_agent_sessions">12`, "Счётчики на", "Списки ниже обновляются при открытии страницы", "Выборка активных сессий"} {
		if !strings.Contains(html, text) {
			t.Errorf("missing overview fact %q", text)
		}
	}
	if strings.Contains(html, "процесс работает") {
		t.Fatal("session represented as live process")
	}
}

func TestAgentDetailDoesNotInventTaskOrPresence(t *testing.T) {
	p := controlFixture()
	p.Agent.Task = nil
	p.Agent.LastEventAt = nil
	p.Agent.Status = "credential_invalid"
	html := rendered(t, "agent", p)
	for _, text := range []string{"Доступ недействителен", "Действующего владения задачей нет", "Нет данных", "События сессии", "Задача завершена"} {
		if !strings.Contains(html, text) {
			t.Errorf("missing agent semantic %q", text)
		}
	}
	if strings.Contains(html, "Задача во владении</small>") {
		t.Fatal("invalid credential received task badge")
	}
}

func TestStatsReportsMissingCoverageWithoutInventingCost(t *testing.T) {
	html := rendered(t, "stats", controlFixture())
	for _, text := range []string{"Данные неполные", "Событий без полной информации: 4", "50.0% попыток", "Токены не переводятся в деньги"} {
		if !strings.Contains(html, text) {
			t.Errorf("missing metrics provenance %q", text)
		}
	}
	if strings.Contains(html, "$") || strings.Contains(html, "USD") {
		t.Fatal("invented money")
	}
}

func TestControlPaginationAndLayoutRetainContext(t *testing.T) {
	p := controlFixture()
	p.View = "tasks"
	p.Query = "СМЕШАННЫЙ & текст"
	p.Role = "developer"
	p.State = "ready"
	p.NextCursor = "opaque+cursor"
	p.Draft = map[string]string{"layout": "board", "owner": "owner", "include_closed": "true"}
	u, err := url.Parse(nextURL(p))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"view": "tasks", "q": p.Query, "role": "developer", "state": "ready", "cursor": p.NextCursor, "layout": "board", "owner": "owner", "include_closed": "true"} {
		if got := u.Query().Get(key); got != want {
			t.Errorf("lost %s: %q", key, got)
		}
	}
	u, _ = url.Parse(layoutURL(p, "list"))
	if u.Query().Get("cursor") != "" || u.Query().Get("q") != p.Query || u.Query().Get("layout") != "list" {
		t.Fatal("layout switch must reset cursor and retain search")
	}
	p.View = "agent"
	p.Draft = map[string]string{"type": "task_done"}
	u, _ = url.Parse(nextURL(p))
	if u.Path != agentURL(p.Agent.SessionID) || u.Query().Get("type") != "task_done" || u.Query().Get("cursor") != p.NextCursor {
		t.Fatal("agent pagination lost session scope")
	}
	p.View = "activity"
	p.Draft = map[string]string{"task_key": "T-8", "session_id": p.Agent.SessionID, "type": "task_done", "limit": "20"}
	u, _ = url.Parse(nextURL(p))
	for key, want := range p.Draft {
		if u.Query().Get(key) != want {
			t.Errorf("activity lost %s", key)
		}
	}
}

func TestBoardModesKeepProtectedCreation(t *testing.T) {
	for _, layout := range []string{"list", "board"} {
		p := controlFixture()
		p.View = "tasks"
		p.Tasks = []core.Task{*p.Task}
		p.Draft["layout"] = layout
		p.Query = "needle"
		html := rendered(t, "board", p)
		for _, text := range []string{`name="q" value="needle"`, `name="layout" value="` + layout + `"`, `action="/web/tasks" data-create`, `value="create-id"`, `name="csrf"`} {
			if !strings.Contains(html, text) {
				t.Errorf("%s missing %s", layout, text)
			}
		}
		if strings.Contains(html, "draggable=") {
			t.Fatal("unsupported mutation gesture")
		}
	}
}

func TestControlAssetsHaveExplicitTrustedMediaTypes(t *testing.T) {
	for name, want := range staticAssets {
		r := httptest.NewRecorder()
		Static().ServeHTTP(r, httptest.NewRequest("GET", "/web/static/"+name, nil))
		if r.Code != 200 || r.Body.Len() == 0 || r.Header().Get("Content-Type") != want || r.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("invalid embedded asset %s: %d %s", name, r.Code, r.Header().Get("Content-Type"))
		}
	}
	for _, name := range []string{"InterVariable-Italic.woff2", "control_test.go", "../assets/InterVariable.woff2", "app.css/extra"} {
		r := httptest.NewRecorder()
		Static().ServeHTTP(r, httptest.NewRequest("GET", "/web/static/"+name, nil))
		if r.Code != http.StatusNotFound {
			t.Errorf("untrusted path exposed: %s", name)
		}
	}
}
