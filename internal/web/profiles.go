package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func profileRoute(path string) (string, string, bool) {
	if path == "/web/profiles" {
		return "", "profile:create", true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/web/profiles/"), "/")
	if len(parts) == 1 && parts[0] != "" {
		return parts[0], "", true
	}
	if len(parts) == 2 && (parts[1] == "update" || parts[1] == "archive") {
		return parts[0], "profile:" + parts[1], true
	}
	return "", "", false
}
func (s *Server) profilePage(r *http.Request, b *browserSession, id string) (Page, error) {
	p, err := s.page(r, b)
	if err != nil {
		return p, err
	}
	p.View = "profiles"
	p.Title = "Профили агентов"
	p.CanManageProfiles = !p.CanContinue
	if s.options.ProjectInfo != nil {
		project, e := s.options.ProjectInfo(r.Context())
		if e != nil {
			return p, e
		}
		p.Project = project
		p.CanManageProfiles = p.CanManageProfiles && project.CanWrite
	}
	if p.CanContinue {
		return p, nil
	}
	q, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		return p, core.Fail("validation", "Неверные параметры URL")
	}
	for key, values := range q {
		if len(values) != 1 || id == "" && key != "state" || id != "" && key != "version" {
			return p, core.Fail("validation", "Неизвестный или повторный параметр")
		}
	}
	if id == "" {
		list, e := s.board.Profiles(r.Context(), b.actor(), core.ProfilesInput{State: q.Get("state")})
		p.Profiles = list.Profiles
		err = e
	} else {
		var version int64
		if raw := q.Get("version"); raw != "" {
			version, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || version < 1 {
				return p, core.Fail("validation", "Неверная версия")
			}
		}
		detail, e := s.board.Profile(r.Context(), b.actor(), core.ProfileInput{ProfileID: id, Version: version})
		p.Profile = &detail.Profile
		p.ProfileRevisions = detail.Revisions
		err = e
		p.View = "profile"
		if version > 0 {
			p.CanManageProfiles = false
		}
	}
	if err != nil {
		return p, err
	}
	if p.CanManageProfiles {
		ops := []string{"profile:create"}
		if id != "" {
			ops = []string{"profile:update", "profile:archive"}
		}
		for _, op := range ops {
			p.Actions[op], err = s.issueAction(r.Context(), b, op)
			if err != nil {
				return p, err
			}
		}
	}
	return p, nil
}
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request, b *browserSession) {
	id, op, ok := profileRoute(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		if id != "" && op != "" {
			http.NotFound(w, r)
			return
		}
		p, err := s.profilePage(r, b, id)
		if err != nil {
			s.fail(w, r, b, err)
			return
		}
		s.render(w, 200, p.View, p)
		return
	}
	if op == "" {
		http.NotFound(w, r)
		return
	}
	action := r.PostForm.Get("action_id")
	if err := s.checkAction(r.Context(), b, action, op); err != nil {
		s.profileError(w, r, b, id, op, err)
		return
	}
	input, err := profileForm(r, id, op, action)
	if err != nil {
		s.profileError(w, r, b, id, op, err)
		return
	}
	raw, err := json.Marshal(input)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	a, err := s.bindAction(r.Context(), b, action, op, raw)
	if err != nil {
		s.profileError(w, r, b, id, op, err)
		return
	}
	out, err := s.runAction(r.Context(), b, a)
	if err != nil {
		s.profileError(w, r, b, id, op, err)
		return
	}
	http.Redirect(w, r, out.Redirect, 303)
}
func profileForm(r *http.Request, id, op, action string) (any, error) {
	allowed := "csrf action_id"
	if op == "profile:archive" {
		allowed += " version state"
	} else {
		allowed += " name description role instructions provider model"
		if op == "profile:update" {
			allowed += " version"
		}
	}
	if r.URL.RawQuery != "" {
		return nil, core.Fail("validation", "Параметры URL для действия запрещены")
	}
	for key := range r.PostForm {
		if !strings.Contains(" "+allowed+" ", " "+key+" ") {
			return nil, core.Fail("validation", "Неизвестное поле формы")
		}
	}
	var version int64
	if op != "profile:create" {
		var err error
		version, err = strconv.ParseInt(r.PostForm.Get("version"), 10, 64)
		if err != nil || version < 1 {
			return nil, core.Fail("validation", "Неверная версия профиля")
		}
	}
	if op == "profile:archive" {
		return core.ProfileArchiveInput{RequestID: action, ProfileID: id, Version: version, State: r.PostForm.Get("state")}, nil
	}
	for _, key := range []string{"name", "description", "role", "instructions", "provider", "model"} {
		if _, ok := r.PostForm[key]; !ok {
			return nil, core.Fail("validation", "Нужны все поля профиля")
		}
	}
	v := core.ProfileSettings{Name: r.PostForm.Get("name"), Description: r.PostForm.Get("description"), Role: r.PostForm.Get("role"), Instructions: r.PostForm.Get("instructions"), Provider: r.PostForm.Get("provider"), Model: r.PostForm.Get("model")}
	if op == "profile:create" {
		return core.ProfileCreateInput{RequestID: action, ProfileSettings: v}, nil
	}
	return core.ProfileUpdateInput{RequestID: action, ProfileID: id, Version: version, ProfileSettings: v}, nil
}
func (s *Server) profileError(w http.ResponseWriter, r *http.Request, b *browserSession, id, op string, err error) {
	p, e := s.profilePage(r, b, id)
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Error = publicError(err)
	p.Draft = draft(r)
	p.ActionID = r.PostForm.Get("action_id")
	p.FieldErrors = map[string]string{}
	if errorStatus(err) == 400 {
		values := r.PostForm
		if strings.TrimSpace(values.Get("name")) == "" || utf8.RuneCountInString(values.Get("name")) > 120 {
			p.FieldErrors["name"] = "Введите имя до120символов"
		}
		if len(values.Get("description")) > 4096 {
			p.FieldErrors["description"] = "До4КиБ"
		}
		if len(values.Get("instructions")) > 16384 {
			p.FieldErrors["instructions"] = "До16КиБ"
		}
		if !core.ValidRole(values.Get("role")) || values.Get("role") == "human" {
			p.FieldErrors["role"] = "Выберите роль агента"
		}
	}
	if errorStatus(err) >= 500 {
		p.Actions[op] = p.ActionID
	}
	browserErrorCode(w, err)
	s.render(w, errorStatus(err), p.View, p)
}
func (s *Server) runProfileAction(ctx context.Context, a core.Actor, action *webAction) (string, error) {
	var profile core.AgentProfile
	var err error
	switch action.Operation {
	case "profile:create":
		var in core.ProfileCreateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			profile, err = s.board.ProfileCreate(ctx, a, in)
		}
	case "profile:update":
		var in core.ProfileUpdateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			profile, err = s.board.ProfileUpdate(ctx, a, in)
		}
	case "profile:archive":
		var in core.ProfileArchiveInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			profile, err = s.board.ProfileArchive(ctx, a, in)
		}
	}
	return "/web/profiles/" + profile.ID, err
}
