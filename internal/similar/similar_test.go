package similar

import "testing"

func score(a, b string) float64 {
	s, _ := Score(New(a, "", ""), New(b, "", ""))
	return s
}

func TestDuplicatesScoreHigh(t *testing.T) {
	pairs := [][2]string{
		{"Добавить авторизацию через GitHub", "Авторизация через GitHub"},
		{"Исправить падение тестов vitest в apps/desktop", "vitest падают в apps/desktop"},
		{"Add OAuth login", "Implement OAuth login flow"},
		{"Экран настроек профиля", "Экран настроек профиля пользователя"},
		{"Ревью api пагинации", "Ревью пагинации api"},
		{"Вход через GitHub OAuth", "Авторизация через GitHub"},
		{"Тёмная тема панели", "Dark theme for the panel"},
		{"Выгрузка задач в JSONL", "Экспорт задач в JSONL"},
	}
	for _, p := range pairs {
		if s := score(p[0], p[1]); s < 0.55 {
			t.Errorf("%q vs %q = %.2f, want ≥0.55", p[0], p[1], s)
		}
	}
}

func TestDifferentWorkScoresLow(t *testing.T) {
	pairs := [][2]string{
		{"Добавить авторизацию через GitHub", "Экспорт задач в JSONL"},
		{"Экран настроек профиля", "Миграция базы на schema 5"},
		{"Add OAuth login", "Fix flaky upload test"},
		{"Экран задач", "Экран агентов"},
		{"Кнопка входа в шапке", "Авторизация через GitHub"},
		{"Входящие уведомления", "Вход через GitHub"},
	}
	for _, p := range pairs {
		if s := score(p[0], p[1]); s >= 0.5 {
			t.Errorf("%q vs %q = %.2f, want <0.5", p[0], p[1], s)
		}
	}
}

func TestExternalIDAndDescription(t *testing.T) {
	if s, r := Score(New("A", "", "JIRA-1"), New("B", "", "JIRA-1")); s != 1 || r[0] != "external_id" {
		t.Fatalf("external id: %v %v", s, r)
	}
	a := New("Кнопка входа", "Нужна кнопка входа через GitHub на странице логина с переходом в OAuth", "")
	b := New("Логин", "На странице логина нужна кнопка входа через GitHub, затем OAuth переход", "")
	if s, _ := Score(a, b); s < 0.3 {
		t.Fatalf("description should raise score, got %.2f", s)
	}
	if s, _ := Score(New("", "", ""), New("", "", "")); s != 0 {
		t.Fatalf("empty docs must not match: %v", s)
	}
}

func TestDifferentNumbersAreDistinct(t *testing.T) {
	if s := score("Ревью T-12", "Ревью T-13"); s >= 0.5 {
		t.Fatalf("different targets matched: %.2f", s)
	}
	if s := score("Ревью T-12", "Ревью задачи T-12"); s < 0.6 {
		t.Fatalf("same target should match: %.2f", s)
	}
}
