package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func coordinationRoute(path string) (string, string, bool) {
	if !strings.HasPrefix(path, "/web/tasks/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/web/tasks/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		return "", "", false
	}
	switch parts[1] {
	case "assign", "link", "transition", "checks", "duplicate":
		return parts[0], "coord:" + parts[1], true
	}
	return "", "", false
}
func (s *Server) coordinationPage(r *http.Request, b *browserSession, p *Page) error {
	if p.Task == nil || p.CanContinue {
		return nil
	}
	out, err := s.board.Coordination(r.Context(), b.actor(), core.CoordinationInput{TaskKey: p.Task.Key})
	if err != nil {
		return err
	}
	p.Coordination = &out
	if similar, e := s.board.Similar(r.Context(), b.actor(), core.SimilarInput{Type: p.Task.Type, Role: p.Task.Role, Title: p.Task.Title, Description: p.Task.Description, ExternalID: p.Task.ExternalID, Areas: p.Task.Areas, ExcludeKey: p.Task.Key, Limit: 5}); e == nil {
		p.Similar = similar.Similar
	}
	p.CanCoordinate = b.Session.Role == "human" || b.Session.Role == "coordinator"
	if s.options.ProjectInfo != nil {
		project, e := s.options.ProjectInfo(r.Context())
		if e != nil {
			return e
		}
		p.Project = project
		p.CanCoordinate = p.CanCoordinate && project.CanWrite
	}
	if !p.CanCoordinate {
		return nil
	}
	for _, op := range []string{"coord:assign", "coord:link", "coord:transition", "coord:checks", "coord:duplicate"} {
		p.Actions[op], err = s.issueAction(r.Context(), b, op)
		if err != nil {
			return err
		}
	}
	return nil
}
func coordinationForm(r *http.Request, key, op, id string) (any, error) {
	allowed := " csrf action_id "
	switch op {
	case "coord:assign":
		allowed += "version mode profile_id profile_version reason "
	case "coord:link":
		allowed += "from type condition unlink generation "
	case "coord:transition":
		allowed += "state reason evidence_ref generation "
	case "coord:checks":
		allowed += "target_version role profile_id profile_version "
	case "coord:duplicate":
		allowed += "canonical_key reason version "
	}
	if r.URL.RawQuery != "" {
		return nil, core.Fail("validation", "Параметры URL для действия запрещены")
	}
	for k := range r.PostForm {
		if !strings.Contains(allowed, " "+k+" ") {
			return nil, core.Fail("validation", "Неизвестное поле формы")
		}
	}
	number := func(name string, required bool) (int64, error) {
		v := r.PostForm.Get(name)
		if v == "" && !required {
			return 0, nil
		}
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil || n < 0 || required && n < 1 {
			return 0, core.Fail("validation", "Неверное числовое поле "+name)
		}
		return n, nil
	}
	switch op {
	case "coord:assign":
		v, e := number("version", true)
		if e != nil {
			return nil, e
		}
		pv, e := number("profile_version", false)
		if e != nil {
			return nil, e
		}
		return core.AssignInput{RequestID: id, TaskKey: key, Version: v, Mode: r.PostForm.Get("mode"), ProfileID: r.PostForm.Get("profile_id"), ProfileVersion: pv, Reason: r.PostForm.Get("reason")}, nil
	case "coord:checks":
		v, e := number("target_version", true)
		if e != nil {
			return nil, e
		}
		pv, e := number("profile_version", false)
		if e != nil {
			return nil, e
		}
		return core.CheckCreateInput{RequestID: id, TargetKey: key, TargetVersion: v, Role: r.PostForm.Get("role"), ProfileID: r.PostForm.Get("profile_id"), ProfileVersion: pv}, nil
	case "coord:link":
		g, e := number("generation", false)
		if e != nil {
			return nil, e
		}
		unlink := r.PostForm.Get("unlink")
		if unlink != "" && unlink != "true" && unlink != "false" {
			return nil, core.Fail("validation", "unlink принимает true или false")
		}
		return core.LinkInput{RequestID: id, From: r.PostForm.Get("from"), To: key, Type: r.PostForm.Get("type"), Condition: r.PostForm.Get("condition"), Unlink: unlink == "true", Generation: g}, nil
	case "coord:duplicate":
		v, e := number("version", false)
		if e != nil {
			return nil, e
		}
		return core.DuplicateInput{RequestID: id, TaskKey: key, CanonicalKey: r.PostForm.Get("canonical_key"), Version: v, Reason: r.PostForm.Get("reason")}, nil
	case "coord:transition":
		g, e := number("generation", false)
		if e != nil {
			return nil, e
		}
		state := r.PostForm.Get("state")
		if state != "ready" && state != "cancelled" && state != "open" && state != "done" {
			return nil, core.Fail("validation", "Недопустимое состояние")
		}
		return core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: id, TaskKey: key, Generation: g, Reason: r.PostForm.Get("reason")}, State: state, EvidenceRef: r.PostForm.Get("evidence_ref")}, nil
	}
	return nil, core.Fail("validation", "Неизвестное действие")
}
func (s *Server) handleCoordination(w http.ResponseWriter, r *http.Request, b *browserSession, key, op string) {
	id := r.PostForm.Get("action_id")
	if err := s.checkAction(r.Context(), b, id, op); err != nil {
		s.coordinationError(w, r, b, key, op, err)
		return
	}
	input, err := coordinationForm(r, key, op, id)
	if err != nil {
		s.coordinationError(w, r, b, key, op, err)
		return
	}
	raw, err := json.Marshal(input)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	action, err := s.bindAction(r.Context(), b, id, op, raw)
	if err != nil {
		s.coordinationError(w, r, b, key, op, err)
		return
	}
	out, err := s.runAction(r.Context(), b, action)
	if err != nil {
		s.coordinationError(w, r, b, key, op, err)
		return
	}
	http.Redirect(w, r, out.Redirect, 303)
}
func (s *Server) runCoordinationAction(ctx context.Context, a core.Actor, action *webAction) (string, error) {
	var task core.Task
	var err error
	switch action.Operation {
	case "coord:assign":
		var in core.AssignInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			task, err = s.board.Assign(ctx, a, in)
		}
	case "coord:link":
		var in core.LinkInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			task, err = s.board.Link(ctx, a, in)
		}
	case "coord:transition":
		var in core.TransitionInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			task, err = s.board.Transition(ctx, a, in)
		}
	case "coord:checks":
		var in core.CheckCreateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			task, err = s.board.CheckCreate(ctx, a, in)
		}
	case "coord:duplicate":
		var in core.DuplicateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			task, err = s.board.Duplicate(ctx, a, in)
		}
	}
	return "/web/tasks/" + task.Key, err
}
func (s *Server) coordinationError(w http.ResponseWriter, r *http.Request, b *browserSession, key, op string, err error) {
	p, e := s.page(r, b)
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	task, e := s.board.Get(r.Context(), b.actor(), core.GetInput{TaskKey: key})
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Task = &task
	p.View = "task"
	if e = s.coordinationPage(r, b, &p); e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Error = publicError(err)
	p.Draft = draft(r)
	p.ActionID = r.PostForm.Get("action_id")
	p.FieldErrors = map[string]string{}
	if errorStatus(err) >= 500 {
		p.Actions[op] = p.ActionID
	}
	browserErrorCode(w, err)
	s.render(w, errorStatus(err), "task", p)
}

// teamActions issues per-pair action IDs so a human can resolve suspected
// duplicates from the team screen: mark the newer item or dismiss the pair.
func (s *Server) teamActions(r *http.Request, b *browserSession, p *Page) error {
	p.CanCoordinate = b.Session.Role == "human" || b.Session.Role == "coordinator"
	if s.options.ProjectInfo != nil {
		project, err := s.options.ProjectInfo(r.Context())
		if err != nil {
			return err
		}
		p.Project = project
		p.CanCoordinate = p.CanCoordinate && project.CanWrite
	}
	if !p.CanCoordinate || p.Team == nil {
		return nil
	}
	for _, pair := range p.Team.Duplicates {
		for _, key := range []string{pair.Older.Key, pair.Newer.Key} {
			for _, op := range []string{"coord:duplicate", "coord:link"} {
				if p.Actions[op+":"+key] != "" {
					continue
				}
				id, err := s.issueAction(r.Context(), b, op)
				if err != nil {
					return err
				}
				p.Actions[op+":"+key] = id
			}
		}
	}
	return nil
}
