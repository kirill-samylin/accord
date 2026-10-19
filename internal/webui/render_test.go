package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/web"
)

func fixture() web.Page {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	text := "<script>alert(1)</script> & \"quote\""
	return web.Page{
		Title: text, CSRF: "csrf-value", Principal: core.Principal{ID: "p", Owner: text, Machine: "mac"},
		Session: core.SessionView{Project: "Accord project", HeartbeatEverySeconds: 10},
		Task:    &core.Task{Key: "T-8", Type: "task", Title: text, Description: text, Role: "human", State: "ready", Generation: 3, Requester: "request-author"},
		Actions: map[string]string{"logout": "logout-id", "continue": "continue-id", "create": "create-id", "upload": "upload-id", "claim:T-8": "claim-id", "answer:T-8": "answer-id", "release:T-8": "release-id", "heartbeat:T-8": "heartbeat-id"},
		Roles:   []string{"human", "developer", "reviewer"}, States: []string{"ready", "in_progress", "done"},
		Draft:     map[string]string{"answer": text, "description": text, "title": text, "generation": "3"},
		Designs:   []web.DesignAsset{{AssetID: "asset", DesignID: "design", Title: text, Filename: text, Version: 1, Size: 1024, CreatedAt: now, DownloadURL: "/web/designs/asset/download", UploaderOwner: text}},
		Decisions: []core.Event{{Cursor: 4, TaskKey: "T-8", At: now, Data: &core.EventData{Answer: &text, QuestionTitle: text, Owner: text, AuthorPrincipal: "author", Generation: 3}}},
	}
}

func rendered(t *testing.T, name string, p web.Page) string {
	t.Helper()
	renderer, err := New()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err = renderer.Render(recorder, 200, name, p); err != nil {
		t.Fatal(err)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatal("missing html content type")
	}
	return recorder.Body.String()
}

func TestAllPagesEscapeUntrustedContent(t *testing.T) {
	for _, name := range []string{"login", "board", "task", "designs", "history", "error"} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			p.Error = p.Title
			p.Notice = p.Title
			p.Tasks = []core.Task{*p.Task}
			p.Questions = p.Tasks
			p.Epics = p.Tasks
			html := rendered(t, name, p)
			if strings.Contains(html, "<script>alert(1)</script>") {
				t.Fatal("unescaped executable user content")
			}
			if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt;") {
				t.Fatal("user text disappeared")
			}
			if strings.Contains(html, "<iframe") || strings.Contains(html, "<object") {
				t.Fatal("active content embedded")
			}
		})
	}
}

func TestUploadCSRFFirstAndFileLast(t *testing.T) {
	html := rendered(t, "designs", fixture())
	start := strings.Index(html, `action="/web/designs" enctype="multipart/form-data"`)
	if start < 0 {
		t.Fatal("missing upload form")
	}
	form := html[start:]
	form = form[:strings.Index(form, "</form>")]
	csrf := strings.Index(form, `name="csrf"`)
	action := strings.Index(form, `name="action_id"`)
	title := strings.Index(form, `name="title"`)
	file := strings.Index(form, `name="file"`)
	if !(csrf >= 0 && csrf < action && action < title && title < file) {
		t.Fatal("multipart security fields must precede metadata and file")
	}
	if strings.Contains(form[file:], `name="`) {
		// The file's own attribute is expected; no subsequent control may follow it.
		if strings.Contains(form[file+len(`name="file"`):], `name="`) {
			t.Fatal("file must be last multipart control")
		}
	}
	if !strings.Contains(form, `value="upload-id"`) {
		t.Fatal("missing operation-specific upload id")
	}
}

func TestOwnedAnswerAndStaleDraft(t *testing.T) {
	p := fixture()
	p.Task.State = "in_progress"
	p.Task.Lease = &core.Lease{SessionID: "s", ExpiresAt: time.Now().Add(time.Minute)}
	html := rendered(t, "task", p)
	if strings.Contains(html, `action="/web/tasks/T-8/answer"`) || strings.Contains(html, "data-answer-editor") {
		t.Fatal("foreign lease exposes answer actions")
	}
	if !strings.Contains(html, "readonly") || !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("stale draft must remain copyable")
	}
	p.CanEdit = true
	html = rendered(t, "task", p)
	for _, value := range []string{`action="/web/tasks/T-8/answer"`, `value="answer-id"`, `value="release-id"`, `data-action="heartbeat-id"`, `name="generation" value="3"`} {
		if !strings.Contains(html, value) {
			t.Fatalf("missing protected owned editor field %s", value)
		}
	}
}

