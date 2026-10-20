package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestBrowserJSONMutationReplayValidationAndQuestionChain(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "mutation")
	cookie := f.login(t, token)
	p := jsonBrowserPage(t, f, cookie, "/web/tasks")
	invalid := jsonBrowserFields(p, "create", url.Values{"type": {"task"}, "title": {""}, "description": {"saved context"}})
	failed := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", invalid, cookie, "https://accord.test"), 400)
	if failed.Error.Code != "VALIDATION" || failed.Draft["description"] != "saved context" || failed.FieldErrors["title"] == "" || failed.Actions["create"] == p.Actions["create"] || failed.RetrySameAction {
		t.Fatalf("validation %+v", failed)
	}
	p.Actions = failed.Actions
	fields := jsonBrowserFields(p, "create", url.Values{"type": {"epic"}, "title": {"JSON epic"}})
	created := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), 200)
	replay := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), 200)
	if created.Redirect == "" || replay.Redirect != created.Redirect {
		t.Fatal("exact create replay")
	}
	fields.Set("title", "changed")
	if decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), 409).Error.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatal("changed create replay accepted")
	}
	epic := jsonBrowserPage(t, f, cookie, created.Redirect).Task
	p = jsonBrowserPage(t, f, cookie, "/web/tasks")
	question := jsonBrowserFields(p, "create", url.Values{"type": {"task"}, "role": {"human"}, "epic_key": {epic.Key}, "title": {"Question"}, "description": {"Context"}, "priority": {"1"}})
	redirect := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", question, cookie, "https://accord.test"), 200).Redirect
	p = jsonBrowserPage(t, f, cookie, redirect)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", redirect+"/claim", jsonBrowserFields(p, "claim", nil), cookie, "https://accord.test"), 200)
	p = jsonBrowserPage(t, f, cookie, redirect)
	if !p.CanEdit || p.Task.Lease == nil {
		t.Fatal("claim did not grant editor")
	}
	leaseExpiry := p.Task.Lease.ExpiresAt
	if !jsonBrowserPage(t, f, cookie, redirect).Task.Lease.ExpiresAt.Equal(leaseExpiry) {
		t.Fatal("JSON GET renewed lease")
	}
	generation := strconv.FormatInt(p.Task.Generation, 10)
	heartbeat := jsonBrowserFields(p, "heartbeat", url.Values{"task_key": {p.Task.Key}, "generation": {generation}})
	w := jsonBrowserRequest(f.s, "POST", "/web/heartbeat", heartbeat, cookie, "https://accord.test")
	decodeBrowser(t, w, 200)
	var beat struct {
		Session      *core.SessionView `json:"session"`
		NextActionID string            `json:"next_action_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &beat); err != nil || beat.Session == nil || len(beat.Session.MyTasks) != 1 || beat.NextActionID == "" {
		t.Fatalf("heartbeat %s %v", w.Body, err)
	}
	answer := jsonBrowserFields(p, "answer", url.Values{"generation": {generation}, "answer": {"JSON answer"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", redirect+"/answer", answer, cookie, "https://accord.test"), 200)
	// Losing the browser result after board commit must retain native replay,
	// even after the original board session/lease would have expired.
	if _, err := f.s.db.Exec(`UPDATE web_actions SET result='' WHERE id=?`, answer.Get("action_id")); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(2 * time.Minute)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", redirect+"/answer", answer, cookie, "https://accord.test"), 200)
	decisions, err := f.board.ListDecisions(t.Context(), f.native(t, token), core.DecisionsInput{Limit: 100})
	if err != nil || len(decisions.Events) != 1 || *decisions.Events[0].Data.Answer != "JSON answer" {
		t.Fatalf("duplicate/missing decision %+v %v", decisions, err)
	}
}

type uncertainJSONBoard struct {
	Board
	fail bool
}

func (b *uncertainJSONBoard) Create(ctx context.Context, actor core.Actor, input core.CreateInput) (core.Task, error) {
	task, err := b.Board.Create(ctx, actor, input)
	if err == nil && b.fail {
		b.fail = false
		return core.Task{}, errors.New("private database failure path")
	}
	return task, err
}
func TestBrowserJSONUnknownOutcomeAndExplicitContinue(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "recovery"))
	p := jsonBrowserPage(t, f, cookie, "/web/tasks")
	f.s.board = &uncertainJSONBoard{Board: f.board, fail: true}
	fields := jsonBrowserFields(p, "create", url.Values{"type": {"epic"}, "title": {"Keep my draft"}})
	w := jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test")
	out := decodeBrowser(t, w, 503)
	if !out.RetrySameAction || out.ActionID != fields.Get("action_id") || out.Actions["create"] != fields.Get("action_id") || out.Draft["title"] != "Keep my draft" || out.Error.Code != "UNAVAILABLE" {
		t.Fatalf("unknown outcome %s", w.Body)
	}
	recovered := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", fields, cookie, "https://accord.test"), 200)
	if recovered.Redirect == "" {
		t.Fatal("unknown outcome replay failed")
	}
	p = jsonBrowserPage(t, f, cookie, "/web/tasks")
	if len(p.Epics) != 1 {
		t.Fatalf("recovery duplicated epic: %+v", p.Epics)
	}
	b, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	old := b.Session.SessionID
	b.Session.ExpiresAt = time.Now().Add(-time.Second)
	if err = f.s.saveSession(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	p = jsonBrowserPage(t, f, cookie, "/web/overview")
	if !p.CanContinue || p.Session.SessionID != old || p.Overview != nil {
		t.Fatal("read continued expired board session")
	}
	fields = jsonBrowserFields(p, "continue", nil)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/session/continue", fields, cookie, "https://accord.test"), 200)
	fresh := jsonBrowserPage(t, f, cookie, "/web/overview")
	if fresh.Session.SessionID == old {
		t.Fatal("explicit continuation missing")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/session/continue", fields, cookie, "https://accord.test"), 200)
	if jsonBrowserPage(t, f, cookie, "/web/overview").Session.SessionID != fresh.Session.SessionID {
		t.Fatal("continuation replay created session")
	}
}

func TestBrowserJSONReleaseFencingAndHeartbeatError(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "owner")
	task := f.question(t, token)
	cookie := f.login(t, token)
	path := "/web/tasks/" + task.Key
	p := jsonBrowserPage(t, f, cookie, path)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/claim", jsonBrowserFields(p, "claim", nil), cookie, "https://accord.test"), 200)
	p = jsonBrowserPage(t, f, cookie, path)
	generation := strconv.FormatInt(p.Task.Generation, 10)
	release := jsonBrowserFields(p, "release", url.Values{"generation": {generation}, "reason": {"Return to queue"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/release", release, cookie, "https://accord.test"), 200)
	other := f.login(t, f.token(t, "other-owner"))
	otherPage := jsonBrowserPage(t, f, other, path)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/claim", jsonBrowserFields(otherPage, "claim", nil), other, "https://accord.test"), 200)
	answer := jsonBrowserFields(p, "answer", url.Values{"generation": {generation}, "answer": {"Saved stale answer"}})
	out := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/answer", answer, cookie, "https://accord.test"), 409)
	if out.Error.Code != "LEASE_LOST" || out.Page.CanEdit || out.Draft["answer"] != "Saved stale answer" {
		t.Fatalf("fencing %+v", out)
	}
	heartbeat := jsonBrowserFields(p, "heartbeat", url.Values{"task_key": {task.Key}, "generation": {generation}})
	out = decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/heartbeat", heartbeat, cookie, "https://accord.test"), 409)
	if out.Error.Code != "LEASE_LOST" || len(out.Actions["heartbeat"]) != 64 || out.Actions["heartbeat"] == heartbeat.Get("action_id") {
		t.Fatalf("heartbeat error %+v", out)
	}
}
