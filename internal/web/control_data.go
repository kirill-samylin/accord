package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agent-board/internal/core"
)

// controlData is deliberately independent of page/action/session continuation.
func (s *Server) controlData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.URL.Path != "/web/data/overview" && r.URL.Path != "/web/data/dashboard" && r.URL.Path != "/web/data/team" && r.URL.Path != "/web/data/similar" {
		s.controlJSONError(w, core.Fail("not_found", "Не найдено"))
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := controlQuery(r); err != nil {
		s.controlDataError(w, r, err)
		return
	}
	b, err := s.loadBrowser(r)
	if err != nil {
		s.controlDataError(w, r, err)
		return
	}
	if !b.Session.ExpiresAt.After(time.Now()) {
		s.controlDataError(w, r, core.Fail("session_expired", "Продолжите сессию явно"))
		return
	}
	var payload any
	switch r.URL.Path {
	case "/web/data/dashboard":
		payload, err = s.readDashboard(r.Context(), b.actor())
	case "/web/data/team":
		payload, err = s.board.ControlTeam(r.Context(), b.actor())
	case "/web/data/similar":
		q := r.URL.Query()
		payload, err = s.board.Similar(r.Context(), b.actor(), core.SimilarInput{Title: q.Get("title"), Description: q.Get("description"), Type: q.Get("type"), Role: q.Get("role"), ExcludeKey: q.Get("exclude_key"), Limit: 6})
	default:
		payload, err = s.board.ControlOverview(r.Context(), b.actor())
	}
	if err != nil {
		s.controlDataError(w, r, err)
		return
	}
	if r.Method != "HEAD" {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func (s *Server) controlDataError(w http.ResponseWriter, r *http.Request, err error) {
	if r.URL.Path == "/web/data/dashboard" && r.Method == "HEAD" {
		w.WriteHeader(errorStatus(err))
		return
	}
	s.controlJSONError(w, err)
}
func (s *Server) controlJSONError(w http.ResponseWriter, err error) {
	code := "UNAVAILABLE"
	var domain *core.Error
	if errors.As(err, &domain) {
		code = domain.Code
	}
	w.WriteHeader(errorStatus(err))
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "error": publicError(err)})
}
