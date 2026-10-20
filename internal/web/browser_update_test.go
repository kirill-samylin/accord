package web

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestBrowserJSONUpdateRequesterValidationAndExactReplay(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "author")
	task := f.question(t, token)
	cookie := f.login(t, token)
	for _, key := range []string{task.Key, task.EpicKey} {
		path := "/web/tasks/" + key
		p := jsonBrowserPage(t, f, cookie, path)
		if !p.CanUpdate || p.Actions["update"] == "" {
			t.Fatal("requester cannot update unclaimed work")
		}
		invalid := jsonBrowserFields(p, "update", url.Values{"title": {" "}, "description": {"saved draft"}, "generation": {"0"}})
		out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", invalid, cookie, "https://accord.test"), 400)
		if out.FieldErrors["title"] == "" || out.Actions["update"] == invalid.Get("action_id") || out.Draft["description"] != "saved draft" || !out.Page.CanUpdate {
			t.Fatal("update validation draft/capability")
		}
		p.Actions = out.Actions
		description := strings.Repeat("я", 8192)
		fields := jsonBrowserFields(p, "update", url.Values{"title": {"Edited " + key}, "description": {description}, "generation": {"0"}, "role": {"architect"}, "priority": {"99"}, "state": {"done"}})
		updated := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
		if updated.Redirect != path {
			t.Fatal("update redirect")
		}
		after := jsonBrowserPage(t, f, cookie, path)
		if after.Task.Title != "Edited "+key || after.Task.Description != description || after.Task.Version != p.Task.Version+1 || after.Task.Role != p.Task.Role || after.Task.State != p.Task.State || after.Task.Priority != p.Task.Priority {
			t.Fatal("update changed wrong metadata")
		}
		decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
		if jsonBrowserPage(t, f, cookie, path).Task.Version != after.Task.Version {
			t.Fatal("update replay incremented version")
		}
		fields.Set("title", "changed replay")
		if decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 409).Error.Code != "IDEMPOTENCY_CONFLICT" {
			t.Fatal("changed update replay accepted")
		}
	}
	p := jsonBrowserPage(t, f, cookie, "/web/tasks/"+task.Key)
	tooLong := jsonBrowserFields(p, "update", url.Values{"title": {"valid"}, "description": {strings.Repeat("a", 16385)}, "generation": {"0"}})
	if decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks/"+task.Key+"/update", tooLong, cookie, "https://accord.test"), 400).FieldErrors["description"] == "" {
		t.Fatal("description limit")
	}
	// Existing SSR callers use exactly the same operation and native result.
	p2 := f.page(t, cookie, "/web/tasks/"+task.Key)
	if w := f.post(cookie, "/web/tasks/"+task.Key+"/update", p2, "update", url.Values{"title": {"SSR update"}, "description": {""}, "generation": {"0"}}); w.Code != 303 {
		t.Fatalf("SSR update %d", w.Code)
	}
}

func TestBrowserJSONUpdateForeignAndLiveLeaseFencing(t *testing.T) {
	f := fixture(t)
	author := f.token(t, "author")
	task := f.question(t, author)
	cookie := f.login(t, author)
	otherToken := f.token(t, "other")
	otherTask := f.question(t, otherToken)
	otherCookie := f.login(t, otherToken)
	path := "/web/tasks/" + task.Key
	p := jsonBrowserPage(t, f, cookie, path)
	foreign := jsonBrowserPage(t, f, otherCookie, path)
	if foreign.CanUpdate || foreign.Actions["update"] != "" || foreign.Actions["update:"+task.Key] != "" {
		t.Fatal("foreign update capability")
	}
	// Operation capabilities do not replace core authorization if deliberately
	// reused for another task, and cannot steal a live lease.
	other := jsonBrowserPage(t, f, otherCookie, "/web/tasks/"+otherTask.Key)
	forged := jsonBrowserFields(other, "update", url.Values{"title": {"forbidden"}, "description": {""}, "generation": {"0"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", forged, otherCookie, "https://accord.test"), 403)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/claim", jsonBrowserFields(p, "claim", nil), cookie, "https://accord.test"), 200)
	owned := jsonBrowserPage(t, f, cookie, path)
	if !owned.CanUpdate {
		t.Fatal("own human lease update missing")
	}
	generation := strconv.FormatInt(owned.Task.Generation, 10)
	fields := jsonBrowserFields(owned, "update", url.Values{"title": {"Owned edit"}, "description": {"Own lease remains"}, "generation": {generation}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
	after := jsonBrowserPage(t, f, cookie, path)
	if after.Task.Generation != owned.Task.Generation || after.Task.Lease.SessionID != owned.Task.Lease.SessionID {
		t.Fatal("update changed lease identity")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/release", jsonBrowserFields(after, "release", url.Values{"generation": {generation}, "reason": {"handoff"}}), cookie, "https://accord.test"), 200)
	other = jsonBrowserPage(t, f, otherCookie, path)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/claim", jsonBrowserFields(other, "claim", nil), otherCookie, "https://accord.test"), 200)
	stale := jsonBrowserFields(after, "update", url.Values{"title": {"stale edit"}, "description": {"saved stale draft"}, "generation": {generation}})
	out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", stale, cookie, "https://accord.test"), 409)
	if out.Error.Code != "LEASE_LOST" || out.Page.CanUpdate || out.Actions["update"] != "" || out.Draft["description"] != "saved stale draft" {
		t.Fatal("stale update capability/fencing")
	}
	if jsonBrowserPage(t, f, cookie, path).CanUpdate {
		t.Fatal("requester got foreign live lease update")
	}
}

type uncertainUpdateBoard struct {
	Board
	fail bool
}

func (b *uncertainUpdateBoard) Update(ctx context.Context, actor core.Actor, input core.UpdateInput) (core.Task, error) {
	task, err := b.Board.Update(ctx, actor, input)
	if err == nil && b.fail {
		b.fail = false
		return core.Task{}, errors.New("private update commit outcome")
	}
	return task, err
}
func TestBrowserJSONUpdateUnknownOutcomeAndRecovery(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "update-recovery")
	task := f.question(t, token)
	cookie := f.login(t, token)
	path := "/web/tasks/" + task.Key
	p := jsonBrowserPage(t, f, cookie, path)
	f.s.board = &uncertainUpdateBoard{Board: f.board, fail: true}
	fields := jsonBrowserFields(p, "update", url.Values{"title": {"Recovered update"}, "description": {"same draft"}, "generation": {"0"}})
	out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 503)
	if !out.RetrySameAction || out.Actions["update"] != fields.Get("action_id") || out.Draft["description"] != "same draft" {
		t.Fatal("unknown update lost replay/draft")
	}
	b, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	// A new create/claim/continuation uses this same recovery routine before it
	// can advance native sessions. Update must participate in that journal.
	if err = f.s.recoverActions(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	var saved string
	if err = f.s.db.QueryRow(`SELECT result FROM web_actions WHERE id=?`, fields.Get("action_id")).Scan(&saved); err != nil || saved == "" {
		t.Fatal("update omitted from recovery")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
	after := jsonBrowserPage(t, f, cookie, path)
	if after.Task.Version != task.Version+1 {
		t.Fatal("recovery repeated update")
	}
	if _, err = f.s.db.Exec(`UPDATE web_actions SET result='' WHERE id=?`, fields.Get("action_id")); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(2 * time.Minute)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/update", fields, cookie, "https://accord.test"), 200)
}
