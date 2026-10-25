package projects

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type adminWire struct {
	OK   bool `json:"ok"`
	Page struct {
		Name     string `json:"name"`
		Registry struct {
			Identity struct {
				ID    string `json:"id"`
				Admin bool   `json:"admin"`
			} `json:"identity"`
			CSRF    string            `json:"csrf"`
			Actions map[string]string `json:"actions"`
		} `json:"registry"`
		People     []Person        `json:"people"`
		Identities []AdminIdentity `json:"identities"`
		Audit      []AdminAudit    `json:"audit"`
	} `json:"page"`
	Person   Person            `json:"person"`
	Issued   map[string]any    `json:"issued"`
	Identity AdminIdentity     `json:"identity"`
	Error    map[string]string `json:"error"`
}

func (f *fixture) admin(t *testing.T, cookies []*http.Cookie) adminWire {
	t.Helper()
	out := f.request("GET", "/web/catalog/admin", cookies, nil)
	if out.Code != 200 {
		t.Fatalf("admin page %d %s", out.Code, out.Body.String())
	}
	var wire adminWire
	if e := json.Unmarshal(out.Body.Bytes(), &wire); e != nil {
		t.Fatal(e)
	}
	return wire
}
func (f *fixture) post(t *testing.T, cookies []*http.Cookie, page adminWire, key, path string, values url.Values) (int, adminWire) {
	t.Helper()
	values.Set("csrf", page.Page.Registry.CSRF)
	values.Set("action_id", page.Page.Registry.Actions[key])
	out := f.request("POST", path, cookies, values)
	var wire adminWire
	json.Unmarshal(out.Body.Bytes(), &wire)
	return out.Code, wire
}
func (f *fixture) makeAdmin(t *testing.T) []*http.Cookie {
	t.Helper()
	if _, e := f.r.BootstrapAdmin(context.Background(), f.who.ID); e != nil {
		t.Fatal(e)
	}
	return f.login(t, f.token.Secret)
}
func TestAdminPageIsHiddenFromMembersAndFlaggedForAdmins(t *testing.T) {
	f := setup(t)
	credential, _, e := f.r.CreateCredential(context.Background(), "B", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	member := f.login(t, credential.Secret)
	if out := f.request("GET", "/web/catalog/admin", member, nil); out.Code != 404 {
		t.Fatalf("member admin page %d", out.Code)
	}
	var catalog adminWire
	json.Unmarshal(f.request("GET", "/web/projects", member, nil).Body.Bytes(), &catalog)
	if catalog.Page.Registry.Identity.Admin {
		t.Fatal("member flagged admin")
	}
	if out := f.request("POST", "/web/catalog/admin/people", member, url.Values{"csrf": {catalog.Page.Registry.CSRF}, "action_id": {"x"}, "name": {"n"}, "role": {"member"}}); out.Code != 404 {
		t.Fatalf("member admin action %d %s", out.Code, out.Body.String())
	}
	admin := f.makeAdmin(t)
	json.Unmarshal(f.request("GET", "/web/projects", admin, nil).Body.Bytes(), &catalog)
	page := f.admin(t, admin)
	if !catalog.Page.Registry.Identity.Admin || page.Page.Name != "admin" || !page.Page.Registry.Identity.Admin || page.Page.Registry.Actions["admin:person:create"] == "" || len(page.Page.People) != 2 || len(page.Page.Identities) != 2 {
		t.Fatalf("admin page %s", f.request("GET", "/web/catalog/admin", admin, nil).Body.String())
	}
	if len(page.Page.Audit) == 0 || page.Page.Audit[0].Event != "admin_bootstrapped" {
		t.Fatalf("audit newest first %v", page.Page.Audit)
	}
}
func TestAdminPersonLifecycleAndOneTimeSecret(t *testing.T) {
	f := setup(t)
	admin := f.makeAdmin(t)
	page := f.admin(t, admin)
	code, created := f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Новый"}, "note": {"note"}, "role": {"member"}})
	if code != 200 || created.Person.ID == "" || created.Person.Version != 1 || !created.Person.Active {
		t.Fatalf("create person %d %+v", code, created)
	}
	if again, replay := f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Новый"}, "note": {"note"}, "role": {"member"}}); again != 200 || replay.Person.ID != created.Person.ID {
		t.Fatalf("create replay %d %+v", again, replay)
	}
	if code, _ = f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Другой"}, "role": {"member"}, "extra": {"x"}}); code != 400 {
		t.Fatalf("unknown field accepted %d", code)
	}
	pid := created.Person.ID
	page = f.admin(t, admin)
	path := "/web/catalog/admin/people/" + pid
	if code, _ = f.post(t, admin, page, "admin:person:"+pid+":update", path, url.Values{"action": {"update"}, "name": {"Имя"}, "note": {""}, "role": {"member"}, "version": {"7"}}); code != 409 {
		t.Fatalf("stale version %d", code)
	}
	page = f.admin(t, admin)
	code, updated := f.post(t, admin, page, "admin:person:"+pid+":update", path, url.Values{"action": {"update"}, "name": {"Имя"}, "note": {"n2"}, "role": {"member"}, "version": {"1"}})
	if code != 200 || updated.Person.Name != "Имя" || updated.Person.Version != 2 {
		t.Fatalf("update %d %+v", code, updated)
	}
	page = f.admin(t, admin)
	out := f.request("POST", path+"/credentials", admin, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["admin:person:"+pid+":credential"]}, "machine": {"laptop"}, "days": {"30"}})
	var issued adminWire
	json.Unmarshal(out.Body.Bytes(), &issued)
	secret, _ := issued.Issued["secret"].(string)
	if out.Code != 200 || !strings.HasPrefix(secret, "ac_") || issued.Issued["owner"] != "Имя" || issued.Issued["secret_delivered"] != nil || out.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("issue %d %s", out.Code, out.Body.String())
	}
	var stored string
	f.r.db.QueryRow(`SELECT result FROM actions WHERE id=?`, page.Page.Registry.Actions["admin:person:"+pid+":credential"]).Scan(&stored)
	if strings.Contains(stored, secret) || !strings.Contains(stored, `"secret_delivered":true`) {
		t.Fatalf("stored result %s", stored)
	}
	replay := f.request("POST", path+"/credentials", admin, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["admin:person:"+pid+":credential"]}, "machine": {"laptop"}, "days": {"30"}})
	if replay.Code != 200 || strings.Contains(replay.Body.String(), secret) || !strings.Contains(replay.Body.String(), `"secret_delivered":true`) {
		t.Fatalf("replay leaked secret %d %s", replay.Code, replay.Body.String())
	}
	var leaked int
	f.r.db.QueryRow(`SELECT COUNT(*) FROM admin_audit WHERE details LIKE '%' || ? || '%'`, secret).Scan(&leaked)
	if leaked != 0 {
		t.Fatal("secret in audit")
	}
	identityID, _ := issued.Issued["identity_id"].(string)
	person := f.login(t, secret)
	if out = f.request("GET", "/web/projects", person, nil); out.Code != 200 {
		t.Fatalf("new credential login %d", out.Code)
	}
	page = f.admin(t, admin)
	if code, _ = f.post(t, admin, page, "admin:person:"+pid+":update", path, url.Values{"action": {"deactivate"}, "version": {"2"}}); code != 200 {
		t.Fatalf("deactivate %d", code)
	}
	if out = f.request("GET", "/web/projects", person, nil); out.Code != 403 {
		t.Fatalf("inactive person session accepted %d", out.Code)
	}
	if _, e := f.r.Authenticate(context.Background(), secret); e == nil {
		t.Fatal("inactive person native credential accepted")
	}
	page = f.admin(t, admin)
	if code, _ = f.post(t, admin, page, "admin:person:"+pid+":update", path, url.Values{"action": {"activate"}}); code != 200 {
		t.Fatalf("activate %d", code)
	}
	if out = f.request("GET", "/web/projects", person, nil); out.Code != 200 {
		t.Fatalf("reactivated person rejected %d", out.Code)
	}
	page = f.admin(t, admin)
	code, revoked := f.post(t, admin, page, "admin:identity:"+identityID+":revoke", "/web/catalog/admin/identities/"+identityID, url.Values{"action": {"revoke"}})
	if code != 200 || !revoked.Identity.Revoked {
		t.Fatalf("revoke %d %+v", code, revoked)
	}
	if out = f.request("GET", "/web/projects", person, nil); out.Code == 200 {
		t.Fatal("revoked credential session accepted")
	}
	var events []string
	rows, _ := f.r.db.Query(`SELECT event FROM admin_audit ORDER BY id`)
	for rows.Next() {
		var event string
		rows.Scan(&event)
		events = append(events, event)
	}
	rows.Close()
	if strings.Join(events, ",") != "admin_bootstrapped,person_created,person_updated,credential_issued,person_deactivated,person_activated,credential_revoked" {
		t.Fatalf("audit trail %v", events)
	}
}
func TestAdminLinkAndLastAdminProtection(t *testing.T) {
	f := setup(t)
	credential, who, e := f.r.CreateCredential(context.Background(), "B", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	admin := f.makeAdmin(t)
	page := f.admin(t, admin)
	var self Person
	for _, p := range page.Page.People {
		if p.Role == "admin" {
			self = p
		}
	}
	selfPath := "/web/catalog/admin/people/" + self.ID
	if code, _ := f.post(t, admin, page, "admin:person:"+self.ID+":update", selfPath, url.Values{"action": {"deactivate"}}); code != 403 {
		t.Fatalf("self deactivation %d", code)
	}
	page = f.admin(t, admin)
	if code, _ := f.post(t, admin, page, "admin:person:"+self.ID+":update", selfPath, url.Values{"action": {"update"}, "name": {self.Name}, "note": {""}, "role": {"member"}, "version": {strconv.FormatInt(self.Version, 10)}}); code != 409 {
		t.Fatalf("last admin demotion %d", code)
	}
	page = f.admin(t, admin)
	code, second := f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Второй"}, "role": {"admin"}})
	if code != 200 {
		t.Fatal("second admin", code)
	}
	page = f.admin(t, admin)
	code, linked := f.post(t, admin, page, "admin:identity:"+who.ID+":link", "/web/catalog/admin/identities/"+who.ID, url.Values{"action": {"link"}, "person_id": {second.Person.ID}})
	if code != 200 || linked.Identity.PersonID != second.Person.ID {
		t.Fatalf("link %d %+v", code, linked)
	}
	if current, e := f.r.Authenticate(context.Background(), credential.Secret); e != nil || !current.Admin {
		t.Fatal("linked identity must inherit admin role", e)
	}
	page = f.admin(t, admin)
	if code, _ = f.post(t, admin, page, "admin:person:"+self.ID+":update", selfPath, url.Values{"action": {"update"}, "name": {self.Name}, "note": {""}, "role": {"member"}, "version": {strconv.FormatInt(self.Version, 10)}}); code != 200 {
		t.Fatalf("demotion with another admin %d", code)
	}
	if out := f.request("GET", "/web/catalog/admin", admin, nil); out.Code != 404 {
		t.Fatalf("demoted admin still sees page %d", out.Code)
	}
}
