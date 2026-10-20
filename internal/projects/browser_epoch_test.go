package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type boardPage struct {
	Page struct {
		CSRF    string            `json:"csrf"`
		Actions map[string]string `json:"actions"`
		Project *Project          `json:"project"`
	}
}

func TestViewerCannotReplayCachedWriteAndOldWebActionCannotRevive(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "browser epochs")
	credential, who, e := f.r.CreateCredential(ctx, "editor", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); e != nil {
		t.Fatal(e)
	}
	cookies := f.login(t, credential.Secret)
	page := f.request("GET", "/web/p/"+p.ID+"/tasks", cookies, nil)
	var wire boardPage
	json.Unmarshal(page.Body.Bytes(), &wire)
	form := url.Values{"csrf": {wire.Page.CSRF}, "action_id": {wire.Page.Actions["create"]}, "type": {"epic"}, "title": {"once"}}
	path := "/web/p/" + p.ID + "/tasks"
	created := f.request("POST", path, cookies, form)
	if created.Code != 200 || !strings.Contains(created.Body.String(), "/web/p/"+p.ID+"/tasks/") {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "viewer", "set", 1); e != nil {
		t.Fatal(e)
	}
	denied := f.request("POST", path, cookies, form)
	if denied.Code != 403 {
		t.Fatalf("viewer replay %d %s", denied.Code, denied.Body.String())
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 2); e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); e != nil {
		t.Fatal(e)
	}
	req := f.request("GET", "/web/projects", cookies, nil)
	var catalog registryPage
	json.Unmarshal(req.Body.Bytes(), &catalog)
	switched := f.request("POST", "/web/catalog/switch", cookies, url.Values{"csrf": {catalog.Page.Registry.CSRF}, "action_id": {catalog.Page.Registry.Actions["switch:"+p.ID]}, "project_id": {p.ID}})
	if switched.Code != 200 {
		t.Fatalf("epoch switch %d %s", switched.Code, switched.Body.String())
	}
	replay := f.request("POST", path, cookies, form)
	if replay.Code < 400 {
		t.Fatalf("old web write action revived %d %s", replay.Code, replay.Body.String())
	}
}
func TestScopedValidationContainsAuthorizedProject(t *testing.T) {
	f := setup(t)
	p := f.create(t, f.who, "validation")
	cookies := f.login(t, f.token.Secret)
	catalog := f.request("GET", "/web/projects", cookies, nil)
	var global registryPage
	json.Unmarshal(catalog.Body.Bytes(), &global)
	out := f.request("POST", "/web/catalog/switch", cookies, url.Values{"csrf": {global.Page.Registry.CSRF}, "action_id": {global.Page.Registry.Actions["switch:"+p.ID]}, "project_id": {p.ID}})
	if out.Code != 200 {
		t.Fatal(out.Body.String())
	}
	page := f.request("GET", "/web/p/"+p.ID+"/tasks", cookies, nil)
	var wire boardPage
	json.Unmarshal(page.Body.Bytes(), &wire)
	failed := f.request("POST", "/web/p/"+p.ID+"/tasks", cookies, url.Values{"csrf": {wire.Page.CSRF}, "action_id": {wire.Page.Actions["create"]}, "type": {"task"}, "title": {""}})
	var result boardPage
	json.Unmarshal(failed.Body.Bytes(), &result)
	if failed.Code != 400 || result.Page.Project == nil || result.Page.Project.ID != p.ID {
		t.Fatalf("validation lost project %d %s", failed.Code, failed.Body.String())
	}
}

func TestCapturedBrowserContextCannotCommitAfterGlobalLogout(t *testing.T) {
	f := setup(t)
	cookies := f.login(t, f.token.Secret)
	request := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	b, e := f.r.browser(request)
	if e != nil {
		t.Fatal(e)
	}
	ctx := globalContext(context.Background(), b)
	v, e := f.r.runtime(f.r.defaultID)
	if e != nil {
		t.Fatal(e)
	}
	principal, e := v.service.Authenticate(ctx, f.token.Secret)
	if e != nil {
		t.Fatal(e)
	}
	session, e := v.browser.ProjectBinding(ctx, b.Hash)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.endSession(context.Background(), b); e != nil {
		t.Fatal(e)
	}
	_, e = v.service.Create(ctx, core.Actor{Principal: principal, SessionID: session.SessionID}, core.CreateInput{RequestID: "captured-before-logout", Type: "epic", Title: "must not commit"})
	var domain *core.Error
	if !errors.As(e, &domain) || domain.Code != "UNAUTHORIZED" {
		t.Fatalf("captured cookie context wrote after logout: %v", e)
	}
}
