package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestTeamPageSimilarDataAndDuplicateAction(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "denis")
	cookie := f.login(t, token)
	p := jsonBrowserPage(t, f, cookie, "/web/tasks")
	principal, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	actor := core.Actor{Principal: principal, SessionID: p.Session.SessionID}
	epic, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "e", Type: "epic", Title: "Panel"})
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "a", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Тёмная тема панели", Description: "x"})
	dup, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "b", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Темная тема для панели", Description: "x"})
	if err != nil || len(dup.Warnings) != 1 {
		t.Fatalf("expected warning: %+v %v", dup, err)
	}
	team := jsonBrowserPage(t, f, cookie, "/web/team")
	if team.View != "team" || team.Team == nil || team.Team.DuplicatesTotal != 1 || !team.CanCoordinate || team.Actions["coord:duplicate:"+dup.Key] == "" {
		t.Fatalf("team page: %+v", team.Team)
	}
	w := jsonBrowserRequest(f.s, "GET", "/web/data/similar?"+url.Values{"title": {"темная тема"}}.Encode(), nil, cookie)
	var similar core.SimilarPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &similar) != nil || len(similar.Similar) != 2 {
		t.Fatalf("similar data: %d %s", w.Code, w.Body)
	}
	if w = jsonBrowserRequest(f.s, "GET", "/web/data/similar?x=1&title=a", nil, cookie); w.Code != 400 {
		t.Fatalf("unknown parameter accepted: %d", w.Code)
	}
	fields := url.Values{"csrf": {team.CSRF}, "action_id": {team.Actions["coord:duplicate:"+dup.Key]}, "canonical_key": {keep.Key}}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/tasks/"+dup.Key+"/duplicate", fields, cookie, "https://accord.test"), 200)
	after := jsonBrowserPage(t, f, cookie, "/web/tasks/"+dup.Key)
	if after.Task.State != "cancelled" || after.Task.DuplicateOf != keep.Key {
		t.Fatalf("duplicate not applied: %+v", after.Task)
	}
	if again := jsonBrowserPage(t, f, cookie, "/web/team"); again.Team.DuplicatesTotal != 0 {
		t.Fatal("resolved pair still listed")
	}
}
