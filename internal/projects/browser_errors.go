package projects

import (
	"agent-board/internal/core"
	"strings"
	"unicode"
)

// registryMessage is the specific Russian text of a validation or conflict
// shown to a browser, or "" to keep the generic text of its status. Sign-in,
// permission and not-found answers stay generic on purpose (no existence or
// role hints). Native clients keep the stable English registry messages;
// Russian ones pass through as written.
func registryMessage(e *core.Error) string {
	switch e.Code {
	case "UNAUTHORIZED", "FORBIDDEN_ROLE", "NOT_FOUND":
		return ""
	}
	if strings.IndexFunc(e.Message, func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0 {
		return e.Message
	}
	return registryMessages[e.Message]
}

var registryMessages = map[string]string{
	"membership already exists":         "Этот участник уже есть в проекте; измените его роль в списке",
	"membership changed":                "Участник изменился: обновите список и повторите",
	"last owner must remain":            "В проекте должен остаться хотя бы один владелец",
	"member capacity reached":           "Достигнут предел участников проекта",
	"repository capacity reached":       "Достигнут предел репозиториев проекта",
	"repository changed":                "Репозиторий изменился: обновите список и повторите",
	"project capacity reached":          "Достигнут предел проектов",
	"project changed":                   "Проект изменился: обновите страницу и повторите",
	"project has active leases":         "В проекте есть задачи в работе: дождитесь их завершения",
	"last active admin must remain":     "Должен остаться хотя бы один активный администратор",
	"last admin credential must remain": "Должен остаться хотя бы один действующий токен администратора",
	"person changed":                    "Данные человека изменились: обновите страницу и повторите",
	"person is deactivated":             "Человек отключён",
	"cannot deactivate yourself":        "Нельзя отключить самого себя",
	"cannot revoke your own credential": "Нельзя отозвать собственный токен",
	"cannot relink your own credential": "Нельзя перепривязать собственный токен",
}

// staleConflicts mean the row moved under the form: its hidden version is old,
// so the browser must reload before another send. Every other recognised
// conflict (a duplicate, a limit, the last owner/admin) is decided by the
// submitted fields alone, so the form stays editable for a corrected send.
var staleConflicts = map[string]bool{
	"membership changed": true, "repository changed": true,
	"project changed": true, "person changed": true,
}

// editableRejection reports whether a browser form may be corrected and sent
// again without reloading the page.
func editableRejection(e *core.Error) bool {
	switch e.Code {
	case "VALIDATION":
		return true
	case "CONFLICT":
		return !staleConflicts[e.Message] && registryMessage(e) != ""
	}
	return false
}
