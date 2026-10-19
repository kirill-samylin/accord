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
	if r.URL.Path != "/web/data/overview" {
		s.controlJSONError(w, core.Fail("not_found", "Не найдено"))
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := controlQuery(r); err != nil {
		s.controlJSONError(w, err)
		return
	}
	b, err := s.loadBrowser(r)
	if err != nil {
		s.controlJSONError(w, err)
		return
	}
	if !b.Session.ExpiresAt.After(time.Now()) {
		s.controlJSONError(w, core.Fail("session_expired", "Продолжите сессию явно"))
		return
	}
	overview, err := s.board.ControlOverview(r.Context(), b.actor())
	if err != nil {
		s.controlJSONError(w, err)
		return
	}
	if r.Method != "HEAD" {
		_ = json.NewEncoder(w).Encode(overview)
	}
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
