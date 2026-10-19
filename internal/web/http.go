package web

import (
	"agent-board/internal/core"
	"errors"
	"net/http"
	"strings"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Basic HTML form POSTs serialize Origin as null under no-referrer.
	// same-origin preserves their Origin while disclosing no referrer off-site.
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	if strings.HasPrefix(r.URL.Path, "/api/v1/designs") {
		w.Header().Set("Referrer-Policy", "no-referrer")
		s.handleDesignsAPI(w, r)
		return
	}
	if r.URL.Path == "/" && (r.Method == "GET" || r.Method == "HEAD") {
		http.Redirect(w, r, "/web/", 303)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/web/") {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Метод недоступен", 405)
		return
	}
	if r.Method == "POST" {
		if r.URL.Path != "/web/login" && r.URL.Path != "/web/designs" {
			if err := parseForm(w, r); err != nil {
				s.fail(w, r, nil, err)
				return
			}
		}
		origins := r.Header.Values("Origin")
		if len(origins) != 1 || origins[0] != s.options.Origin || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			s.render(w, 403, "error", Page{Title: "Действие отклонено", Error: "Обновите форму: источник запроса не разрешён", Draft: draft(r)})
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, "/web/static/") && (r.Method == "GET" || r.Method == "HEAD") {
		if s.options.StaticHandler == nil {
			http.NotFound(w, r)
		} else {
			s.options.StaticHandler.ServeHTTP(w, r)
		}
		return
	}
	if r.URL.Path == "/web/login" {
		if r.Method == "POST" {
			s.login(w, r)
		} else {
			s.loginPage(w, r, "", 200)
		}
		return
	}
	// Mutations of a browser session are serialized through board call and local
	// result persistence, preventing two tabs from racing session continuation.
	if r.Method == "POST" && r.URL.Path != "/web/designs" {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	b, err := s.loadBrowser(r)
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	if r.Method == "POST" && r.URL.Path != "/web/designs" {
		csrf := r.PostForm.Get("csrf")
		if values := r.Header.Values("X-CSRF-Token"); len(values) == 1 {
			csrf = values[0]
		}
		if !constantEqual(csrf, b.CSRF) {
			if r.URL.Path == "/web/heartbeat" {
				s.mutationError(w, r, b, core.Fail("forbidden", "Проверка безопасности не пройдена"))
				return
			}
			s.render(w, 403, "error", Page{Title: "Действие отклонено", Error: "Обновите форму: проверка безопасности не пройдена", Draft: draft(r)})
			return
		}
	}
	if r.URL.Path == "/web/designs" || strings.HasPrefix(r.URL.Path, "/web/designs/") {
		s.handleDesigns(w, r, b)
		return
	}
	if r.Method == "POST" {
		s.mutate(w, r, b)
		return
	}
	s.read(w, r, b)
}
func parseForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return core.Fail("validation", "expected form body")
	}
	if err := r.ParseForm(); err != nil {
		return core.Fail("validation", "form too large or malformed")
	}
	for _, v := range r.PostForm {
		if len(v) != 1 {
			return core.Fail("validation", "duplicate form fields")
		}
	}
	return nil
}
func draft(r *http.Request) map[string]string {
	d := map[string]string{}
	for k, v := range r.PostForm {
		if k != "token" && k != "token_file" && k != "csrf" && len(v) == 1 {
			d[k] = v[0]
		}
	}
	return d
}
func (s *Server) render(w http.ResponseWriter, status int, name string, p Page) {
	if err := s.options.Renderer.Render(w, status, name, p); err != nil {
		http.Error(w, "Страница временно недоступна", 500)
	}
}
func errorStatus(err error) int {
	var e *core.Error
	if !errors.As(err, &e) {
		return 503
	}
	switch e.Code {
	case "UNAUTHORIZED":
		return 401
	case "FORBIDDEN_ROLE":
		return 403
	case "NOT_FOUND":
		return 404
	case "VALIDATION":
		return 400
	default:
		return 409
	}
}
func publicError(err error) string {
	var e *core.Error
	if !errors.As(err, &e) {
		return "Сервис временно недоступен. Результат действия не подтверждён; повторите ту же отправку"
	}
	switch e.Code {
	case "UNAUTHORIZED":
		return "Доступ истёк или отозван. Войдите снова"
	case "FORBIDDEN_ROLE":
		return "Недостаточно прав для этого действия"
	case "NOT_FOUND":
		return "Не найдено"
	case "LEASE_LOST":
		return "Время ответа истекло. Текст сохранён на этой странице. Обновите состояние вопроса"
	default:
		return e.Message
	}
}
func (s *Server) fail(w http.ResponseWriter, r *http.Request, b *browserSession, err error) {
	p := Page{Title: "Действие не выполнено", Error: publicError(err), Draft: draft(r)}
	if b != nil {
		p.Principal = b.Principal
		p.Session = b.Session
		p.CSRF = b.CSRF
		p.ActionID = r.PostForm.Get("action_id")
	}
	s.render(w, errorStatus(err), "error", p)
}