func TestClaimOnlyReadyHumanTask(t *testing.T) {
	p := fixture()
	delete(p.Draft, "answer")
	html := rendered(t, "task", p)
	if !strings.Contains(html, `value="claim-id"`) {
		t.Fatal("ready human question missing exact claim")
	}
	for _, kind := range []string{"developer", "waiting", "epic", "done"} {
		p = fixture()
		p.Task.Role = "human"
		p.Task.Type = "task"
		p.Task.State = "ready"
		switch kind {
		case "developer":
			p.Task.Role = kind
		case "epic":
			p.Task.Type = kind
		default:
			p.Task.State = kind
		}
		if strings.Contains(rendered(t, "task", p), `action="/web/tasks/T-8/claim"`) {
			t.Fatalf("claim exposed for %s", kind)
		}
	}
}

func TestLoginNeverEchoesDraftToken(t *testing.T) {
	p := fixture()
	p.Draft["token"] = "never-echo-secret"
	p.Draft["token_file"] = "never-echo-file"
	html := rendered(t, "login", p)
	if strings.Contains(html, "never-echo") {
		t.Fatal("login echoed token draft")
	}
	if !strings.Contains(html, `type="password"`) {
		t.Fatal("token input must obscure secret")
	}
	if !strings.Contains(html, `name="csrf"`) {
		t.Fatal("login csrf missing")
	}
}

func TestHistoryUsesEventFacts(t *testing.T) {
	p := fixture()
	p.Task.Title = "CURRENT TITLE MUST NOT APPEAR"
	p.Task.Evidence.Answer = "CURRENT ANSWER MUST NOT APPEAR"
	html := rendered(t, "history", p)
	if strings.Contains(html, "CURRENT ") {
		t.Fatal("history reconstructed from current task")
	}
	if !strings.Contains(html, `id="decision-4"`) || !strings.Contains(html, "19.10.2026") {
		t.Fatal("missing stable event reference or server date")
	}
}

func TestPaginationPreservesFilters(t *testing.T) {
	p := fixture()
	p.View = "questions"
	p.Role = "human"
	p.EpicKey = "E-5"
	p.State = "done"
	p.NextCursor = "a+b&c"
	next := nextURL(p)
	for _, part := range []string{"view=questions", "role=human", "epic_key=E-5", "state=done", "cursor=a%2Bb%26c"} {
		if !strings.Contains(next, part) {
			t.Fatalf("missing pagination filter %s", part)
		}
	}
	p.View = "designs"
	p.Draft["design_id"] = "d1"
	p.Draft["scope"] = "task"
	p.Draft["scope_key"] = "T-8"
	if !strings.HasPrefix(nextURL(p), "/web/designs?") || !strings.Contains(nextURL(p), "design_id=d1") {
		t.Fatal("wrong design pagination")
	}
}

func TestStaticOnlyKnownAssets(t *testing.T) {
	for _, name := range []string{"app.css", "app.js"} {
		recorder := httptest.NewRecorder()
		Static().ServeHTTP(recorder, httptest.NewRequest("GET", "/web/static/"+name, nil))
		if recorder.Code != 200 || recorder.Body.Len() == 0 {
			t.Fatal("embedded asset missing")
		}
	}
	for _, name := range []string{"", "../render.go", "templates/layout.html"} {
		recorder := httptest.NewRecorder()
		Static().ServeHTTP(recorder, httptest.NewRequest("GET", "/web/static/"+name, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatal("unexpected asset exposed")
		}
	}
	recorder := httptest.NewRecorder()
	Static().ServeHTTP(recorder, httptest.NewRequest("POST", "/web/static/app.js", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatal("static mutation allowed")
	}
}

func TestInvalidTemplateDoesNotWriteResponse(t *testing.T) {
	renderer, err := New()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if renderer.Render(recorder, 200, "unknown", web.Page{}) == nil || recorder.Body.Len() != 0 {
		t.Fatal("unknown view must fail before publishing")
	}
}
