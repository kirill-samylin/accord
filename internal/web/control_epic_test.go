package web

import (
	"net/url"
	"testing"
)

// A canonical epic must survive the browser perimeter on all existing links,
// including human questions and decision-history deep links.
func TestControlEpicFiltersPreserveExistingRoutes(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "epic-filter")
	task := f.question(t, token)
	cookie := f.login(t, token)
	epic := url.QueryEscape(task.EpicKey)
	for _, path := range []string{
		"/web/tasks?epic_key=" + epic + "&role=human&state=ready&layout=list",
		"/web/?view=tasks&epic_key=" + epic,
		"/web/questions?epic_key=" + epic,
		"/web/?view=questions&epic_key=" + epic,
		"/web/history?epic_key=" + epic,
	} {
		response := request(f.s, "GET", path, nil, cookie)
		if response.Code != 200 {
			t.Fatalf("valid epic route %s: %d", path, response.Code)
		}
		if f.response.page.EpicKey != task.EpicKey {
			t.Fatalf("epic filter lost on %s", path)
		}
		for _, got := range f.response.page.Tasks {
			if got.EpicKey != task.EpicKey {
				t.Fatalf("wrong epic on %s", path)
			}
		}
	}
	for _, invalid := range []string{"T-1", "E-0", "E-01", "E--1", "E-9223372036854775808", "E-1<script>"} {
		response := request(f.s, "GET", "/web/tasks?epic_key="+url.QueryEscape(invalid), nil, cookie)
		if response.Code != 400 {
			t.Fatalf("invalid epic %q accepted: %d", invalid, response.Code)
		}
	}
	validTask := request(f.s, "GET", "/web/activity?task_key="+url.QueryEscape(task.Key), nil, cookie)
	if validTask.Code != 200 {
		t.Fatalf("valid task filter: %d", validTask.Code)
	}
	wrongTask := request(f.s, "GET", "/web/activity?task_key="+epic, nil, cookie)
	if wrongTask.Code != 400 {
		t.Fatalf("epic accepted as task filter: %d", wrongTask.Code)
	}
}
