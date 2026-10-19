package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-board/internal/core"
)

func controlPageName(r *http.Request) (string, string) {
	switch r.URL.Path {
	case "/web/":
		if view := r.URL.Query().Get("view"); view != "" {
			return "board", view
		}
		return "overview", "overview"
	case "/web/overview":
		return "overview", "overview"
	case "/web/tasks":
		return "board", "tasks"
	case "/web/questions":
		return "board", "questions"
	case "/web/agents":
		return "agents", "agents"
	case "/web/activity":
		return "activity", "activity"
	case "/web/projects":
		return "projects", "projects"
	case "/web/settings":
		return "settings", "settings"
	case "/web/stats":
		return "stats", "stats"
	case "/web/history":
		return "history", "history"
	}
	if id, ok := strings.CutPrefix(r.URL.Path, "/web/agents/"); ok && !strings.Contains(id, "/") {
		return "agent", "agent"
	}
	if key, ok := strings.CutPrefix(r.URL.Path, "/web/tasks/"); ok && key != "" && !strings.Contains(key, "/") {
		return "task", "task"
	}
	return "", ""
}
func (s *Server) read(w http.ResponseWriter, r *http.Request, b *browserSession) {
	name, view := controlPageName(r)
	if name == "" {
		http.NotFound(w, r)
		return
	}
	if err := controlQuery(r); err != nil {
		s.fail(w, r, b, err)
		return
	}
	if name == "agent" && !core.ValidControlSessionID(strings.TrimPrefix(r.URL.Path, "/web/agents/")) {
		s.fail(w, r, b, core.Fail("validation", "Неверный ID сессии"))
		return
	}
	p, err := s.page(r, b)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	p.View = view
	if p.Draft["layout"] == "" {
		p.Draft["layout"] = "list"
	}
	if p.CanContinue {
		p.Notice = "Сессия доски истекла. Нажмите «Продолжить сессию»"
		s.render(w, 200, name, p)
		return
	}
	switch name {
	case "board":
		err = s.boardPage(r, b, &p)
	case "task":
		var task core.Task
		task, err = s.board.Get(r.Context(), b.actor(), core.GetInput{TaskKey: strings.TrimPrefix(r.URL.Path, "/web/tasks/")})
		if err == nil {
			p.Task = &task
			p.CanEdit = canEdit(b, &task)
			err = s.taskActions(r.Context(), b, &p, []core.Task{task})
		}
	case "history":
		err = s.historyPage(r, b, &p)
	default:
		err = s.controlPage(r, b, &p)
	}
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	s.render(w, 200, name, p)
}
func (s *Server) controlPage(r *http.Request, b *browserSession, p *Page) error {
	switch p.View {
	case "overview":
		overview, err := s.board.ControlOverview(r.Context(), b.actor())
		if err != nil {
			return err
		}
		p.Overview = &overview
		p.ObservedAt = overview.ObservedAt
		agents, err := s.board.ControlSessions(r.Context(), b.actor(), core.ControlSessionsInput{Kind: "agent", State: "active", Limit: 6})
		if err != nil {
			return err
		}
		p.Agents = agents.Sessions
		events, err := s.board.RecentEvents(r.Context(), b.actor(), core.RecentEventsInput{Limit: 20})
		if err != nil {
			return err
		}
		p.Activity = events.Events
	case "agents":
		return s.agentsPage(r, b, p)
	case "agent":
		id := strings.TrimPrefix(r.URL.Path, "/web/agents/")
		agents, err := s.board.ControlSessions(r.Context(), b.actor(), core.ControlSessionsInput{SessionID: id, IncludeClosed: true, Limit: 1})
		if err != nil {
			return err
		}
		if len(agents.Sessions) == 0 {
			return core.Fail("not_found", "Сессия не найдена")
		}
		p.Agent = &agents.Sessions[0]
		p.ObservedAt = agents.ObservedAt
		return s.activityPage(r, b, p, id)
	case "activity":
		return s.activityPage(r, b, p, "")
	case "stats":
		days := 7
		if raw := r.URL.Query().Get("days"); raw != "" {
			days, _ = strconv.Atoi(raw)
		}
		group := r.URL.Query().Get("group_by")
		if group == "" {
			group = "role"
		}
		p.Draft["days"] = strconv.Itoa(days)
		p.Draft["group_by"] = group
		now := time.Now().UTC()
		stats, err := s.board.Stats(r.Context(), b.actor(), core.StatsInput{From: now.Add(-time.Duration(days) * 24 * time.Hour), To: now, GroupBy: group})
		if err != nil {
			return err
		}
		p.Stats = &stats
		p.ObservedAt = now
	case "projects", "settings":
		// Authorize the current board session even on pages with session-only fields.
		overview, err := s.board.ControlOverview(r.Context(), b.actor())
		if err != nil {
			return err
		}
		p.Overview = &overview
		p.ObservedAt = overview.ObservedAt
	}
	return nil
}
func (s *Server) agentsPage(r *http.Request, b *browserSession, p *Page) error {
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind == "" {
		kind = "agent"
	}
	p.Draft["kind"] = kind
	limit := 100
	if value := q.Get("limit"); value != "" {
		limit, _ = strconv.Atoi(value)
	}
	page, err := s.board.ControlSessions(r.Context(), b.actor(), core.ControlSessionsInput{Cursor: q.Get("cursor"), Limit: limit, Role: p.Role, Owner: q.Get("owner"), State: p.AgentState, Kind: kind, IncludeClosed: q.Get("include_closed") == "true"})
	if err != nil {
		return err
	}
	p.Agents = page.Sessions
	p.NextCursor = page.NextCursor
	p.ObservedAt = page.ObservedAt
	return nil
}
func (s *Server) activityPage(r *http.Request, b *browserSession, p *Page, sessionID string) error {
	before, err := eventBefore(r)
	if err != nil {
		return err
	}
	q := r.URL.Query()
	if sessionID == "" {
		sessionID = q.Get("session_id")
	}
	limit := 20
	if value := q.Get("limit"); value != "" {
		limit, _ = strconv.Atoi(value)
	}
	events, err := s.board.RecentEvents(r.Context(), b.actor(), core.RecentEventsInput{BeforeCursor: before, Limit: limit, SessionID: sessionID, TaskKey: q.Get("task_key"), Type: q.Get("type")})
	if err != nil {
		return err
	}
	p.Activity = events.Events
	p.ObservedAt = events.ObservedAt
	if events.HasMore {
		p.NextCursor = strconv.FormatInt(events.NextCursor, 10)
	}
	return nil
}
