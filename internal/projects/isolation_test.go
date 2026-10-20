package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCatalogIdentityHasNoDefaultAccessAndIndependentSessions(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	credential, who, e := f.r.CreateCredential(ctx, "new account", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if list, e := f.r.List(ctx, who); e != nil || len(list) != 0 {
		t.Fatalf("new identity has default membership: %#v %v", list, e)
	}
	p := f.create(t, who, "Project B")
	session := f.describe(t, p.ID, credential.Secret, "session-b", "human")
	out := f.native(t, "GET", "/api/v1/tasks", credential.Secret, session.SessionID, "")
	if out.Code != 404 {
		t.Fatalf("unscoped default accessed: %d %s", out.Code, out.Body.String())
	}
	cookies := f.login(t, credential.Secret)
	catalog := f.request("GET", "/web/projects", cookies, nil)
	if catalog.Code != 200 || strings.Contains(catalog.Body.String(), f.r.defaultID) || !strings.Contains(catalog.Body.String(), p.ID) {
		t.Fatalf("catalog leaked default: %d %s", catalog.Code, catalog.Body.String())
	}
	board := f.request("GET", "/web/p/"+p.ID+"/", cookies, nil)
	if board.Code != 200 || !strings.Contains(board.Body.String(), `"project":{"id":"`+p.ID) {
		t.Fatalf("board scope: %d %s", board.Code, board.Body.String())
	}
	defaultPage := f.request("GET", "/web/", cookies, nil)
	if defaultPage.Code != 404 {
		t.Fatalf("default browser access: %d", defaultPage.Code)
	}
	denied := f.native(t, "POST", "/mcp/p/"+p.ID, credential.Secret, "", `{}`)
	if denied.Code == 404 {
		t.Fatal("MCP scoped route unimplemented")
	}
}
func TestRevokeRegrantDoesNotReviveNativeSessionsOrDescribeReplay(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	project := f.create(t, f.who, "isolated")
	cred, who, e := f.r.CreateCredential(ctx, "member", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, project.ID, who.ID, "editor", "grant", 0); e != nil {
		t.Fatal(e)
	}
	old := f.describe(t, project.ID, cred.Secret, "same-input", "human")
	if e = f.r.changeMember(ctx, f.who, project.ID, who.ID, "", "revoke", 1); e != nil {
		t.Fatal(e)
	}
	if out := f.native(t, "GET", "/api/v1/p/"+project.ID+"/tasks", cred.Secret, old.SessionID, ""); out.Code != 404 {
		t.Fatalf("revoked read %d", out.Code)
	}
	if e = f.r.changeMember(ctx, f.who, project.ID, who.ID, "viewer", "grant", 0); e != nil {
		t.Fatal(e)
	}
	if out := f.native(t, "GET", "/api/v1/p/"+project.ID+"/tasks", cred.Secret, old.SessionID, ""); out.Code != 401 {
		t.Fatalf("old epoch revived %d %s", out.Code, out.Body.String())
	}
	fresh := f.describe(t, project.ID, cred.Secret, "same-input", "human")
	if fresh.SessionID == old.SessionID {
		t.Fatal("old Describe replay revived revoked session")
	}
	denied := f.native(t, "POST", "/api/v1/p/"+project.ID+"/session", cred.Secret, "", `{"request_id":"architect","role":"architect","run_id":"run","machine":"machine"}`)
	if denied.Code != 403 {
		t.Fatalf("viewer architect %d", denied.Code)
	}
	if out := f.native(t, "GET", "/api/v1/p/"+project.ID+"/tasks", cred.Secret, fresh.SessionID, ""); out.Code != 200 {
		t.Fatalf("viewer reads %d %s", out.Code, out.Body.String())
	}
	denied = f.native(t, "POST", "/api/v1/p/"+project.ID+"/tasks", cred.Secret, fresh.SessionID, `{"request_id":"write"}`)
	if denied.Code != 403 {
		t.Fatalf("viewer writes %d %s", denied.Code, denied.Body.String())
	}
}
func TestLegacyDefaultRevokeCannotBeBypassedOrReimported(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.r.BootstrapDefaultOwner(ctx, f.token.ID); e != nil {
		t.Fatal(e)
	}
	other, e := f.store.CreateToken(ctx, core.TokenInput{Owner: "other", Machine: "machine", TTL: time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	who, e := f.r.Authenticate(ctx, other.Secret)
	if e != nil {
		t.Fatal(e)
	}
	old := f.describe(t, "", other.Secret, "old-session", "human")
	if e = f.r.changeMember(ctx, f.who, f.r.defaultID, who.ID, "", "revoke", 1); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/api/v1/tasks", "/api/v1/p/" + f.r.defaultID + "/tasks"} {
		if out := f.native(t, "GET", path, other.Secret, old.SessionID, ""); out.Code != 404 {
			t.Fatalf("revoked default %s %d", path, out.Code)
		}
	}
	var active bool
	if e = f.r.db.QueryRow(`SELECT active FROM memberships WHERE project_id=? AND identity_id=?`, f.r.defaultID, who.ID).Scan(&active); e != nil || active {
		t.Fatal("legacy import recreated revoked membership", e)
	}
}
func TestDeepLinkRequiresExplicitSwitchAndGlobalLogoutTombstone(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "second")
	cookies := f.login(t, f.token.Secret)
	out := f.request("GET", "/web/p/"+p.ID+"/tasks?layout=board", cookies, nil)
	if out.Code != 409 || !strings.Contains(out.Body.String(), "PROJECT_SESSION_REQUIRED") {
		t.Fatalf("deep link %d %s", out.Code, out.Body.String())
	}
	var page struct {
		Page struct {
			Registry struct {
				CSRF    string            `json:"csrf"`
				Actions map[string]string `json:"actions"`
			}
		}
	}
	catalog := f.request("GET", "/web/projects", cookies, nil)
	json.Unmarshal(catalog.Body.Bytes(), &page)
	target := "/web/p/" + p.ID + "/tasks?layout=board"
	switched := f.request("POST", "/web/catalog/switch", cookies, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["switch:"+p.ID]}, "project_id": {p.ID}, "target": {target}})
	if switched.Code != 200 || !strings.Contains(switched.Body.String(), target) {
		t.Fatalf("switch %d %s", switched.Code, switched.Body.String())
	}
	if board := f.request("GET", target, cookies, nil); board.Code != 200 {
		t.Fatalf("switched read %d %s", board.Code, board.Body.String())
	}
	// The global form logs out even when no default project session is selected.
	ended := f.request("POST", "/web/logout", cookies, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["logout"]}})
	if ended.Code != 200 {
		t.Fatalf("logout %d %s", ended.Code, ended.Body.String())
	}
	for _, path := range []string{"/web/projects", target, "/web/"} {
		if replay := f.request("GET", path, cookies, nil); replay.Code != 401 {
			t.Fatalf("logout cookie replay %s %d", path, replay.Code)
		}
	}
	// Tombstones survive the absence of any global session row.
	_, e := transaction(f.r, ctx, func(tx *sql.Tx) (bool, error) { _, e := tx.Exec(`DELETE FROM sessions`); return true, e })
	if e != nil {
		t.Fatal(e)
	}
	if replay := f.request("GET", "/web/projects", cookies, nil); replay.Code != 401 {
		t.Fatal("legacy fallback resurrected cookie")
	}
}
