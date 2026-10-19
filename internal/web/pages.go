package web

import (
	"agent-board/internal/core"
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) page(r *http.Request, b *browserSession) (Page, error) {
	p := Page{Title: "Accord", Principal: b.Principal, Session: b.Session, CSRF: b.CSRF, CanContinue: !b.Session.ExpiresAt.After(time.Now()), Draft: map[string]string{}, Actions: map[string]string{}, Roles: []string{"coordinator", "analyst", "architect", "developer", "reviewer", "tester", "human"}, States: []string{"ready", "in_progress", "done", "cancelled"}, Complexities: []string{"low", "normal", "high"}, View: r.URL.Query().Get("view"), EpicKey: r.URL.Query().Get("epic_key"), Role: r.URL.Query().Get("role"), State: r.URL.Query().Get("state")}
	p.Draft["epic_cursor"] = r.URL.Query().Get("epic_cursor")
	for _, op := range []string{"logout", "continue", "create", "upload"} {
		id, err := s.issueAction(r.Context(), b, op)
		if err != nil {
			return p, err
		}
		p.Actions[op] = id
	}
	return p, nil
}
func (s *Server) taskActions(ctx context.Context, b *browserSession, p *Page, tasks []core.Task) error {
	for _, t := range tasks {
		if t.Type != "task" || t.Role != "human" {
			continue
		}
		for _, op := range []string{"claim", "answer", "release", "heartbeat"} {
			id, err := s.issueAction(ctx, b, op)
			if err != nil {
				return err
			}
			p.Actions[op+":"+t.Key] = id
			if p.Task != nil && p.Task.Key == t.Key {
				p.Actions[op] = id
			}
		}
	}
	return nil
}
func canEdit(b *browserSession, t *core.Task) bool {
	return t != nil && t.Role == "human" && t.Type == "task" && t.State == "in_progress" && t.Lease != nil && t.Lease.SessionID == b.Session.SessionID && t.Lease.ExpiresAt.After(time.Now()) && b.Session.ExpiresAt.After(time.Now())
}
func (s *Server) read(w http.ResponseWriter, r *http.Request, b *browserSession) {
	p, err := s.page(r, b)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	name := ""
	switch {
	case r.URL.Path == "/web/":
		name = "board"
	case r.URL.Path == "/web/history":
		name = "history"
	case strings.HasPrefix(r.URL.Path, "/web/tasks/") && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/web/tasks/"), "/"):
		name = "task"
	default:
		http.NotFound(w, r)
		return
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
	}
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	s.render(w, 200, name, p)
}
func (s *Server) boardPage(r *http.Request, b *browserSession, p *Page) error {
	if p.View == "" {
		p.View = "questions"
	}
	if p.View == "questions" {
		p.Role = "human"
	}
	input := core.ListInput{Cursor: r.URL.Query().Get("cursor"), Limit: 100, State: p.State, Role: p.Role, EpicKey: p.EpicKey, IncludeClosed: p.State == "done" || p.State == "cancelled"}
	var tasks core.TasksPage
	var err error
	if p.View == "questions" {
		tasks, err = s.board.ListQuestions(r.Context(), b.actor(), input)
	} else {
		tasks, err = s.board.ListTasks(r.Context(), b.actor(), input)
	}
	if err != nil {
		return err
	}
	p.Tasks = tasks.Tasks
	p.NextCursor = tasks.NextCursor
	epics, err := s.board.ListEpics(r.Context(), b.actor(), core.ListInput{Cursor: r.URL.Query().Get("epic_cursor"), Limit: 100, IncludeClosed: true})
	if err != nil {
		return err
	}
	p.Epics = epics.Tasks
	p.EpicsNextCursor = epics.NextCursor
	seenOwn := map[string]bool{}
	for _, cached := range b.Session.MyTasks {
		task, e := s.board.Get(r.Context(), b.actor(), core.GetInput{TaskKey: cached.Key})
		if e != nil {
			return e
		}
		if canEdit(b, &task) {
			p.MyQuestions = append(p.MyQuestions, task)
			seenOwn[task.Key] = true
		}
	}
	for _, t := range tasks.Tasks {
		if t.Role != "human" {
			continue
		}
		if canEdit(b, &t) {
			if !seenOwn[t.Key] {
				p.MyQuestions = append(p.MyQuestions, t)
			}
		} else {
			p.Questions = append(p.Questions, t)
		}
	}
	sort.SliceStable(p.Questions, func(i, j int) bool {
		a, z := p.Questions[i], p.Questions[j]
		ar, zr := a.State == "ready" && !a.Blocked && a.Lease == nil, z.State == "ready" && !z.Blocked && z.Lease == nil
		if ar != zr {
			return ar
		}
		if a.Priority != z.Priority {
			return a.Priority < z.Priority
		}
		if !a.ReadyAt.Equal(z.ReadyAt) {
			return a.ReadyAt.Before(z.ReadyAt)
		}
		return a.Key < z.Key
	})
	return s.taskActions(r.Context(), b, p, tasks.Tasks)
}
func (s *Server) historyPage(r *http.Request, b *browserSession, p *Page) error {
	before, err := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	if r.URL.Query().Get("cursor") == "" {
		before = 0
		err = nil
	}
	if err != nil || before < 0 {
		return core.Fail("validation", "Неверная страница истории")
	}
	page, err := s.board.ListDecisions(r.Context(), b.actor(), core.DecisionsInput{BeforeCursor: before, Limit: 50, EpicKey: p.EpicKey})
	if err != nil {
		return err
	}
	p.Decisions = page.Events
	if page.HasMore {
		p.NextCursor = strconv.FormatInt(page.NextCursor, 10)
	}
	// Pre-web tasks expose only the current answer. This bounded legacy section
	// never invents attempt-time text or authors for missing historical events.
	legacy, err := s.board.ListTasks(r.Context(), b.actor(), core.ListInput{Limit: 100, Role: "human", State: "done", EpicKey: p.EpicKey})
	if err != nil {
		return err
	}
	for _, task := range legacy.Tasks {
		if task.Evidence.Answer == "" {
			continue
		}
		events, e := s.board.Events(r.Context(), b.actor(), core.EventsInput{Limit: 1, TaskKey: task.Key, Types: []string{"human_answered"}})
		if e != nil {
			return e
		}
		if len(events.Events) == 0 {
			p.LegacyDecisions = append(p.LegacyDecisions, task)
		}
	}
	return nil
}
