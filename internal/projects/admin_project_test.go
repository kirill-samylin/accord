package projects

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

// A global admin manages a foreign project (members, settings) without a
// membership, but cannot open or read its board until a member role exists.
func TestAdminManagesForeignProjectWithoutBoardAccess(t *testing.T) {
	f := setup(t)
	credential, owner, e := f.r.CreateCredential(context.Background(), "B", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	foreign := f.create(t, owner, "foreign")
	admin := f.makeAdmin(t)
	out := f.request("GET", "/web/catalog/projects/"+foreign.ID, admin, nil)
	var page struct {
		Page struct {
			Registry struct {
				CSRF    string            `json:"csrf"`
				Actions map[string]string `json:"actions"`
			} `json:"registry"`
			Project Project  `json:"project"`
			Members []Member `json:"members"`
		} `json:"page"`
	}
	json.Unmarshal(out.Body.Bytes(), &page)
	if out.Code != 200 || !page.Page.Project.CanManage || page.Page.Project.CanWrite || len(page.Page.Members) != 1 || page.Page.Registry.Actions["switch:"+foreign.ID] != "" || page.Page.Registry.Actions["member:grant"] == "" {
		t.Fatalf("foreign settings %d %s", out.Code, out.Body.String())
	}
	if native := f.native(t, "GET", "/api/v1/p/"+foreign.ID+"/tasks", f.token.Secret, "", ""); native.Code != 404 {
		t.Fatalf("admin read foreign board natively %d", native.Code)
	}
	if browser := f.request("GET", "/web/p/"+foreign.ID+"/", admin, nil); browser.Code != 404 {
		t.Fatalf("admin opened foreign board %d", browser.Code)
	}
	grant := f.request("POST", "/web/catalog/projects/"+foreign.ID+"/members", admin, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["member:grant"]}, "action": {"grant"}, "identity_id": {f.who.ID}, "role": {"editor"}})
	if grant.Code != 200 {
		t.Fatalf("admin self grant %d %s", grant.Code, grant.Body.String())
	}
	out = f.request("GET", "/web/catalog/projects/"+foreign.ID, admin, nil)
	json.Unmarshal(out.Body.Bytes(), &page)
	if !page.Page.Project.CanManage || !page.Page.Project.CanWrite || page.Page.Project.Role != "editor" || page.Page.Registry.Actions["switch:"+foreign.ID] == "" {
		t.Fatalf("admin editor membership %s", out.Body.String())
	}
	if _, e = f.r.Authenticate(context.Background(), credential.Secret); e != nil {
		t.Fatal(e)
	}
	member := f.login(t, credential.Secret)
	if out = f.request("GET", "/web/catalog/projects/"+f.r.defaultID, member, nil); out.Code != 404 {
		t.Fatalf("plain owner of B reached default settings %d", out.Code)
	}
}
