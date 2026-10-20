// Package webui renders the browser interface without interpreting user content.
package webui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/web"
)

//go:embed templates/*.html assets/*
var content embed.FS

// Renderer owns immutable, concurrently reusable HTML templates.
type Renderer struct{ templates *template.Template }

// New parses all templates before the server starts accepting requests.
func New() (*Renderer, error) {
	functions := template.FuncMap{
		"date": date, "state": state, "role": role, "size": size,
		"agentURL": agentURL, "sessionStatus": sessionStatus, "eventLabel": eventLabel,
		"percent": percent, "boardColumns": boardColumns, "layoutURL": layoutURL,
		"eventLabels": func() map[string]string { return eventLabels }, "taskViews": taskViews, "agentRoles": agentRoles, "navigation": navigation, "refreshURL": refreshURL, "ruleLabel": ruleLabel, "ruleValue": ruleValue,
		"draft":  func(p web.Page, key string) string { return p.Draft[key] },
		"action": action, "fieldError": func(p web.Page, key string) string { return p.FieldErrors[key] },
		"taskURL":    func(key string) string { return "/web/tasks/" + url.PathEscape(key) },
		"generation": generation, "canClaim": canClaim, "answer": answer,
		"nextURL": nextURL, "epicsURL": epicsURL, "join": strings.Join, "savedDraft": savedDraft, "scope": scope,
	}
	templates, err := template.New("accord").Funcs(functions).ParseFS(content, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
}

// Render buffers first: a template failure never publishes a partial success page.
func (r *Renderer) Render(w http.ResponseWriter, status int, name string, page web.Page) error {
	switch name {
	case "login", "board", "task", "designs", "history", "error", "overview", "agents", "agent", "activity", "projects", "settings", "stats":
	default:
		return fmt.Errorf("unknown web template %q", name)
	}
	if name != "board" && page.View == "" {
		page.View = name
	}
	if name == "board" && page.View == "" {
		page.View = "questions"
	}
	if page.Title == "" || page.Title == "Accord" {
		titles := map[string]string{"login": "Вход в Accord", "board": "Задачи", "task": "Карточка задачи", "designs": "Дизайн", "history": "История решений", "error": "Действие не выполнено", "overview": "Обзор", "agents": "Агенты", "agent": "Сессия агента", "activity": "Активность", "projects": "Проекты", "settings": "Настройки", "stats": "Статистика"}
		page.Title = titles[name]
		if page.View == "questions" {
			page.Title = "Вопросы"
		}
	}
	var output bytes.Buffer
	if err := r.templates.ExecuteTemplate(&output, name, page); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(output.Bytes())
	return err
}

// Static returns the embedded asset handler for this renderer.
func (r *Renderer) Static() http.Handler { return Static() }

// Static serves an explicit allowlist of trusted embedded interface assets.
func Static() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/web/static/")
		name = strings.TrimPrefix(name, "/")
		mediaType, allowed := staticAssets[name]
		if !allowed {
			http.NotFound(w, r)
			return
		}
		data, err := content.ReadFile("assets/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", mediaType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	})
}

func date(value time.Time) string {
	if value.IsZero() {
		return "Не указано"
	}
	return value.Format("02.01.2006 15:04 MST")
}
func size(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d Б", value)
	}
	if value < 1024*1024 {
		return fmt.Sprintf("%.1f КиБ", float64(value)/1024)
	}
	return fmt.Sprintf("%.1f МиБ", float64(value)/(1024*1024))
}
func state(value string) string {
	names := map[string]string{"created": "Создано", "open": "Открыто", "ready": "Готово к работе", "waiting": "Ожидает", "in_progress": "В работе", "done": "Завершено", "closed": "Закрыто", "cancelled": "Отменено", "blocked": "Заблокировано", "review": "На проверке"}
	if label, ok := names[value]; ok {
		return label
	}
	return value
}
func role(value string) string {
	names := map[string]string{"human": "Человек", "coordinator": "Координатор", "analyst": "Аналитик", "architect": "Архитектор", "developer": "Разработчик", "reviewer": "Ревьюер", "tester": "Тестировщик"}
	if label, ok := names[value]; ok {
		return label
	}
	return value
}
func action(p web.Page, op string, keys ...string) string {
	if len(keys) > 0 {
		if id := p.Actions[op+":"+keys[0]]; id != "" {
			return id
		}
	}
	if id := p.Actions[op]; id != "" {
		return id
	}
	return p.ActionID
}
func generation(p web.Page) string {
	if value := p.Draft["generation"]; value != "" {
		return value
	}
	if p.Task != nil {
		return fmt.Sprint(p.Task.Generation)
	}
	return ""
}
func canClaim(t core.Task) bool {
	return t.Type == "task" && t.State == "ready" && t.Role == "human" && t.Lease == nil && !t.Blocked
}
func answer(p web.Page) string {
	if value, ok := p.Draft["answer"]; ok {
		return value
	}
	return ""
}

type draftValue struct{ Label, Value string }

func savedDraft(p web.Page) []draftValue {
	fields := []struct{ key, label string }{
		{"title", "Заголовок"}, {"description", "Описание"}, {"answer", "Ответ"}, {"reason", "Причина"},
		{"epic_key", "Эпик"}, {"role", "Роль"}, {"priority", "Приоритет"}, {"owner", "Ответственный"},
		{"areas", "Области работы"}, {"target_key", "Проверяемая задача"}, {"target_commit", "Проверяемый commit"},
		{"scope", "Область дизайна"}, {"scope_key", "Ключ области"}, {"design_id", "Материал дизайна"},
	}
	var values []draftValue
	for _, field := range fields {
		if value := p.Draft[field.key]; value != "" {
			values = append(values, draftValue{field.label, value})
		}
	}
	return values
}
func scope(value string) string {
	switch value {
	case "project":
		return "Проект"
	case "epic":
		return "Эпик"
	case "task":
		return "Задача"
	}
	return value
}
