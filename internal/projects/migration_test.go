package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/transport"
	"agent-board/internal/web"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestSchema1BrowserImportPreservesHashCSRFSessionAndLegacyDescribe(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	principal, e := f.store.Authenticate(ctx, f.token.Secret)
	if e != nil {
		t.Fatal(e)
	}
	input := core.OpenSession{RequestID: "legacy-before-upgrade", Role: "human", RunID: "run", Machine: "machine"}
	session, e := f.store.Describe(ctx, principal, input)
	if e != nil {
		t.Fatal(e)
	}
	actor := core.Actor{Principal: principal, SessionID: session.SessionID}
	epicInput := core.CreateInput{RequestID: "legacy-epic", Type: "epic", Title: "preserved epic"}
	epic, e := f.store.Create(ctx, actor, epicInput)
	if e != nil {
		t.Fatal(e)
	}
	leased, e := f.store.Create(ctx, actor, core.CreateInput{RequestID: "leased-task", Type: "task", EpicKey: epic.Key, Role: "human", Title: "leased", Description: "preserved context"})
	if e != nil {
		t.Fatal(e)
	}
	claim, e := f.store.Next(ctx, actor, core.NextInput{RequestID: "legacy-claim", TaskKey: leased.Key})
	if e != nil || claim.Task == nil {
		t.Fatal("seed lease", e)
	}
	leased = *claim.Task
	doneSession, e := f.store.Describe(ctx, principal, core.OpenSession{RequestID: "done-session", Role: "human", RunID: "done-run", Machine: "machine"})
	if e != nil {
		t.Fatal(e)
	}
	doneActor := core.Actor{Principal: principal, SessionID: doneSession.SessionID}
	done, e := f.store.Create(ctx, doneActor, core.CreateInput{RequestID: "done-task", Type: "task", EpicKey: epic.Key, Role: "human", Title: "evidence", Description: "completed"})
	if e != nil {
		t.Fatal(e)
	}
	doneClaim, e := f.store.Next(ctx, doneActor, core.NextInput{RequestID: "done-claim", TaskKey: done.Key})
	if e != nil || doneClaim.Task == nil {
		t.Fatal(e)
	}
	_, e = f.store.Transition(ctx, doneActor, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: "done-evidence", TaskKey: done.Key, Generation: doneClaim.Task.Generation}, State: "done", Evidence: &core.Evidence{Answer: "preserved answer"}})
	if e != nil {
		t.Fatal(e)
	}
	secret, _ := random("", 32)
	csrf, _ := random("", 32)
	raw, _ := json.Marshal(session)
	f.r.Close()
	f.browser.Close()
	db, e := sql.Open("sqlite", filepath.Join(f.webDir, "web.db"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`DROP TABLE web_action_epochs; PRAGMA user_version=1;`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO browser_sessions VALUES(?,?,?,?,?,'')`, hash(secret), f.token.ID, csrf, time.Now().Add(time.Hour).UnixNano(), string(raw)); e != nil {
		t.Fatal(e)
	}
	db.Close()
	options := f.r.options
	browser, e := web.New(f.store, web.Options{Origin: options.Origin, DataDir: f.webDir, Renderer: options.Renderer})
	if e != nil {
		t.Fatal(e)
	}
	defer browser.Close()
	options.Browser = browser
	r, e := Open(options)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	f.r = r
	f.browser = browser
	f.handler = transport.NewHandlerWithOptions(f.store, r.Ready, transport.HandlerOptions{ProjectRouter: r})
	before, e := f.store.ListSessions(ctx)
	if e != nil {
		t.Fatal(e)
	}
	cookies := []*http.Cookie{{Name: r.cookieName(), Value: secret}}
	page := f.request("GET", "/web/", cookies, nil)
	if page.Code != 200 {
		t.Fatalf("import page %d %s", page.Code, page.Body.String())
	}
	var wire boardPage
	json.Unmarshal(page.Body.Bytes(), &wire)
	if wire.Page.CSRF != csrf {
		t.Fatal("legacy CSRF changed")
	}
	binding, e := browser.ProjectBinding(ctx, hash(secret))
	if e != nil || binding.SessionID != session.SessionID {
		t.Fatal("legacy binding changed", e)
	}
	after, e := f.store.ListSessions(ctx)
	if e != nil || len(after) != len(before) {
		t.Fatal("GET imported browser described a new session", e)
	}
	replay := f.describe(t, "", f.token.Secret, input.RequestID, "human")
	if replay.SessionID != session.SessionID {
		t.Fatal("legacy Describe request ID no longer compatible")
	}
	leaseRead := f.native(t, "GET", "/api/v1/tasks/"+leased.Key, f.token.Secret, session.SessionID, "")
	var leaseAfter core.Task
	json.Unmarshal(leaseRead.Body.Bytes(), &leaseAfter)
	if leaseRead.Code != 200 || leaseAfter.Lease == nil || leaseAfter.Generation != leased.Generation || !leaseAfter.Lease.ExpiresAt.Equal(leased.Lease.ExpiresAt) {
		t.Fatalf("migration altered lease %d %s", leaseRead.Code, leaseRead.Body.String())
	}
	evidenceRead := f.native(t, "GET", "/api/v1/tasks/"+done.Key, f.token.Secret, session.SessionID, "")
	var evidenceAfter core.Task
	json.Unmarshal(evidenceRead.Body.Bytes(), &evidenceAfter)
	if evidenceRead.Code != 200 || evidenceAfter.Evidence.Answer != "preserved answer" {
		t.Fatalf("migration lost evidence %d %s", evidenceRead.Code, evidenceRead.Body.String())
	}
	replayBody, _ := json.Marshal(epicInput)
	cached := f.native(t, "POST", "/api/v1/tasks", f.token.Secret, session.SessionID, string(replayBody))
	var cachedEpic core.Task
	json.Unmarshal(cached.Body.Bytes(), &cachedEpic)
	if cached.Code != 200 || cachedEpic.Key != epic.Key {
		t.Fatalf("legacy cached mutation lost %d %s", cached.Code, cached.Body.String())
	}
	db, e = sql.Open("sqlite", filepath.Join(f.webDir, "web.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var version int
	if e = db.QueryRow(`PRAGMA user_version`).Scan(&version); e != nil || version != 2 {
		t.Fatal("web schema not upgraded to 2", e)
	}
}
