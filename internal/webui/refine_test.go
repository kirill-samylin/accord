package webui

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRefinedNavigationUsesRealDomainFilters(t *testing.T) {
	p := controlFixture()
	p.View = "tasks"
	p.Draft["layout"] = "board"
	for _, item := range taskViews(p) {
		u, err := url.Parse(item.URL)
		if err != nil {
			t.Fatal(err)
		}
		if u.Query().Get("layout") != "board" {
			t.Fatal("quick view lost board layout")
		}
		if item.Label == "Все задачи" && u.Query().Get("include_closed") != "true" {
			t.Fatal("all tasks excluded terminal tasks")
		}
		if item.Label == "Ревью" && (u.Query().Get("role") != "reviewer" || u.Query().Get("state") != "") {
			t.Fatal("review invented a state")
		}
	}
	html := rendered(t, "board", p)
	for _, required := range []string{`aria-label="Быстрые представления задач"`, `id="command-palette"`, `action="/web/tasks" method="get" role="search"`, `data-command-open hidden`, `class="epic-group"`} {
		if !strings.Contains(html, required) {
			t.Errorf("missing progressive navigation %s", required)
		}
	}
}

func TestAgentKindCannotOfferConflictingRolesWithoutJavaScript(t *testing.T) {
	p := controlFixture()
	p.Roles = []string{"human", "developer", "reviewer"}
	for _, kind := range []string{"agent", "human", "all"} {
		p.Draft["kind"] = kind
		for _, role := range agentRoles(p) {
			if kind == "agent" && role == "human" || kind == "human" && role != "human" {
				t.Fatalf("kind %s offered %s", kind, role)
			}
		}
		html := rendered(t, "agents", p)
		if !strings.Contains(html, `name="kind" value="`+kind+`"`) || strings.Contains(html, `<select name="kind">`) {
			t.Fatal("kind links must reset incompatible role on a native GET")
		}
	}
}

func TestOverviewDistinctCollectionTimesAndOptionalLiveControls(t *testing.T) {
	p := controlFixture()
	p.AgentsObservedAt = time.Date(2026, 10, 20, 8, 1, 0, 0, time.UTC)
	p.ActivityObservedAt = p.AgentsObservedAt.Add(time.Minute)
	html := rendered(t, "overview", p)
	for _, required := range []string{`href="/web/tasks?include_closed=true"`, `data-dashboard-agents-time datetime="2026-10-20T08:01:00Z"`, `data-dashboard-activity-time datetime="2026-10-20T08:02:00Z"`, `hidden data-dashboard-option`, `type="checkbox" data-dashboard-live`, `data-event-type="task_done">Задача завершена`} {
		if !strings.Contains(html, required) {
			t.Errorf("missing dashboard contract %s", required)
		}
	}
	if strings.Contains(html, `checked`) {
		t.Fatal("live collections must be opt-in")
	}
}

func TestTaskTimelineIsScopedAndDoesNotShowForEpic(t *testing.T) {
	p := controlFixture()
	html := rendered(t, "task", p)
	for _, required := range []string{`id="timeline-title"`, `/web/activity?task_key=T-8`, `Задача завершена`} {
		if !strings.Contains(html, required) {
			t.Errorf("missing scoped timeline %s", required)
		}
	}
	p.Task.Type = "epic"
	if strings.Contains(rendered(t, "task", p), `id="timeline-title"`) {
		t.Fatal("unsupported epic activity query exposed")
	}
}
