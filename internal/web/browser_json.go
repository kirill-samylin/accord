package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

// Browser JSON is a representation of the existing form/page contract. It must
// never acquire its own authentication, mutation, or idempotency rule path.
type browserJSONWriter struct {
	http.ResponseWriter
	request *http.Request
	body    bytes.Buffer
	status  int
	handled bool
	code    string
}

func (w *browserJSONWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *browserJSONWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *browserJSONWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}
func (w *browserJSONWriter) respond(status int, value any) {
	w.handled = true
	w.Header().Del("Location")
	w.Header().Del("Content-Length")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.ResponseWriter.WriteHeader(status)
	if w.request.Method != http.MethodHead {
		_ = json.NewEncoder(w.ResponseWriter).Encode(value)
	}
}
func (w *browserJSONWriter) finish() {
	if w.handled {
		return
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	if status >= 300 && status < 400 && strings.HasPrefix(w.Header().Get("Location"), "/web/") {
		w.respond(http.StatusOK, map[string]any{"ok": true, "redirect": scopedURL(w.request.Context(), w.Header().Get("Location"))})
		return
	}
	// Raw HTTP errors are deliberately generic. Storage paths, database errors,
	// and token-validation details must not become a second public error API.
	code, message := statusError(status)
	p := projectBrowserPage("error", Page{Draft: draft(w.request), ActionID: w.request.PostForm.Get("action_id")})
	w.respond(status, map[string]any{"ok": false, "error": map[string]string{"code": code, "message": message},
		"page": p, "draft": p.Draft, "action_id": p.ActionID, "actions": p.Actions, "field_errors": p.FieldErrors, "retry_same_action": status >= 500})
}
func statusError(status int) (string, string) {
	switch status {
	case 400:
		return "VALIDATION", "Проверьте поля формы"
	case 401:
		return "UNAUTHORIZED", "Доступ истёк или отозван. Войдите снова"
	case 403:
		return "FORBIDDEN", "Обновите форму: проверка безопасности не пройдена"
	case 404:
		return "NOT_FOUND", "Не найдено"
	case 405:
		return "METHOD_NOT_ALLOWED", "Метод недоступен"
	case 429:
		return "RATE_LIMITED", "Повторите позже"
	default:
		return "UNAVAILABLE", "Сервис временно недоступен. Результат действия не подтверждён; повторите ту же отправку"
	}
}
func browserJSONRoute(path string) bool {
	if !strings.HasPrefix(path, "/web/") || strings.HasPrefix(path, "/web/static/") || strings.HasPrefix(path, "/web/data/") {
		return false
	}
	// Binary transfers retain their download/preview representation regardless
	// of Accept, including deadline and streaming behavior.
	return !strings.HasPrefix(path, "/web/designs/")
}
func wantsBrowserJSON(r *http.Request) bool {
	jsonQ, htmlQ := 0.0, 0.0
	for _, value := range r.Header.Values("Accept") {
		for _, part := range strings.Split(value, ",") {
			media, params, err := mime.ParseMediaType(strings.TrimSpace(part))
			if err != nil {
				continue
			}
			q := 1.0
			if raw, ok := params["q"]; ok {
				q, err = strconv.ParseFloat(raw, 64)
				if err != nil || q < 0 || q > 1 {
					continue
				}
			}
			if media == "application/json" && q > jsonQ {
				jsonQ = q
			}
			if media == "text/html" && q > htmlQ {
				htmlQ = q
			}
		}
	}
	return jsonQ > 0 && jsonQ >= htmlQ
}
func browserErrorCode(w http.ResponseWriter, err error) {
	if out, ok := w.(*browserJSONWriter); ok {
		var domain *core.Error
		var design *designError
		switch {
		case errors.As(err, &domain):
			out.code = domain.Code
		case errors.As(err, &design):
			out.code = design.code
		default:
			out.code = "UNAVAILABLE"
		}
	}
}
func renderBrowserJSON(w http.ResponseWriter, status int, name string, p Page) bool {
	out, ok := w.(*browserJSONWriter)
	if !ok {
		return false
	}
	page := projectBrowserPage(name, p)
	scopePage(out.request.Context(), &page)
	if status < 400 {
		out.respond(status, map[string]any{"ok": true, "page": page})
		return true
	}
	code, message := statusError(status)
	if out.code != "" {
		code = out.code
	}
	if p.Error != "" {
		message = p.Error
	}
	out.respond(status, map[string]any{
		"ok": false, "error": map[string]string{"code": code, "message": message}, "page": page,
		"draft": page.Draft, "field_errors": p.FieldErrors, "action_id": p.ActionID,
		"actions": p.Actions, "retry_same_action": status >= 500 || status == 429 && p.ActionID != "",
	})
	return true
}
