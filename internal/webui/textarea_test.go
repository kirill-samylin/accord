package webui

import (
	"html"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type textareaValue struct {
	ID, Value string
}

// textareaValues models the relevant HTML RCDATA rules for our escaped SSR
// output, including the single initial LF ignored after a textarea start tag.
// It is a focused unit reader, not a general HTML parser or browser E2E.
// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody
func textareaValues(document string) []textareaValue {
	areas := regexp.MustCompile(`(?s)<textarea\b([^>]*)>(.*?)</textarea>`)
	ids := regexp.MustCompile(`\bid="([^"]*)"`)
	values := []textareaValue{}
	for _, area := range areas.FindAllStringSubmatch(document, -1) {
		id := ""
		if match := ids.FindStringSubmatch(area[1]); match != nil {
			id = match[1]
		}
		body := strings.ReplaceAll(area[2], "\r\n", "\n")
		body = strings.ReplaceAll(body, "\r", "\n")
		body = html.UnescapeString(body)
		values = append(values, textareaValue{id, strings.TrimPrefix(body, "\n")})
	}
	return values
}

func TestTextareaDraftValuesExact(t *testing.T) {
	cases := []struct{ name, value string }{
		{"empty", ""},
		{"plain", "exact retained draft"},
		{"multiline_cyrillic", "Первая строка\nВторая строка\nТретья"},
		{"html_looking", `</textarea><script>alert("текст")</script> & &#10; 'quoted'`},
		{"leading_lf", "\nНачальная строка"},
		{"multiple_leading_lf", "\n\nНачальная строка"},
		{"trailing_lf", "Последняя строка\n\n"},
		{"whitespace_only", " \t\n\t "},
		{"single_lf", "\n"},
		{"surrounding_whitespace", " \t\nСохрани пробелы\n \t"},
		{"answer_limit_16384_bytes", strings.Repeat("я", 8192)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := fixture()
			p.Draft = map[string]string{"answer": tc.value, "reason": tc.value, "description": tc.value}
			p.CanEdit = true
			assertTextareaValues(t, rendered(t, "task", p), []textareaValue{
				{"answer", tc.value}, {"release-reason", tc.value},
			})

			p.CanEdit = false
			wantReadonly := []textareaValue{}
			if tc.value != "" {
				wantReadonly = append(wantReadonly, textareaValue{"", tc.value})
			}
			assertTextareaValues(t, rendered(t, "task", p), wantReadonly)

			p.View = "tasks"
			assertTextareaValues(t, rendered(t, "board", p), []textareaValue{{"create-description", tc.value}})

			wantRecovery := []textareaValue{}
			if tc.value != "" {
				for range 3 {
					wantRecovery = append(wantRecovery, textareaValue{"", tc.value})
				}
			}
			assertTextareaValues(t, rendered(t, "error", p), wantRecovery)
		})
	}
}

func assertTextareaValues(t *testing.T, document string, want []textareaValue) {
	t.Helper()
	if got := textareaValues(document); !reflect.DeepEqual(got, want) {
		t.Fatalf("textarea values differ: got %#v; want %#v", got, want)
	}
}
