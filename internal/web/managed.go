package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type managedBoard interface {
	ManagedPermission(context.Context, core.Actor, core.ManagedPermissionInput) (core.ManagedHost, error)
	ManagedList(context.Context, core.Actor, core.ManagedListInput) (core.ManagedPage, error)
	ManagedRun(context.Context, core.Actor, core.ManagedRunInput) (core.ManagedRunPage, error)
	ManagedHostDisable(context.Context, core.Actor, core.ManagedHostDisableInput) (core.ManagedHost, error)
	ManagedStart(context.Context, core.Actor, core.ManagedStartInput) (core.ManagedResult, error)
	ManagedCommand(context.Context, core.Actor, core.ManagedCommandInput) (core.ManagedResult, error)
}

func (s *Server) checkManagedPermission(ctx context.Context, a core.Actor, action *webAction) error {
	m, ok := s.board.(managedBoard)
	if !ok {
		return core.Fail("not_found", "Управляемый запуск недоступен")
	}
	var in core.ManagedPermissionInput
	switch action.Operation {
	case "managed:start":
		var p core.ManagedStartInput
		if err := json.Unmarshal([]byte(action.Payload), &p); err != nil {
			return err
		}
		in = core.ManagedPermissionInput{HostID: p.HostID, Kind: "start"}
	case "managed:command":
		var p core.ManagedCommandInput
		if err := json.Unmarshal([]byte(action.Payload), &p); err != nil {
			return err
		}
		in = core.ManagedPermissionInput{RunID: p.RunID, Kind: p.Kind}
	case "managed:disable":
		var p core.ManagedHostDisableInput
		if err := json.Unmarshal([]byte(action.Payload), &p); err != nil {
			return err
		}
		in = core.ManagedPermissionInput{HostID: p.HostID, Kind: "disable"}
	default:
		return core.Fail("validation", "Неизвестная операция")
	}
	_, err := m.ManagedPermission(ctx, a, in)
	return err
}

