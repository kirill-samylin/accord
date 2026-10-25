package projects

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"agent-board/internal/core"
)

// Regressions from the logged-in UI sweep of the Accord stand (26.10.2026).

func browserMessage(t *testing.T, err error) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	fail(w, err)
	var out struct {
		Error struct{ Message string } `json:"error"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	return w.Code, out.Error.Message
}

func TestRepositoryConnectedOnlyOncePerBranch(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "repositories")
	add := func(raw, branch string) error {
		url, b, e := repositoryFields(raw, branch)
		if e != nil {
			t.Fatal(e)
		}
		return f.r.repository(ctx, f.who, p.ID, "", "add", url, b, 0)
	}
	if e := add("https://git.example.test/team/app.git", "main"); e != nil {
		t.Fatal(e)
	}
	for _, same := range []string{"https://GIT.example.test/team/app", "https://git.example.test/team/app/", "ssh://git@git.example.test/team/app.git"} {
		code, message := browserMessage(t, add(same, "main"))
		if code != 409 || message != "Этот репозиторий уже подключён" {
			t.Fatalf("%s: %d %q", same, code, message)
		}
	}
	// Another branch or another path is a different checkout target.
	if e := add("https://git.example.test/team/app.git", "release"); e != nil {
		t.Fatal(e)
	}
	if e := add("https://git.example.test/team/App", "main"); e != nil {
		t.Fatal(e)
	}
	// Updating a row onto an existing target is rejected; keeping its own is not.
	var id string
	var version int64
	if e := f.r.db.QueryRow(`SELECT id,version FROM repositories WHERE project_id=? AND branch='release'`, p.ID).Scan(&id, &version); e != nil {
		t.Fatal(e)
	}
	if _, message := browserMessage(t, f.r.repository(ctx, f.who, p.ID, id, "update", "https://git.example.test/team/app", "main", version)); message != "Этот репозиторий уже подключён" {
		t.Fatalf("update collision: %q", message)
	}
	if e := f.r.repository(ctx, f.who, p.ID, id, "update", "https://git.example.test/team/app", "release", version); e != nil {
		t.Fatalf("update of itself: %v", e)
	}
}

func TestRegrantingActiveMemberExplainsRoleChange(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "members")
	_, who, e := f.r.CreateCredential(ctx, "editor", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); e != nil {
		t.Fatal(e)
	}
	code, message := browserMessage(t, f.r.changeMember(ctx, f.who, p.ID, who.ID, "viewer", "grant", 0))
	if code != 409 || message != "Этот участник уже есть в проекте; измените его роль в списке" {
		t.Fatalf("regrant: %d %q", code, message)
	}
	// A stale version gets its own specific text as well.
	if _, message = browserMessage(t, f.r.changeMember(ctx, f.who, p.ID, who.ID, "viewer", "set", 99)); message != "Участник изменился: обновите список и повторите" {
		t.Fatalf("stale version: %q", message)
	}
}

func TestEditableRejectionKeepsOnlyFieldDecidedConflictsEditable(t *testing.T) {
	for _, c := range []struct {
		code, message string
		want          bool
	}{
		{"VALIDATION", "anything", true},
		{"CONFLICT", "membership already exists", true},
		{"CONFLICT", "Этот репозиторий уже подключён", true},
		{"CONFLICT", "member capacity reached", true},
		{"CONFLICT", "project changed", false},
		{"CONFLICT", "membership changed", false},
		{"CONFLICT", "some unknown conflict", false},
		{"NOT_FOUND", "x", false},
		{"FORBIDDEN_ROLE", "x", false},
	} {
		if got := editableRejection(&core.Error{Code: c.code, Message: c.message}); got != c.want {
			t.Errorf("%s %q: editable=%v, want %v", c.code, c.message, got, c.want)
		}
	}
}
