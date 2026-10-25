package web

import (
	"agent-board/internal/core"
	"errors"
	"net/http"
	"strings"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) && !WantsBrowserJSON(r) {
		if target, ok := BrowserPageDestination(r); ok {
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, target, http.StatusSeeOther)
			return
		}
		if BrowserJSONRoute(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
	}
	if BrowserJSONRoute(r.URL.Path) {
		w.Header().Add("Vary", "Accept")
		if WantsBrowserJSON(r) || r.Method == http.MethodPost {
			out := &browserJSONWriter{ResponseWriter: w, request: r}
			w = out
			defer out.finish()
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Keep same-origin browser requests attributable without disclosing a
	// referrer to other sites; JSON clients retain the existing Origin boundary.
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	if strings.HasPrefix(r.URL.Path, "/api/v1/designs") {
		w.Header().Set("Referrer-Policy", "no-referrer")
		s.handleDesignsAPI(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/web/") {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == "/web/data/dashboard" {
		s.controlData(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "Метод недоступен", 405)
		return
	}
	if r.Method == "POST" {
		origins := r.Header.Values("Origin")
		if len(origins) != 1 || origins[0] != s.options.Origin || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			// Preserve the bounded public draft for JSON clients even when Origin
			// fails. Non-JSON submissions never reach body parsing or mutations.
			if WantsBrowserJSON(r) && r.URL.Path != "/web/login" && r.URL.Path != "/web/designs" {
				_ = parseForm(w, r)
			}
			s.render(w, 403, "error", Page{Title: "Действие отклонено", Error: "Обновите форму: источник запроса не разрешён", Draft: draft(r)})
			return
		}
		if !WantsBrowserJSON(r) {
			http.Error(w, "Откройте Accord в веб-клиенте", http.StatusNotAcceptable)
			return
		}
		if r.URL.Path != "/web/login" && r.URL.Path != "/web/designs" {
			if err := parseForm(w, r); err != nil {
				s.fail(w, r, nil, err)
				return
			}
		}
	}
	if strings.HasPrefix(r.URL.Path, "/web/static/") {
		http.NotFound(w, r)
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
	if strings.HasPrefix(r.URL.Path, "/web/data/") {
		s.controlData(w, r)
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
	if r.URL.Path == "/web/profiles" || strings.HasPrefix(r.URL.Path, "/web/profiles/") {
		s.handleProfiles(w, r, b)
		return
	}
	if r.URL.Path == "/web/automation" || strings.HasPrefix(r.URL.Path, "/web/automation/") {
		s.handleAutomation(w, r, b)
		return
	}
	if r.URL.Path == "/web/managed" || strings.HasPrefix(r.URL.Path, "/web/managed/") {
		s.handleManaged(w, r, b)
		return
	}
	if r.URL.Path == "/web/designs" || strings.HasPrefix(r.URL.Path, "/web/designs/") {
		s.handleDesigns(w, r, b)
		return
	}
	if r.Method == "POST" {
		if key, op, ok := coordinationRoute(r.URL.Path); ok {
			s.handleCoordination(w, r, b, key, op)
			return
		}
		s.mutate(w, r, b)
		return
	}
	s.read(w, r, b)
}
func parseForm(w http.ResponseWriter, r *http.Request) error {
	maxBody := int64(32 * 1024)
	// A 16 KiB description can expand threefold under URL encoding. Update
	// still validates decoded field sizes before reaching the authoritative API.
	if strings.HasPrefix(r.URL.Path, "/web/automation") || strings.HasPrefix(r.URL.Path, "/web/profiles") || strings.HasPrefix(r.URL.Path, "/web/managed") || strings.HasPrefix(r.URL.Path, "/web/tasks/") && strings.HasSuffix(r.URL.Path, "/update") {
		maxBody = 64 * 1024
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
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
	if s.options.ProjectInfo != nil {
		if current, err := s.options.ProjectInfo(projectResponseContext(w)); err == nil {
			p.Project = current
		}
	}
	if renderBrowserJSON(w, status, name, p) {
		return
	}
	if status >= http.StatusBadRequest {
		_, message := statusError(status)
		http.Error(w, message, status)
		return
	}
	http.Error(w, "Требуется JSON-клиент", http.StatusNotAcceptable)
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
func (s *Server) fail(w http.ResponseWriter, r *http.Request, b *browserSession, err error) {
	browserErrorCode(w, err)
	p := Page{Title: "Действие не выполнено", Error: publicError(err), Draft: draft(r)}
	if b != nil {
		p.Principal = b.Principal
		p.Session = b.Session
		p.CSRF = b.CSRF
		p.ActionID = r.PostForm.Get("action_id")
	}
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			p.Draft[key] = values[0]
		}
	}
	s.render(w, errorStatus(err), "error", p)
}
