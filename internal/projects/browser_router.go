package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/web"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, err error) {
	code, message, status := "UNAVAILABLE", "Сервис временно недоступен", 503
	editable := false
	var domain *core.Error
	if errors.As(err, &domain) {
		editable = editableRejection(domain)
		code = domain.Code
		switch code {
		case "UNAUTHORIZED":
			status = 401
			message = "Войдите снова"
		case "NOT_FOUND":
			status = 404
			message = "Проект или участник не найден"
		case "FORBIDDEN_ROLE":
			status = 403
			message = "Недостаточно прав"
		case "VALIDATION":
			status = 400
			message = "Проверьте поля формы"
		default:
			status = 409
			message = "Состояние изменилось: обновите форму"
		}
		if specific := registryMessage(domain); specific != "" {
			message = specific
		}
	}
	write(w, status, map[string]any{"ok": false, "error": map[string]string{"code": code, "message": message}, "retry_same_action": status >= 500, "editable": editable})
}
func redirect(w http.ResponseWriter, req *http.Request, target string) {
	if web.WantsBrowserJSON(req) {
		write(w, 200, map[string]any{"ok": true, "redirect": target})
		return
	}
	http.Redirect(w, req, target, 303)
}
func (r *Registry) allowedBrowser(req *http.Request) bool {
	return req.Method != "POST" || len(req.Header.Values("Origin")) == 1 && req.Header.Get("Origin") == r.options.Origin && req.Header.Get("Sec-Fetch-Site") != "cross-site"
}
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/mcp") {
		r.serveNative(w, req)
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	if !r.allowedBrowser(req) {
		fail(w, denied())
		return
	}
	if target, ok := web.BrowserPageDestination(req); ok {
		http.Redirect(w, req, target, http.StatusSeeOther)
		return
	}
	if (req.Method == http.MethodGet || req.Method == http.MethodHead) && !web.WantsBrowserJSON(req) {
		_, path, _ := projectRoute(req.URL.Path, "/web/")
		if web.BrowserJSONRoute(path) {
			http.NotFound(w, req)
			return
		}
	}
	if req.Method == http.MethodPost && !web.WantsBrowserJSON(req) {
		write(w, http.StatusNotAcceptable, map[string]any{"ok": false, "error": map[string]string{"code": "JSON_REQUIRED", "message": "Откройте Accord в веб-клиенте"}})
		return
	}
	if req.URL.Path == "/web/login" {
		r.login(w, req)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/web/static/") {
		r.options.Browser.ServeHTTP(w, req)
		return
	}
	if req.URL.Path == "/" {
		redirect(w, req, "/web/projects")
		return
	}
	b, err := r.browser(req)
	if err != nil {
		fail(w, err)
		return
	}
	req = req.WithContext(globalContext(req.Context(), b))
	if req.URL.Path == "/web/logout" && req.Method == "POST" {
		r.logout(w, req, b, r.options.Browser)
		return
	}
	if req.URL.Path == "/web/projects" || strings.HasPrefix(req.URL.Path, "/web/catalog/") {
		r.catalogHTTP(w, req, b)
		return
	}
	pid, path, scoped := projectRoute(req.URL.Path, "/web/")
	if !scoped {
		pid = r.defaultID
		path = req.URL.Path
	}
	a, err := transaction(r, req.Context(), func(tx *sql.Tx) (access, error) { return r.access(req.Context(), tx, b.Identity, pid, false) })
	if err != nil {
		fail(w, err)
		return
	}
	v, err := r.runtime(pid)
	if err != nil {
		fail(w, err)
		return
	}
	if path == "/web/logout" && req.Method == "POST" {
		r.logout(w, req, b, v.browser)
		return
	}
	binding, err := v.browser.ProjectBinding(req.Context(), b.Hash)
	if err != nil || binding.SessionID == "" || v.service.SessionAllowed(req.Context(), binding.SessionID) != nil {
		target := "/web/p/" + pid + strings.TrimPrefix(path, "/web")
		if req.URL.RawQuery != "" {
			target += "?" + req.URL.RawQuery
		}
		write(w, 409, map[string]any{"ok": false, "error": map[string]string{"code": "PROJECT_SESSION_REQUIRED", "message": "Откройте проект"}, "project": a.project, "target": target})
		return
	}
	cloned := req.Clone(web.WithProjectScope(req.Context(), func() string {
		if scoped {
			return pid
		}
		return ""
	}()))
	copied := *req.URL
	copied.Path = path
	copied.RawPath = ""
	cloned.URL = &copied
	v.browser.ServeHTTP(w, cloned)
}
func query(req *http.Request) string {
	if req.URL.RawQuery != "" {
		return "?" + req.URL.RawQuery
	}
	return ""
}
func projectRoute(path, prefix string) (string, string, bool) {
	if !strings.HasPrefix(path, prefix+"p/") {
		return "", path, false
	}
	tail := strings.TrimPrefix(path, prefix+"p/")
	parts := strings.SplitN(tail, "/", 2)
	id := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}
	return id, prefix + rest, true
}
func (r *Registry) serveNative(w http.ResponseWriter, req *http.Request) {
	fields := strings.Fields(req.Header.Get("Authorization"))
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		fail(w, core.Fail("unauthorized", "bearer required"))
		return
	}
	pid, path, scoped := projectRoute(req.URL.Path, "/api/v1/")
	if strings.HasPrefix(req.URL.Path, "/mcp/p/") {
		pid = strings.TrimPrefix(req.URL.Path, "/mcp/p/")
		path = "/mcp"
		scoped = true
	}
	if !scoped {
		pid = r.defaultID
		path = req.URL.Path
	}
	ctx, who, e := r.nativeIdentity(req.Context(), fields[1], pid, path, scoped)
	if e != nil {
		fail(w, core.Fail("unauthorized", "bearer required"))
		return
	}
	if _, e = transaction(r, ctx, func(tx *sql.Tx) (access, error) { return r.access(ctx, tx, who, pid, false) }); e != nil {
		fail(w, e)
		return
	}
	v, e := r.runtime(pid)
	if e != nil {
		fail(w, e)
		return
	}
	cloned := req.Clone(web.WithProjectScope(ctx, func() string {
		if scoped {
			return pid
		}
		return ""
	}()))
	copied := *req.URL
	copied.Path = path
	copied.RawPath = ""
	cloned.URL = &copied
	v.native.ServeHTTP(w, cloned)
}
func (r *Registry) switchProject(ctx context.Context, b browserIdentity, pid, target string) (string, error) {
	ctx = globalContext(ctx, b)
	expected := "/web/p/" + pid + "/"
	if target == "" {
		target = expected
	}
	u, e := url.Parse(target)
	if e != nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, expected) || strings.Contains(u.Path, "..") || strings.Contains(u.Path, "\\") {
		return "", invalid()
	}
	a, e := transaction(r, ctx, func(tx *sql.Tx) (access, error) { return r.access(ctx, tx, b.Identity, pid, false) })
	if e != nil {
		return "", e
	}
	v, e := r.runtime(pid)
	if e != nil {
		return "", e
	}
	if e = v.browser.OpenProject(accountContext(ctx, b.Identity), b.Hash, a.principal, b.Expires); e != nil {
		return "", e
	}
	return target, nil
}
func (r *Registry) logout(w http.ResponseWriter, req *http.Request, b browserIdentity, browser *web.Server) {
	if e := catalogForm(w, req); e != nil {
		fail(w, e)
		return
	}
	authorized := false
	if equal(req.PostForm.Get("csrf"), b.CSRF) {
		_, e := r.bindCatalog(req.Context(), b, req.PostForm.Get("action_id"), "logout", "", map[string]string{}, false)
		authorized = e == nil
	}
	if !authorized && browser != nil {
		authorized = browser.ValidateLogout(req) == nil
	}
	if !authorized {
		fail(w, denied())
		return
	}
	if e := r.endSession(req.Context(), b); e != nil {
		fail(w, e)
		return
	}
	r.cookie(w, r.cookieName(), "", time.Time{})
	redirect(w, req, "/web/login")
}
