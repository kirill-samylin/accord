package webui

import (
	"fmt"
	"net/url"

	"agent-board/internal/core"
	"agent-board/internal/web"
)

var staticAssets = map[string]string{
	"license-paperclip.txt": "text/plain; charset=utf-8",
	"license-inter.txt":     "text/plain; charset=utf-8",
	"sidebar.css":           "text/css; charset=utf-8",
	"cards.css":             "text/css; charset=utf-8",
	"forms.css":             "text/css; charset=utf-8",
	"tasks.css":             "text/css; charset=utf-8",
	"app.css":               "text/css; charset=utf-8",
	"shell.css":             "text/css; charset=utf-8",
	"control.css":           "text/css; charset=utf-8",
	"app.js":                "text/javascript; charset=utf-8",
	"control.js":            "text/javascript; charset=utf-8",
	"theme.js":              "text/javascript; charset=utf-8",
	"drawer.js":             "text/javascript; charset=utf-8",
	"commands.js":           "text/javascript; charset=utf-8",
	"dashboard.js":          "text/javascript; charset=utf-8",
	"commands.css":          "text/css; charset=utf-8",
	"InterVariable.woff2":   "font/woff2",
}

type navItem struct {
	View, URL, Label, Icon string
	Active                 bool
}

func navigation(p web.Page) []navItem {
	items := []navItem{
		{View: "overview", URL: "/web/", Label: "Обзор", Icon: "overview"},
		{View: "questions", URL: "/web/questions", Label: "Вопросы", Icon: "questions"},
		{View: "tasks", URL: "/web/tasks", Label: "Задачи", Icon: "tasks"},
		{View: "agents", URL: "/web/agents", Label: "Агенты", Icon: "agents"},
		{View: "projects", URL: "/web/projects", Label: "Проекты", Icon: "projects"},
		{View: "activity", URL: "/web/activity", Label: "Активность", Icon: "activity"},
		{View: "stats", URL: "/web/stats", Label: "Статистика", Icon: "stats"},
		{View: "designs", URL: "/web/designs", Label: "Дизайн", Icon: "designs"},
		{View: "history", URL: "/web/history", Label: "История решений", Icon: "history"},
		{View: "settings", URL: "/web/settings", Label: "Настройки", Icon: "settings"},
	}
	for i := range items {
		items[i].Active = items[i].View == p.View || (items[i].View == "agents" && p.View == "agent") || (items[i].View == "tasks" && p.View == "task")
	}
	return items
}
func agentURL(id string) string { return "/web/agents/" + url.PathEscape(id) }
func sessionStatus(s string) string {
	switch s {
	case "active":
		return "Сессия активна"
	case "expired":
		return "Срок сессии истёк"
	case "credential_invalid":
		return "Доступ недействителен"
	}
	return "Неизвестное состояние"
}
func percent(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }

var eventLabels = map[string]string{
	"token_created":      "Доступ выдан",
	"token_revoked":      "Доступ отозван",
	"tokens_revoked_all": "Все доступы отозваны",
	"session_opened":     "Сессия открыта",
	"task_created":       "Задача создана",
	"task_updated":       "Задача обновлена",
	"claimed":            "Задача взята",
	"released":           "Задача возвращена",
	"commented":          "Комментарий добавлен",
	"linked":             "Связь добавлена",
	"unlinked":           "Связь удалена",
	"task_ready":         "Задача возвращена в очередь",
	"task_in_progress":   "Задача в работе",
	"task_done":          "Задача завершена",
	"task_cancelled":     "Задача отменена",
	"human_answered":     "Получен ответ человека",
	"epic_open":          "Эпик открыт",
	"epic_done":          "Эпик завершён",
	"epic_cancelled":     "Эпик отменён",
	"epic_reopened":      "Эпик открыт заново",
	"lease_expired":      "Владение истекло",
	"lease_recovered":    "Владение восстановлено",
	"attempt_recovered":  "Попытка восстановлена",
	"database_recovered": "Хранилище восстановлено",
	"note_created":       "Заметка создана",
	"note_deleted":       "Заметка удалена",
	"blocked_changed":    "Блокировка изменена",
}

func eventLabel(s string) string {
	if label, ok := eventLabels[s]; ok {
		return label
	}
	return s
}

type boardColumn struct {
	State string
	Tasks []core.Task
}

// Board columns group only the returned page; they never imply project totals.
func boardColumns(tasks []core.Task) []boardColumn {
	columns := []boardColumn{{State: "created"}, {State: "ready"}, {State: "in_progress"}, {State: "done"}, {State: "cancelled"}}
	for _, task := range tasks {
		found := false
		for i := range columns {
			if columns[i].State == task.State {
				columns[i].Tasks = append(columns[i].Tasks, task)
				found = true
				break
			}
		}
		if !found {
			columns = append(columns, boardColumn{State: task.State, Tasks: []core.Task{task}})
		}
	}
	return columns
}

func ruleLabel(key string) string {
	labels := map[string]string{"independent_review": "Независимое ревью",
		"one_task_per_agent": "Одна задача на агента",
		"cycle_check":        "Проверка циклов зависимостей",
		"singleton_role":     "Одна активная сессия для единоличной роли"}
	if label, ok := labels[key]; ok {
		return label
	}
	return key
}
func ruleValue(value string) string {
	switch value {
	case "block":
		return "Обязательное правило"
	case "warn":
		return "Предупреждение"
	case "off":
		return "Выключено"
	}
	return value
}

// Kind changes use separate links so a no-JavaScript form cannot submit an
// incompatible human/agent role retained from the previous kind.
func agentRoles(p web.Page) []string {
	var roles []string
	for _, role := range p.Roles {
		kind := p.Draft["kind"]
		if kind == "human" && role != "human" {
			continue
		}
		if kind != "all" && kind != "human" && role == "human" {
			continue
		}
		roles = append(roles, role)
	}
	return roles
}
