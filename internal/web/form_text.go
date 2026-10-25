package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

// Browser forms pre-check free text with the same core.PlainText rule the
// store enforces authoritatively, so the person gets a Russian message on the
// exact field before an action payload is bound. Single-line fields reject
// every control character; multi-line bodies allow \n, \t and CRLF only.
const (
	lineTextMessage = "Поле не должно содержать управляющие символы"
	bodyTextMessage = "Текст может содержать переносы строк и табуляцию, но не другие управляющие символы"
	priorityMessage = "Приоритет — целое число от 0 до 100; пустое поле означает 2"
)

// formPriority keeps "omitted" distinct from an explicit P0: an empty field is
// nil (the store default 2), "0" is a real top priority.
func formPriority(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > 100 {
		return nil, core.Fail("validation", priorityMessage)
	}
	return &v, nil
}

// formTextErrors reports control-character violations per field.
func formTextErrors(values url.Values, lines, bodies []string) map[string]string {
	fields := map[string]string{}
	for _, name := range lines {
		if !core.PlainText(values.Get(name), false) {
			fields[name] = lineTextMessage
		}
	}
	for _, name := range bodies {
		if !core.PlainText(values.Get(name), true) {
			fields[name] = bodyTextMessage
		}
	}
	return fields
}

// formTextError returns the first violation as a VALIDATION error, checking
// fields in the given order so the message is deterministic.
func formTextError(values url.Values, lines, bodies []string) error {
	fields := formTextErrors(values, lines, bodies)
	for _, name := range append(append([]string{}, lines...), bodies...) {
		if message := fields[name]; message != "" {
			return core.Fail("validation", message)
		}
	}
	return nil
}

// fieldErrors attributes a rejected task form to its fields; a 400 without a
// recognised field falls back to the title so the message stays visible.
func fieldErrors(r *http.Request, err error) map[string]string {
	fields := map[string]string{}
	if errorStatus(err) != 400 {
		return fields
	}
	values := r.PostForm
	switch {
	case r.URL.Path == "/web/tasks":
		if strings.TrimSpace(values.Get("title")) == "" || len([]rune(values.Get("title"))) > 200 {
			fields["title"] = "Введите заголовок до 200 символов"
		}
		if values.Get("type") == "task" && strings.TrimSpace(values.Get("description")) == "" || len(values.Get("description")) > 16384 {
			fields["description"] = "Введите описание до 16 КиБ"
		}
		if _, e := formPriority(values.Get("priority")); e != nil {
			fields["priority"] = priorityMessage
		}
		for name, message := range formTextErrors(values, []string{"title"}, []string{"description", "force_reason"}) {
			fields[name] = message
		}
		if values.Get("type") == "task" && values.Get("epic_key") == "" {
			fields["epic_key"] = "Выберите эпик"
		}
		if values.Get("type") == "task" && !core.ValidRole(values.Get("role")) {
			fields["role"] = "Выберите роль"
		}
		if values.Get("role") == "reviewer" || values.Get("role") == "tester" {
			if values.Get("target_key") == "" {
				fields["target_key"] = "Укажите проверяемую задачу разработчика"
			}
			if len(values.Get("target_commit")) != 40 {
				fields["target_commit"] = "Укажите полный commit SHA"
			}
		}
	case strings.HasSuffix(r.URL.Path, "/answer"):
		fields["answer"] = publicError(err)
	case strings.HasSuffix(r.URL.Path, "/release"):
		fields["reason"] = publicError(err)
	case strings.HasSuffix(r.URL.Path, "/update"):
		if strings.TrimSpace(values.Get("title")) == "" || len([]rune(values.Get("title"))) > 200 {
			fields["title"] = "Введите заголовок до 200 символов"
		}
		if len(values.Get("description")) > 16384 {
			fields["description"] = "Введите описание до 16 КиБ"
		}
		for name, message := range formTextErrors(values, []string{"title"}, []string{"description"}) {
			fields[name] = message
		}
	}
	if len(fields) == 0 {
		fields["title"] = publicError(err)
	}
	return fields
}