func (s *Server) managedPage(r *http.Request, b *browserSession, id string) (Page, error) {
	p, err := s.page(r, b)
	if err != nil {
		return p, err
	}
	p.View = "managed"
	p.Title = "Управление агентами"
	if p.CanContinue {
		return p, nil
	}
	if r.URL.RawQuery != "" {
		return p, core.Fail("validation", "Параметры URL не поддерживаются")
	}
	m, ok := s.board.(managedBoard)
	if !ok {
		return p, core.Fail("not_found", "Управляемый запуск недоступен")
	}
	data, err := m.ManagedList(r.Context(), b.actor(), core.ManagedListInput{})
	if err != nil {
		return p, err
	}
	p.Managed = &data
	if id != "" {
		detail, e := m.ManagedRun(r.Context(), b.actor(), core.ManagedRunInput{RunID: id})
		if e != nil {
			return p, e
		}
		p.ManagedRun = &detail
		p.View = "managed-run"
	}
	profiles, err := s.board.Profiles(r.Context(), b.actor(), core.ProfilesInput{})
	if err != nil {
		return p, err
	}
	p.Profiles = profiles.Profiles
	canWrite := true
	if s.options.ProjectInfo != nil {
		project, e := s.options.ProjectInfo(r.Context())
		if e != nil {
			return p, e
		}
		p.Project = project
		canWrite = project.CanWrite
	}
	for i := range data.Hosts {
		data.Hosts[i].CanControl = data.Hosts[i].CanControl && canWrite
	}
	p.Managed = &data
	control := false
	for _, h := range data.Hosts {
		control = control || h.CanControl
	}
	if control {
		for _, op := range []string{"managed:start", "managed:command", "managed:disable"} {
			p.Actions[op], err = s.issueAction(r.Context(), b, op)
			if err != nil {
				return p, err
			}
		}
	}
	return p, nil
}
func (s *Server) handleManaged(w http.ResponseWriter, r *http.Request, b *browserSession) {
	id := ""
	if strings.HasPrefix(r.URL.Path, "/web/managed/runs/") {
		id = strings.TrimPrefix(r.URL.Path, "/web/managed/runs/")
		if strings.Contains(id, "/") || id == "" {
			http.NotFound(w, r)
			return
		}
	}
	if r.Method != "POST" {
		if r.URL.Path != "/web/managed" && id == "" {
			http.NotFound(w, r)
			return
		}
		p, err := s.managedPage(r, b, id)
		if err != nil {
			s.fail(w, r, b, err)
			return
		}
		s.render(w, 200, p.View, p)
		return
	}
	op := "managed:" + strings.TrimPrefix(r.URL.Path, "/web/managed/")
	if op != "managed:start" && op != "managed:command" && op != "managed:disable" {
		http.NotFound(w, r)
		return
	}
	action := r.PostForm.Get("action_id")
	if err := s.checkAction(r.Context(), b, action, op); err != nil {
		s.managedError(w, r, b, err)
		return
	}
	in, err := managedForm(r, op, action)
	if err != nil {
		s.managedError(w, r, b, err)
		return
	}
	raw, err := json.Marshal(in)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	a, err := s.bindAction(r.Context(), b, action, op, raw)
	if err != nil {
		s.managedError(w, r, b, err)
		return
	}
	out, err := s.runAction(r.Context(), b, a)
	if err != nil {
		s.managedError(w, r, b, err)
		return
	}
	http.Redirect(w, r, out.Redirect, 303)
}
func managedForm(r *http.Request, op, action string) (any, error) {
	allowed := "csrf action_id "
	switch op {
	case "managed:start":
		allowed += "host_id workspace provider profile_id profile_version prompt task_key"
	case "managed:command":
		allowed += "run_id version kind text"
	case "managed:disable":
		allowed += "host_id version"
	}
	if r.URL.RawQuery != "" {
		return nil, core.Fail("validation", "Параметры URL запрещены")
	}
	for key := range r.PostForm {
		if !strings.Contains(" "+allowed+" ", " "+key+" ") {
			return nil, core.Fail("validation", "Неизвестное поле формы")
		}
	}
	versionField := "version"
	if op == "managed:start" {
		versionField = "profile_version"
	}
	version, err := strconv.ParseInt(r.PostForm.Get(versionField), 10, 64)
	if err != nil || version < 1 {
		return nil, core.Fail("validation", "Нужна положительная ожидаемая версия")
	}
	switch op {
	case "managed:start":
		return core.ManagedStartInput{RequestID: action, HostID: r.PostForm.Get("host_id"), Workspace: r.PostForm.Get("workspace"), Provider: r.PostForm.Get("provider"), ProfileID: r.PostForm.Get("profile_id"), ProfileVersion: version, Prompt: r.PostForm.Get("prompt"), TaskKey: r.PostForm.Get("task_key")}, nil
	case "managed:command":
		return core.ManagedCommandInput{RequestID: action, RunID: r.PostForm.Get("run_id"), Version: version, Kind: r.PostForm.Get("kind"), Text: r.PostForm.Get("text")}, nil
	default:
		return core.ManagedHostDisableInput{RequestID: action, HostID: r.PostForm.Get("host_id"), Version: version}, nil
	}
}
func (s *Server) managedError(w http.ResponseWriter, r *http.Request, b *browserSession, err error) {
	id := ""
	if r.URL.Path == "/web/managed/command" {
		id = r.PostForm.Get("run_id")
	}
	p, e := s.managedPage(r, b, id)
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Error = publicError(err)
	p.Draft = draft(r)
	p.ActionID = r.PostForm.Get("action_id")
	if errorStatus(err) >= 500 {
		p.Actions["managed:"+strings.TrimPrefix(r.URL.Path, "/web/managed/")] = p.ActionID
	}
	browserErrorCode(w, err)
	s.render(w, errorStatus(err), p.View, p)
}
func (s *Server) runManagedAction(ctx context.Context, a core.Actor, action *webAction) (string, error) {
	m, ok := s.board.(managedBoard)
	if !ok {
		return "", core.Fail("not_found", "Управляемый запуск недоступен")
	}
	var out core.ManagedResult
	var err error
	switch action.Operation {
	case "managed:start":
		var in core.ManagedStartInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			out, err = m.ManagedStart(ctx, a, in)
		}
	case "managed:command":
		var in core.ManagedCommandInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			out, err = m.ManagedCommand(ctx, a, in)
		}
	case "managed:disable":
		var in core.ManagedHostDisableInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			_, err = m.ManagedHostDisable(ctx, a, in)
		}
		return "/web/managed", err
	default:
		return "", core.Fail("validation", "Неизвестное managed действие")
	}
	return "/web/managed/runs/" + out.Run.ID, err
}
