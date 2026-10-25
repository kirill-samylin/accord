package web

import (
	"agent-board/internal/core"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) mutate(w http.ResponseWriter, r *http.Request, b *browserSession) {
	op, key := "", ""
	switch r.URL.Path {
	case "/web/logout":
		op = "logout"
	case "/web/session/continue":
		op = "continue"
	case "/web/tasks":
		op = "create"
	case "/web/heartbeat":
		op = "heartbeat"
	default:
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/web/tasks/"), "/")
		if !strings.HasPrefix(r.URL.Path, "/web/tasks/") || len(parts) != 2 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		key = parts[0]
		op = parts[1]
		if op != "claim" && op != "answer" && op != "release" && op != "update" {
			http.NotFound(w, r)
			return
		}
	}
	id := r.PostForm.Get("action_id")
	if err := s.checkAction(r.Context(), b, id, op); err != nil {
		s.mutationError(w, r, b, err)
		return
	}
	if op == "logout" {
		if err := s.deleteBrowser(r.Context(), b.Hash); err != nil {
			s.fail(w, r, b, err)
			return
		}
		if _, err := s.db.ExecContext(r.Context(), `INSERT INTO web_audit(event,principal,at) VALUES('logout',?,?)`, b.Principal.ID, time.Now().UnixNano()); err != nil {
			s.fail(w, r, b, err)
			return
		}
		s.cookie(w, s.cookieName, "", time.Time{})
		http.Redirect(w, r, "/web/login", 303)
		return
	}
	if op == "continue" {
		a, e := s.bindAction(r.Context(), b, id, op, []byte("{}"))
		if e != nil {
			s.fail(w, r, b, e)
			return
		}
		if a.Result != "" || a.SessionID != b.Session.SessionID && b.PendingDescribe == "" {
			http.Redirect(w, r, "/web/", 303)
			return
		}
		if err := s.recoverActions(r.Context(), b); err != nil {
			s.fail(w, r, b, err)
			return
		}
		if err := s.ensureSession(r.Context(), b); err != nil {
			s.fail(w, r, b, err)
			return
		}
		if err := s.completeAction(r.Context(), id, "continued"); err != nil {
			s.fail(w, r, b, err)
			return
		}
		http.Redirect(w, r, "/web/", 303)
		return
	}
	// A bound action always keeps its original session, including exact replay
	// after expiry. Only a new create/claim may explicitly open a fresh session.
	var existing string
	if err := s.db.QueryRowContext(r.Context(), `SELECT payload FROM web_actions WHERE id=?`, id).Scan(&existing); err != nil {
		s.fail(w, r, b, err)
		return
	}
	if existing == "" && (op == "create" || op == "claim") {
		if err := s.recoverActions(r.Context(), b); err != nil {
			s.fail(w, r, b, err)
			return
		}
		if err := s.ensureSession(r.Context(), b); err != nil {
			s.fail(w, r, b, err)
			return
		}
	}
	var input any
	switch op {
	case "update":
		var err error
		input, err = browserUpdateInput(r, id, key)
		if err != nil {
			s.mutationError(w, r, b, err)
			return
		}
	case "create":
		priority, err := formPriority(r.PostForm.Get("priority"))
		if err == nil {
			err = formTextError(r.PostForm, []string{"title"}, []string{"description", "force_reason"})
		}
		if err != nil {
			s.mutationError(w, r, b, err)
			return
		}
		areas := []string{}
		for _, v := range strings.Split(r.PostForm.Get("areas"), ",") {
			if v = strings.TrimSpace(v); v != "" {
				areas = append(areas, v)
			}
		}
		profileVersion := int64(0)
		if raw := r.PostForm.Get("profile_version"); raw != "" {
			var e error
			profileVersion, e = strconv.ParseInt(raw, 10, 64)
			if e != nil || profileVersion < 1 {
				s.mutationError(w, r, b, core.Fail("validation", "Неверная версия профиля"))
				return
			}
		}
		input = core.CreateInput{ProfileID: r.PostForm.Get("profile_id"), ProfileVersion: profileVersion, RequestID: id, Type: r.PostForm.Get("type"), EpicKey: r.PostForm.Get("epic_key"), Role: r.PostForm.Get("role"), Title: r.PostForm.Get("title"), Description: r.PostForm.Get("description"), Priority: priority, Owner: r.PostForm.Get("owner"), Areas: areas, TargetKey: r.PostForm.Get("target_key"), TargetCommit: r.PostForm.Get("target_commit"), Category: r.PostForm.Get("category"), Complexity: r.PostForm.Get("complexity"), ExternalID: r.PostForm.Get("external_id"), Force: r.PostForm.Get("force") == "true", ForceReason: r.PostForm.Get("force_reason")}
	case "claim":
		input = core.NextInput{RequestID: id, TaskKey: key}
	case "answer", "release", "heartbeat":
		generation, err := strconv.ParseInt(r.PostForm.Get("generation"), 10, 64)
		if err != nil || generation < 1 {
			s.mutationError(w, r, b, core.Fail("validation", "Нужно действующее поколение владения"))
			return
		}
		lease := core.LeaseInput{RequestID: id, TaskKey: key, Generation: generation, Reason: r.PostForm.Get("reason")}
		if op == "answer" && (strings.TrimSpace(r.PostForm.Get("answer")) == "" || len(r.PostForm.Get("answer")) > 16384) {
			s.mutationError(w, r, b, core.Fail("validation", "Введите непустой ответ до 16 КиБ"))
			return
		}
		if op == "release" && (strings.TrimSpace(lease.Reason) == "" || len(lease.Reason) > 4096) {
			s.mutationError(w, r, b, core.Fail("validation", "Укажите причину до 4 КиБ"))
			return
		}
		if op != "heartbeat" {
			if err := formTextError(r.PostForm, nil, []string{"answer", "reason"}); err != nil {
				s.mutationError(w, r, b, err)
				return
			}
		}
		if op == "answer" {
			input = core.TransitionInput{LeaseInput: lease, State: "done", Evidence: &core.Evidence{Answer: r.PostForm.Get("answer")}}
		} else if op == "release" {
			input = lease
		} else {
			input = core.HeartbeatInput{RequestID: id, TaskKey: r.PostForm.Get("task_key"), Generation: generation}
		}
	}
	raw, err := json.Marshal(input)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	a, err := s.bindAction(r.Context(), b, id, op, raw)
	if err != nil {
		s.mutationError(w, r, b, err)
		return
	}
	out, err := s.runAction(r.Context(), b, a)
	if err != nil {
		s.mutationError(w, r, b, err)
		return
	}
	if op == "heartbeat" {
		if outWriter, ok := w.(*browserJSONWriter); ok {
			outWriter.respond(http.StatusOK, map[string]any{"ok": true, "session": out.Session, "next_action_id": out.NextActionID})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	http.Redirect(w, r, out.Redirect, 303)
}
func (s *Server) mutationError(w http.ResponseWriter, r *http.Request, b *browserSession, err error) {
	browserErrorCode(w, err)
	_, negotiated := w.(*browserJSONWriter)
	if r.URL.Path == "/web/heartbeat" && !negotiated {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errorStatus(err))
		_ = json.NewEncoder(w).Encode(map[string]string{"error": publicError(err)})
		return
	}
	p, e := s.page(r, b)
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Error = publicError(err)
	p.FieldErrors = fieldErrors(r, err)
	p.View = "tasks"
	if r.PostForm.Get("type") == "" {
		p.View = "questions"
	}
	p.Draft = draft(r)
	p.ActionID = r.PostForm.Get("action_id")
	if r.URL.Path == "/web/heartbeat" {
		id, issueErr := s.issueAction(r.Context(), b, "heartbeat")
		if issueErr != nil {
			s.fail(w, r, b, issueErr)
			return
		}
		p.Actions["heartbeat"] = id
	}
	name := "board"
	if strings.HasPrefix(r.URL.Path, "/web/tasks/") {
		name = "task"
		key := strings.Split(strings.TrimPrefix(r.URL.Path, "/web/tasks/"), "/")[0]
		task, e := s.board.Get(r.Context(), b.actor(), core.GetInput{TaskKey: key})
		if e == nil {
			p.Task = &task
			p.CanEdit = canEdit(b, &task)
			if e = s.taskActions(r.Context(), b, &p, []core.Task{task}); e != nil {
				s.fail(w, r, b, e)
				return
			}
		} else {
			p.CanEdit = false
		}
	}
	// Unknown outcomes retain the same form's action ID. Confirmed domain errors
	// receive a new ID so corrected fields do not conflict with the old payload.
	if errorStatus(err) == 503 {
		operation := "create"
		if r.URL.Path == "/web/heartbeat" {
			operation = "heartbeat"
		}
		if name == "task" {
			parts := strings.Split(r.URL.Path, "/")
			operation = parts[len(parts)-1]
		}
		p.Actions[operation] = p.ActionID
		if p.Task != nil {
			p.Actions[operation+":"+p.Task.Key] = p.ActionID
		}
	}
	s.render(w, errorStatus(err), name, p)
}
