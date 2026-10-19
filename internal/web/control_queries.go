package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"agent-board/internal/core"
)

// controlQuery rejects ambiguous and unknown parameters before any board read.
func controlQuery(r *http.Request) error {
	allowed := ""
	switch {
	case r.URL.Path == "/web/data/overview", r.URL.Path == "/web/overview", r.URL.Path == "/web/projects", r.URL.Path == "/web/settings":
	case r.URL.Path == "/web/":
		allowed = "view cursor epic_cursor state role epic_key owner include_closed q layout"
	case r.URL.Path == "/web/tasks", r.URL.Path == "/web/questions":
		allowed = "cursor epic_cursor state role epic_key owner include_closed q layout"
	case r.URL.Path == "/web/agents":
		allowed = "cursor limit role owner state kind include_closed"
	case strings.HasPrefix(r.URL.Path, "/web/agents/"):
		allowed = "cursor type"
	case r.URL.Path == "/web/activity":
		allowed = "cursor limit task_key session_id type"
	case r.URL.Path == "/web/stats":
		allowed = "days group_by"
	case r.URL.Path == "/web/history":
		allowed = "cursor epic_key"
	case strings.HasPrefix(r.URL.Path, "/web/tasks/"):
		allowed = ""
	default:
		return nil
	}
	values, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		return core.Fail("validation", "Неверные параметры URL")
	}
	var err error
	for key, items := range values {
		if key == "" || !strings.Contains(" "+allowed+" ", " "+key+" ") || len(items) != 1 {
			return core.Fail("validation", "Неизвестный или повторный параметр: "+key)
		}
		value := items[0]
		switch key {
		case "view":
			if value != "" && value != "tasks" && value != "questions" {
				err = core.Fail("validation", "Неверный вид страницы")
			}
		case "state":
			valid := value == ""
			states := []string{"created", "ready", "in_progress", "done", "cancelled"}
			if r.URL.Path == "/web/agents" {
				states = []string{"active", "expired", "credential_invalid"}
			}
			for _, state := range states {
				if state == value {
					valid = true
				}
			}
			if !valid {
				err = core.Fail("validation", "Неверное состояние")
			}
		case "role":
			if value != "" && !core.ValidRole(value) {
				err = core.Fail("validation", "Неизвестная роль")
			}
		case "epic_key", "task_key":
			if value != "" && !core.ValidControlTaskKey(value) {
				err = core.Fail("validation", "Неверный ключ задачи")
			}
		case "session_id":
			if value != "" && !core.ValidControlSessionID(value) {
				err = core.Fail("validation", "Неверный ID сессии")
			}
		case "type":
			if value != "" && !core.ValidControlEventType(value) {
				err = core.Fail("validation", "Неизвестный тип события")
			}
		case "layout":
			if value != "" && value != "list" && value != "board" {
				err = core.Fail("validation", "Неверный вид доски")
			}
		case "include_closed":
			if value != "" && value != "true" && value != "false" {
				err = core.Fail("validation", "Неверный флаг закрытых")
			}
		case "kind":
			if value != "" && value != "all" && value != "agent" && value != "human" {
				err = core.Fail("validation", "Неверный тип сессии")
			}
		case "owner":
			if len(value) > 128 {
				err = core.Fail("validation", "Неверный владелец")
			}
		case "q":
			if r.URL.Path != "/web/tasks" && !(r.URL.Path == "/web/" && values.Get("view") == "tasks") {
				return core.Fail("validation", "Поиск доступен в задачах")
			}
			if !utf8.ValidString(value) || len(value) > 1024 || utf8.RuneCountInString(strings.TrimSpace(value)) > 200 {
				err = core.Fail("validation", "Поиск ограничен 200 символами")
			}
		case "days":
			if value != "1" && value != "7" && value != "30" && value != "90" {
				err = core.Fail("validation", "Неверный период")
			}
		case "group_by":
			if value != "role" && value != "owner" && value != "epic" {
				err = core.Fail("validation", "Неверная группировка")
			}
		case "limit":
			_, err = controlInt(value, 1, 100)
		case "cursor", "epic_cursor":
			if len(value) > 1024 {
				err = core.Fail("validation", "Неверная страница")
			}
		}
		if err != nil {
			return err
		}
	}
	if r.URL.Path == "/web/agents" {
		kind := values.Get("kind")
		role := values.Get("role")
		if role != "" && (kind == "agent" && role == "human" || kind == "human" && role != "human" || kind == "" && role == "human") {
			return core.Fail("validation", "Роль не соответствует типу сессии")
		}
	}
	return nil
}
func controlInt(raw string, minValue, maxValue int) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < minValue || n > maxValue {
		return 0, core.Fail("validation", "Неверное числовое значение")
	}
	return n, nil
}
func eventBefore(r *http.Request) (int64, error) {
	value := r.URL.Query().Get("cursor")
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		return 0, core.Fail("validation", "Неверная страница истории")
	}
	return n, nil
}
