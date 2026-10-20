package web

import (
	"agent-board/internal/core"
	"context"
	"errors"
	"net/url"
	"strconv"
	"testing"
)

func TestCoordinationBrowserAssignmentDependenciesConflictAndRecovery(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "coordinator")
	cookie := f.login(t, token)
	p := jsonBrowserPage(t, f, cookie, "/web/profiles")
	created := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/profiles", jsonBrowserFields(p, "profile:create", profileFields()), cookie, "https://accord.test"), 200)
	profile := jsonBrowserPage(t, f, cookie, created.Redirect).Profile
	principal, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	actor := core.Actor{Principal: principal, SessionID: p.Session.SessionID}
	epic, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Coordination"})
	if err != nil {
		t.Fatal(err)
	}
	create := jsonBrowserPage(t, f, cookie, "/web/tasks")
	input := jsonBrowserFields(create, "create", url.Values{"type": {"task"}, "role": {"developer"}, "epic_key": {epic.Key}, "title": {"Assigned from browser"}, "description": {"Implement"}, "priority": {"2"}, "profile_id": {profile.ID}, "profile_version": {"1"}})
	taskResult := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks", input, cookie, "https://accord.test"), 200)
	task := jsonBrowserPage(t, f, cookie, taskResult.Redirect)
	if task.Task.Assignment == nil || task.Coordination == nil || !task.CanCoordinate || len(task.AssignmentProfiles) != 1 || task.AssignmentProfiles[0].Instructions != "" {
		t.Fatal("missing metadata or permissions")
	}
	fields := jsonBrowserFields(task, "coord:assign", url.Values{"version": {strconv.FormatInt(task.Task.Version, 10)}, "mode": {"role_queue"}, "reason": {"General role queue"}})
	wrapped := &uncertainAssignBoard{Board: f.board, fail: true}
	f.s.board = wrapped
	unknown := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/assign", fields, cookie, "https://accord.test"), 503)
	if !unknown.RetrySameAction || unknown.Actions["coord:assign"] != fields.Get("action_id") || unknown.Draft["reason"] != "General role queue" {
		t.Fatal("unknown result lost original action")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/assign", fields, cookie, "https://accord.test"), 200)
	current := jsonBrowserPage(t, f, cookie, taskResult.Redirect)
	if current.Task.Version != task.Task.Version+1 || current.Task.Assignment != nil {
		t.Fatal("lost-response retry duplicated assign")
	}
	stale := jsonBrowserFields(current, "coord:assign", url.Values{"version": {strconv.FormatInt(task.Task.Version, 10)}, "mode": {"profile"}, "profile_id": {profile.ID}, "profile_version": {"1"}, "reason": {"Keep my draft"}})
	conflict := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/assign", stale, cookie, "https://accord.test"), 409)
	if conflict.Draft["version"] != strconv.FormatInt(task.Task.Version, 10) || conflict.Draft["reason"] != "Keep my draft" || conflict.Actions["coord:assign"] == stale.Get("action_id") {
		t.Fatal("conflict overwrote draft or action")
	}
	prerequisite, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "dependency", Type: "task", EpicKey: epic.Key, Role: "human", Title: "Decision", Description: "Choose"})
	if err != nil {
		t.Fatal(err)
	}
	link := jsonBrowserFields(current, "coord:link", url.Values{"from": {prerequisite.Key}, "type": {"blocks"}, "condition": {"completed"}, "generation": {"0"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/link", link, cookie, "https://accord.test"), 200)
	blocked := jsonBrowserPage(t, f, cookie, taskResult.Redirect)
	if !blocked.Task.Blocked || len(blocked.Coordination.Blockers) != 1 {
		t.Fatal("dependency not reflected")
	}
	cancel := jsonBrowserFields(blocked, "coord:transition", url.Values{"state": {"cancelled"}, "reason": {"Pause work"}, "generation": {"0"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/transition", cancel, cookie, "https://accord.test"), 200)
	bad := jsonBrowserFields(blocked, "coord:transition", url.Values{"state": {"ready"}, "reason": {"restore"}, "evidence": {"forged"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", taskResult.Redirect+"/transition", bad, cookie, "https://accord.test"), 400)
}

type uncertainAssignBoard struct {
	Board
	fail bool
}

func (b *uncertainAssignBoard) Assign(ctx context.Context, a core.Actor, in core.AssignInput) (core.Task, error) {
	v, e := b.Board.Assign(ctx, a, in)
	if e == nil && b.fail {
		b.fail = false
		return core.Task{}, errors.New("synthetic committed lost response")
	}
	return v, e
}

func TestCoordinationBrowserCheckCreationUsesTargetVersion(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "checks")
	cookie := f.login(t, token)
	page := jsonBrowserPage(t, f, cookie, "/web/tasks")
	principal, e := f.board.Authenticate(context.Background(), token.Secret)
	if e != nil {
		t.Fatal(e)
	}
	human := core.Actor{Principal: principal, SessionID: page.Session.SessionID}
	epic, e := f.board.Create(context.Background(), human, core.CreateInput{RequestID: "check-epic", Type: "epic", Title: "Check"})
	if e != nil {
		t.Fatal(e)
	}
	task, e := f.board.Create(context.Background(), human, core.CreateInput{RequestID: "check-task", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Work", Description: "Implement"})
	if e != nil {
		t.Fatal(e)
	}
	session, e := f.board.Describe(context.Background(), principal, core.OpenSession{RequestID: "dev", Role: "developer", RunID: "dev", Machine: principal.Machine})
	if e != nil {
		t.Fatal(e)
	}
	dev := core.Actor{Principal: principal, SessionID: session.SessionID}
	claim, e := f.board.Next(context.Background(), dev, core.NextInput{RequestID: "claim", TaskKey: task.Key})
	if e != nil {
		t.Fatal(e)
	}
	target, e := f.board.Transition(context.Background(), dev, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: "done", TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &core.Evidence{Commit: "1111111111111111111111111111111111111111", RunSummary: "Passed"}})
	if e != nil {
		t.Fatal(e)
	}
	path := "/web/tasks/" + task.Key
	p := jsonBrowserPage(t, f, cookie, path)
	fields := jsonBrowserFields(p, "coord:checks", url.Values{"target_version": {strconv.FormatInt(target.Version, 10)}, "role": {"reviewer"}})
	result := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/checks", fields, cookie, "https://accord.test"), 200)
	check := jsonBrowserPage(t, f, cookie, result.Redirect)
	if check.Task.Review == nil || check.Task.Review.TargetCommit != target.Evidence.Commit {
		t.Fatal("check lost target binding")
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/checks", fields, cookie, "https://accord.test"), 200)
	qa := jsonBrowserFields(p, "coord:checks", url.Values{"target_version": {strconv.FormatInt(target.Version, 10)}, "role": {"tester"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", path+"/checks", qa, cookie, "https://accord.test"), 409)
}
