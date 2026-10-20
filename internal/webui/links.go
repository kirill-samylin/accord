package webui

import (
	"net/url"

	"agent-board/internal/web"
)

// pageValues preserve the server's validated query context, never action IDs.
func pageValues(p web.Page) (string, url.Values) {
	values := url.Values{}
	base := "/web/"
	switch p.View {
	case "agent":
		if p.Agent != nil {
			base = agentURL(p.Agent.SessionID)
		}
	case "designs", "history", "agents", "activity", "projects", "settings", "stats":
		base += p.View
	case "tasks", "questions":
		values.Set("view", p.View)
	}
	keys := []string{}
	switch p.View {
	case "tasks", "questions":
		if p.EpicKey != "" {
			values.Set("epic_key", p.EpicKey)
		}
		if p.Role != "" {
			values.Set("role", p.Role)
		}
		if p.State != "" {
			values.Set("state", p.State)
		}
		if p.Query != "" {
			values.Set("q", p.Query)
		}
		keys = []string{"layout", "owner", "include_closed", "epic_cursor"}
	case "designs":
		keys = []string{"scope", "scope_key", "design_id"}
	case "history":
		if p.EpicKey != "" {
			values.Set("epic_key", p.EpicKey)
		}
	case "agents":
		if p.Role != "" {
			values.Set("role", p.Role)
		}
		if p.AgentState != "" {
			values.Set("state", p.AgentState)
		}
		keys = []string{"kind", "owner", "session_id", "include_closed", "limit"}
	case "agent":
		keys = []string{"type"}
	case "activity":
		keys = []string{"task_key", "session_id", "type", "limit"}
	case "stats":
		keys = []string{"days", "group_by"}
	}
	for _, key := range keys {
		if v := p.Draft[key]; v != "" {
			values.Set(key, v)
		}
	}
	return base, values
}
func nextURL(p web.Page) string {
	base, values := pageValues(p)
	values.Set("cursor", p.NextCursor)
	return base + "?" + values.Encode()
}
func refreshURL(p web.Page) string {
	base, values := pageValues(p)
	if p.View == "agent" && p.Agent != nil {
		return agentURL(p.Agent.SessionID)
	}
	if p.View == "task" && p.Task != nil {
		return taskURL(p.Task.Key)
	}
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}
func taskURL(key string) string { return "/web/tasks/" + url.PathEscape(key) }
func layoutURL(p web.Page, layout string) string {
	base, values := pageValues(p)
	values.Set("layout", layout)
	return base + "?" + values.Encode()
}

// Epic pagination is independent of task pagination; task page starts over.
func epicsURL(p web.Page) string {
	base, values := pageValues(p)
	values.Set("epic_cursor", p.EpicsNextCursor)
	return base + "?" + values.Encode()
}

// Quick views preserve the layout, and start a fresh project-wide filter.
// Review is a role, never a fabricated task state.
func taskViews(p web.Page) []navItem {
	layout := p.Draft["layout"]
	items := []navItem{
		{URL: "/web/tasks", Label: "Текущие", Active: p.State == "" && p.Role == "" && p.Draft["include_closed"] != "true"},
		{URL: "/web/tasks?include_closed=true", Label: "Все задачи", Active: p.State == "" && p.Role == "" && p.Draft["include_closed"] == "true"},
		{URL: "/web/tasks?state=in_progress", Label: "В работе", Active: p.State == "in_progress" && p.Role == ""},
		{URL: "/web/tasks?state=ready", Label: "В очереди", Active: p.State == "ready" && p.Role == ""},
		{URL: "/web/tasks?role=reviewer", Label: "Ревью", Active: p.Role == "reviewer" && p.State == "" && p.Draft["include_closed"] != "true"},
	}
	for i := range items {
		if p.Query != "" || p.EpicKey != "" || p.Draft["owner"] != "" {
			items[i].Active = false
		}
		if layout == "board" {
			u, _ := url.Parse(items[i].URL)
			q := u.Query()
			q.Set("layout", layout)
			u.RawQuery = q.Encode()
			items[i].URL = u.String()
		}
	}
	return items
}
