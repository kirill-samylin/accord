package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"agent-board/internal/core"
)

// Regressions from the logged-in UI sweep of the Accord stand (26.10.2026).

func TestBrowserCreateKeepsExplicitPriorityZero(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "author")
	epic := f.question(t, token).EpicKey
	cookie := f.login(t, token)
	create := func(priority string, status int) browserEnvelope {
		p := jsonBrowserPage(t, f, cookie, "/web/")
		fields := jsonBrowserFields(p, "create", url.Values{"type": {"task"}, "epic_key": {epic}, "role": {"developer"}, "title": {"Priority " + priority}, "description": {"work"}, "priority": {priority}})
		return decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), status)
	}
	for priority, want := range map[string]int{"0": 0, "": 2, "7": 7} {
		out := create(priority, 200)
		if got := jsonBrowserPage(t, f, cookie, out.Redirect).Task.Priority; got != want {
			t.Fatalf("priority %q stored as %d, want %d", priority, got, want)
		}
	}
	for _, bad := range []string{"-1", "101", "x"} {
		if out := create(bad, 400); out.FieldErrors["priority"] != priorityMessage || out.Error.Message != priorityMessage {
			t.Fatalf("priority %q: %+v", bad, out)
		}
	}
}

func TestBrowserTextFieldsRejectControlCharactersInRussian(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "author")
	task := f.question(t, token)
	cookie := f.login(t, token)
	path := "/web/tasks/" + task.Key
	for _, description := range []string{"bell\x07", "esc\x1b[2J", "lone\rcr", "bad\xff"} {
		p := jsonBrowserPage(t, f, cookie, path)
		fields := jsonBrowserFields(p, "update", url.Values{"title": {"Edited"}, "description": {description}, "generation": {"0"}})
		out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 400)
		if out.FieldErrors["description"] != bodyTextMessage || out.FieldErrors["title"] != "" || out.Error.Message != bodyTextMessage {
			t.Fatalf("%q: %+v", description, out)
		}
	}
	p := jsonBrowserPage(t, f, cookie, path)
	fields := jsonBrowserFields(p, "update", url.Values{"title": {"Edited"}, "description": {"line\r\nnext\ttab\n"}, "generation": {"0"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
	p = jsonBrowserPage(t, f, cookie, "/web/")
	create := jsonBrowserFields(p, "create", url.Values{"type": {"task"}, "epic_key": {task.EpicKey}, "role": {"developer"}, "title": {"t\x1b"}, "description": {"ok"}})
	if out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", create, cookie, "https://accord.test"), 400); out.FieldErrors["title"] != lineTextMessage {
		t.Fatalf("create title: %+v", out)
	}
}

func TestPublicErrorNeverShowsEnglishStoreText(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{core.Fail("invalid_input", "a link must connect two different items"), "Задача не может зависеть от самой себя"},
		{core.Fail("cycle", "dependency creates a cycle"), "Эта зависимость создаст цикл: задачи будут ждать друг друга"},
		{core.Fail("invalid_input", "areas must not repeat: web"), "Области не должны повторяться"},
		{core.Fail("invalid_input", "some new English text"), "Проверьте поля формы"},
		{core.Fail("conflict", "anything"), codeMessages["CONFLICT"]},
		{core.Fail("limit_exceeded", "note quota exceeded; oldest keys: T-1"), "Достигнут предел заметок. Удалите старые заметки"},
		{core.Fail("rule_violation", "unknown rule"), codeMessages["RULE_VIOLATION"]},
		{core.Fail("evidence_missing", "findings required"), codeMessages["EVIDENCE_MISSING"]},
		{core.Fail("idempotency_conflict", "request_id already used with a different payload"), codeMessages["IDEMPOTENCY_CONFLICT"]},
		{core.Fail("role_held", "singleton role has an active session"), codeMessages["ROLE_HELD"]},
		{core.Fail("brand_new_code", "english"), "Действие не выполнено. Обновите страницу и повторите"},
		{core.Fail("conflict", "Вопрос уже завершён"), "Вопрос уже завершён"},
	} {
		if got := publicError(c.err); got != c.want {
			t.Fatalf("%v: got %q want %q", c.err, got, c.want)
		}
	}
	for code, message := range codeMessages {
		if strings.IndexFunc(message, func(r rune) bool { return r >= 'a' && r <= 'z' }) >= 0 {
			t.Fatalf("%s fallback is not Russian: %q", code, message)
		}
	}
}

// Two login tabs share one cookie jar: the second GET reuses the live preauth
// so the first form stays valid; a consumed preauth is never reused.
func TestSecondLoginTabKeepsFirstFormValid(t *testing.T) {
	f := fixture(t)
	first := jsonBrowserRequest(f.s, "GET", "/web/login", nil, nil)
	csrf := decodeBrowser(t, first, 200).Page.CSRF
	pre := first.Result().Cookies()[0]
	second := jsonBrowserRequest(f.s, "GET", "/web/login", nil, pre)
	if got := decodeBrowser(t, second, 200).Page.CSRF; got != csrf {
		t.Fatal("second tab rotated the login form")
	}
	if renewed := second.Result().Cookies(); len(renewed) != 1 || renewed[0].Value != pre.Value {
		t.Fatal("preauth cookie not renewed in place")
	}
	token := f.token(t, "owner")
	out := jsonBrowserRequest(f.s, "POST", "/web/login", url.Values{"csrf": {csrf}, "token": {token.Secret}}, pre, "https://accord.test")
	if out.Code != http.StatusOK {
		t.Fatalf("login from the first tab: %d %s", out.Code, out.Body)
	}
	again := jsonBrowserRequest(f.s, "GET", "/web/login", nil, pre)
	if got := decodeBrowser(t, again, 200).Page.CSRF; got == csrf {
		t.Fatal("consumed preauth was reused")
	}
}
