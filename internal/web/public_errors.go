package web

import (
	"agent-board/internal/core"
	"errors"
	"strings"
	"unicode"
)

// publicError is the only text of a domain error that reaches the browser.
// Native REST/MCP clients keep the store's stable English messages; the web
// layer translates them here. Russian store messages pass through unchanged,
// known English ones get a specific translation, and anything else falls back
// to a Russian explanation of its code, so English never reaches a person.
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
	case "LEASE_HELD":
		return "Задачу сейчас выполняет другой участник или агент. Обновите страницу и дождитесь освобождения"
	}
	return localizedMessage(e)
}

func localizedMessage(e *core.Error) string {
	if strings.IndexFunc(e.Message, func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0 {
		return e.Message
	}
	if message, ok := storeMessages[e.Message]; ok {
		return message
	}
	for _, p := range storePrefixes {
		if strings.HasPrefix(e.Message, p.english) {
			return p.russian
		}
	}
	if message, ok := codeMessages[e.Code]; ok {
		return message
	}
	return "Действие не выполнено. Обновите страницу и повторите"
}

// codeMessages explain a code when its English store message is not listed.
var codeMessages = map[string]string{
	"VALIDATION":           "Проверьте поля формы",
	"CONFLICT":             "Состояние изменилось. Обновите страницу и повторите действие",
	"CYCLE":                "Эта зависимость создаст цикл: задачи будут ждать друг друга",
	"LIMIT_EXCEEDED":       "Достигнут предел. Удалите лишнее и повторите",
	"CAPACITY":             "Достигнут предел. Удалите лишнее и повторите",
	"RULE_VIOLATION":       "Действие нарушает правила доски для этой задачи",
	"EVIDENCE_MISSING":     "Для завершения не хватает результата: заполните обязательные поля",
	"IDEMPOTENCY_CONFLICT": "Эта форма уже отправлена с другими данными. Обновите страницу и повторите",
	"ROLE_HELD":            "Эта роль уже занята действующей сессией. Дождитесь её завершения",
	"INVALID_TRANSITION":   "Этот переход состояния сейчас недоступен",
	"HUMAN_REQUIRED":       "Нужно решение человека",
}

// storeMessages translate exact English store messages a person can reach
// from web forms (task create/edit, links, answers, notes, lists).
var storeMessages = map[string]string{
	"a link must connect two different items": "Задача не может зависеть от самой себя",
	"dependency creates a cycle":              "Эта зависимость создаст цикл: задачи будут ждать друг друга",
	"link already exists":                     "Такая связь уже есть",
	"link does not exist":                     "Связь не найдена",
	"unsupported link type":                   "Выберите тип связи из списка",
	"unsupported link condition":              "Выберите условие связи из списка",
	"condition incompatible with source role": "Условие не подходит к роли исходной задачи: «одобрено» — для ревью, «пройдено» — для QA",
	"pilot links require tasks":               "Связывать можно только задачи",
	"duplicate_of accepts only from/to; restore a duplicate with transition to ready": "Отметка дубликата не принимает условие и поколение",
	"invalid title":                               "Введите заголовок до 200 символов без управляющих символов",
	"invalid title, description or priority":      "Проверьте заголовок (до 200 символов), описание (до 16 КиБ) и приоритет (от 0 до 100)",
	"type must be task or epic":                   "Выберите тип: задача или эпик",
	"active epic required":                        "Выберите действующий эпик",
	"task role and description required":          "Для задачи нужны роль и описание",
	"review requires developer target and commit": "Для проверки укажите задачу разработчика и полный commit SHA",
	"description too long":                        "Описание — до 16 КиБ",
	"description must be valid UTF-8 without control characters other than newline and tab": bodyTextMessage,
	"comment must be valid UTF-8 without control characters other than newline and tab":     bodyTextMessage,
	"comment must contain 1..4096 bytes":                                                    "Комментарий — от 1 до 4096 байт",
	"owner applies only to epics; a task's participant is the session holding its lease":    "Ответственного можно указать только у эпика",
	"force requires force_reason (≤1 KiB)":                                                  "Укажите причину создания, несмотря на похожие задачи (до 1 КиБ)",
	"force_reason requires force":                                                           "Причина указывается только вместе с подтверждением создания",
	"category or external_id too long or invalid UTF8":                                      "Категория или внешний ID слишком длинные или содержат недопустимые символы",
	"complexity must be low, normal or high":                                                "Выберите сложность: низкая, обычная или высокая",
	"external_id cannot be whitespace":                                                      "Внешний ID не может состоять только из пробелов",
	"area must not contain . or .. path segments":                                           "Область не может содержать сегменты «.» и «..»",
	"cancellation reason required":                                                          "Укажите причину отмены",
	"restoration reason required":                                                           "Укажите причину восстановления",
	"reopen reason required":                                                                "Укажите причину возврата в работу",
	"open epic and cancellation reason required":                                            "Отменить можно открытый эпик, указав причину",
	"only open epic may finish":                                                             "Завершить можно только открытый эпик",
	"state transition is not allowed":                                                       "Этот переход состояния сейчас недоступен",
	"invalid epic state":                                                                    "Недопустимое состояние эпика",
	"dependencies are not currently satisfied":                                              "Задача ждёт незавершённые зависимости",
	"epic contains unfinished work":                                                         "В эпике есть незавершённые задачи",
	"epic contains unsatisfied gates":                                                       "В эпике есть невыполненные условия",
	"epic requires current positive verification for each required role":                    "Эпику нужна актуальная положительная проверка каждой обязательной роли",
	"author attempt cannot review its own result":                                           "Автор не может проверять собственный результат",
	"review must use a separate server session":                                             "Проверка должна выполняться в отдельной сессии",
	"verification must reference current completed developer commit":                        "Проверка должна ссылаться на текущий завершённый commit разработчика",
	"answer required":                                                                       "Введите ответ",
	"completion evidence required":                                                          "Для завершения нужен результат",
	"release requires reason and bounded branch":                                            "Укажите причину освобождения до 4 КиБ",
	"evidence exceeds limits":                                                               "Результат превышает допустимый размер",
	"note text must be UTF8 and contain 1..2048 bytes":                                      "Заметка — от 1 до 2048 байт",
	"questions require human role":                                                          "Вопросы доступны только роли человека",
	"invalid question cursor":                                                               "Список изменился. Обновите страницу",
	"invalid question page limit":                                                           "Список изменился. Обновите страницу",
	"question cursor does not match filters":                                                "Список изменился. Обновите страницу",
	"invalid list cursor":                                                                   "Список изменился. Обновите страницу",
	"cursor does not match this list pass":                                                  "Список изменился. Обновите страницу",
	"expected form body":                                                                    "Форма повреждена. Обновите страницу",
	"form too large or malformed":                                                           "Форма слишком велика или повреждена. Сократите текст и повторите",
	"duplicate form fields":                                                                 "Форма повреждена. Обновите страницу",
	"browser action capacity reached; sign in again":                                        "Слишком много незавершённых действий. Войдите снова",
	"request_id already used with a different payload":                                      "Эта форма уже отправлена с другими данными. Обновите страницу и повторите",
	"action fields changed":                                                                 "Эта форма уже отправлена с другими данными. Обновите страницу и повторите",
	"singleton role has an active session":                                                  "Эта роль уже занята действующей сессией. Дождитесь её завершения",
}

// storePrefixes translate English messages that end with a variable part.
var storePrefixes = []struct{ english, russian string }{
	{"areas must not repeat: ", "Области не должны повторяться"},
	{"external_id already belongs to ", "Этот внешний ID уже принадлежит другой задаче"},
	{"note quota exceeded; oldest keys: ", "Достигнут предел заметок. Удалите старые заметки"},
	{"invalid state filter for ", "Неверный фильтр состояния"},
}
