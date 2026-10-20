package web

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"agent-board/internal/core"
)

type designError struct {
	status        int
	code, message string
}

func (e *designError) Error() string                    { return e.message }
func designFail(status int, code, message string) error { return &designError{status, code, message} }
func designEqual(a, b string) bool {
	return len(a) > 0 && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func designErrorInfo(err error) (int, string, string) {
	var e *designError
	if errors.As(err, &e) {
		return e.status, e.code, e.message
	}
	var ce *core.Error
	if errors.As(err, &ce) {
		switch ce.Code {
		case "AUTH_REQUIRED", "UNAUTHENTICATED", "UNAUTHORIZED", "TOKEN_EXPIRED":
			return 401, ce.Code, "Срок доступа истёк или доступ отозван. Войдите снова"
		case "NOT_FOUND":
			return 404, ce.Code, "Не найдено"
		case "VALIDATION":
			return 400, ce.Code, "Проверьте поля формы"
		case "LEASE_LOST", "NO_SESSION":
			return 409, ce.Code, "Срок работы истёк. Обновите вход"
		case "FORBIDDEN_ROLE":
			return 403, ce.Code, "Недостаточно прав для этого действия"
		case "CONFLICT", "IDEMPOTENCY_CONFLICT":
			return 409, ce.Code, "Запрос конфликтует с предыдущим действием"
		}
	}
	return 500, "INTERNAL", "Не удалось обработать запрос. Повторите позже"
}
func designHTTPError(w http.ResponseWriter, err error) {
	status, code, message := designErrorInfo(err)
	if out, ok := w.(*browserJSONWriter); ok {
		out.respond(status, map[string]any{"ok": false, "error": map[string]string{"code": code, "message": message}, "retry_same_action": status >= 500})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func designDeadline(w http.ResponseWriter) error {
	controller := http.NewResponseController(w)
	deadline := time.Now().Add(120 * time.Second)
	if err := controller.SetReadDeadline(deadline); err != nil {
		return designFail(503, "DEADLINE", "Загрузка файлов временно недоступна")
	}
	if err := controller.SetWriteDeadline(deadline); err != nil {
		return designFail(503, "DEADLINE", "Передача файлов временно недоступна")
	}
	return nil
}
func designFilter(r *http.Request) (DesignFilter, error) {
	q := r.URL.Query()
	for key, values := range q {
		if len(values) != 1 {
			return DesignFilter{}, designFail(400, "VALIDATION", "Параметр повторяется")
		}
		switch key {
		case "scope", "scope_key", "design_id", "cursor", "limit":
		default:
			return DesignFilter{}, designFail(400, "VALIDATION", "Неизвестный параметр")
		}
	}
	f := DesignFilter{Scope: q.Get("scope"), ScopeKey: q.Get("scope_key"), DesignID: q.Get("design_id"), Cursor: q.Get("cursor")}
	if q.Get("limit") != "" {
		limit, err := strconv.Atoi(q.Get("limit"))
		if err != nil {
			return f, designFail(400, "VALIDATION", "Неверный размер страницы")
		}
		f.Limit = limit
	}
	return f, nil
}

func (s *Server) handleDesigns(w http.ResponseWriter, r *http.Request, b *browserSession) {
	path := strings.TrimPrefix(r.URL.Path, "/web/designs")
	if path == "" && r.Method == http.MethodPost {
		asset, err := s.uploadDesign(w, r, b)
		if err != nil {
			s.designPageError(w, r, b, err)
			return
		}
		http.Redirect(w, r, "/web/designs?design_id="+asset.DesignID, http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Метод не поддерживается", 405)
		return
	}
	if path != "" {
		fields := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(fields) != 2 || (fields[1] != "download" && fields[1] != "preview") {
			http.NotFound(w, r)
			return
		}
		if err := s.designActor(r.Context(), b.actor()); err != nil {
			designHTTPError(w, err)
			return
		}
		v, err := s.designs.get(r.Context(), fields[0])
		if err != nil {
			designHTTPError(w, err)
			return
		}
		s.serveDesign(w, r, v, fields[1] == "preview")
		return
	}
	f, err := designFilter(r)
	if err != nil {
		s.designPageError(w, r, b, err)
		return
	}
	page, err := s.ListDesigns(r.Context(), b.actor(), f)
	if err != nil {
		s.designPageError(w, r, b, err)
		return
	}
	assets := page.Assets
	for i := range assets {
		assets[i].DownloadURL = "/web/designs/" + assets[i].AssetID + "/download"
		v, e := s.designs.get(r.Context(), assets[i].AssetID)
		if e != nil {
			s.designPageError(w, r, b, e)
			return
		}
		if v.Preview != "" {
			assets[i].PreviewURL = "/web/designs/" + assets[i].AssetID + "/preview"
		}
	}
	s.renderDesignPage(w, r, b, Page{Title: "Дизайн", Designs: assets, NextCursor: page.NextCursor, Draft: map[string]string{"scope": f.Scope, "scope_key": f.ScopeKey, "design_id": f.DesignID}}, 200)
}

func (s *Server) designPageError(w http.ResponseWriter, r *http.Request, b *browserSession, err error) {
	browserErrorCode(w, err)
	status, _, message := designErrorInfo(err)
	action := r.PostForm.Get("action_id")
	if status == 400 || status == 413 {
		action = ""
	}
	s.renderDesignPage(w, r, b, Page{Title: "Дизайн", Error: message, Draft: draft(r), ActionID: action}, status)
}
func (s *Server) renderDesignPage(w http.ResponseWriter, r *http.Request, b *browserSession, page Page, status int) {
	page.Principal = b.Principal
	page.Session = b.Session
	page.CSRF = b.CSRF
	// The backend issues fresh operation-bound IDs when rendering a page.
	base, err := s.page(r, b)
	if err != nil {
		designHTTPError(w, err)
		return
	}
	page.Actions = base.Actions
	if page.ActionID != "" {
		page.Actions["upload"] = page.ActionID
	} else {
		page.ActionID = base.Actions["upload"]
	}
	page.CanContinue = base.CanContinue
	if renderBrowserJSON(w, status, "designs", page) {
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(status)
		return
	}
	if err := s.options.Renderer.Render(w, status, "designs", page); err != nil {
		return
	}
}

func (s *Server) handleDesignsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if _, ok := r.Header["Origin"]; ok {
		designHTTPError(w, designFail(403, "ORIGIN", "Origin запрещён"))
		return
	}
	auth := strings.Fields(r.Header.Get("Authorization"))
	if len(r.Header.Values("Authorization")) != 1 || len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
		designHTTPError(w, designFail(401, "AUTH_REQUIRED", "Требуется вход"))
		return
	}
	p, err := s.board.Authenticate(r.Context(), auth[1])
	if err != nil {
		designHTTPError(w, err)
		return
	}
	if len(r.Header.Values("X-Session")) != 1 || r.Header.Get("X-Session") == "" {
		designHTTPError(w, designFail(401, "NO_SESSION", "Требуется действующая сессия"))
		return
	}
	actor := core.Actor{Principal: p, SessionID: r.Header.Get("X-Session")}
	if err = s.designActor(r.Context(), actor); err != nil {
		designHTTPError(w, err)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/designs")
	if path == "" {
		f, e := designFilter(r)
		if e != nil {
			designHTTPError(w, e)
			return
		}
		page, e := s.ListDesigns(r.Context(), actor, f)
		if e != nil {
			designHTTPError(w, e)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(page)
		}
		return
	}
	fields := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(fields) > 2 || len(fields) == 2 && fields[1] != "download" {
		http.NotFound(w, r)
		return
	}
	v, e := s.designs.get(r.Context(), fields[0])
	if e != nil {
		designHTTPError(w, e)
		return
	}
	if len(fields) == 2 {
		s.serveDesign(w, r, v, false)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodHead {
		_ = json.NewEncoder(w).Encode(designMetadata(v, false))
	}
}

func (s *Server) serveDesign(w http.ResponseWriter, r *http.Request, v storedDesign, preview bool) {
	name, size := v.Blob, v.Size
	if preview {
		if v.Preview == "" {
			http.NotFound(w, r)
			return
		}
		name = v.Preview
		size = v.PreviewSize
	}
	if err := designDeadline(w); err != nil {
		designHTTPError(w, err)
		return
	}
	f, err := designOpen(s.designs.dir, "blobs", name, os.O_RDONLY)
	if err != nil {
		designHTTPError(w, designFail(404, "NOT_FOUND", "Файл не найден"))
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.Size() != size {
		designHTTPError(w, designFail(503, "STORAGE", "Файл временно недоступен"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": v.Filename}))
	if preview {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Disposition", "inline")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.CopyN(w, f, size)
}
