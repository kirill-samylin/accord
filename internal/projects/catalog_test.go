package projects

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type registryPage struct {
	Page struct {
		Registry struct {
			CSRF    string            `json:"csrf"`
			Actions map[string]string `json:"actions"`
		}
	}
}

func TestConcurrentCatalogAddIsExactlyOnceAndRejectsSensitiveURL(t *testing.T) {
	f := setup(t)
	p := f.create(t, f.who, "repositories")
	cookies := f.login(t, f.token.Secret)
	page := f.request("GET", "/web/catalog/projects/"+p.ID, cookies, nil)
	var wire registryPage
	if e := json.Unmarshal(page.Body.Bytes(), &wire); e != nil {
		t.Fatal(e)
	}
	values := url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["repository:add"]}, "url": {"https://github.com/example/project.git"}, "branch": {"main"}}
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			responses <- f.request("POST", "/web/catalog/projects/"+p.ID+"/repositories", cookies, values)
		}()
	}
	close(start)
	wg.Wait()
	close(responses)
	for out := range responses {
		if out.Code != 200 {
			t.Errorf("concurrent replay %d %s", out.Code, out.Body.String())
		}
	}
	var count int
	if e := f.r.db.QueryRow(`SELECT COUNT(*) FROM repositories WHERE project_id=?`, p.ID).Scan(&count); e != nil || count != 1 {
		t.Fatalf("duplicate repositories %d %v", count, e)
	}
	invalid := url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["repository:add"]}, "url": {"https://user:secret@github.com/example/repo?token=private"}}
	out := f.request("POST", "/web/catalog/projects/"+p.ID+"/repositories", cookies, invalid)
	if out.Code != 400 || strings.Contains(out.Body.String(), "secret") || strings.Contains(out.Body.String(), "private") {
		t.Fatalf("credential URL leaked %d %s", out.Code, out.Body.String())
	}
	var payload string
	if e := f.r.db.QueryRow(`SELECT payload FROM actions WHERE id=?`, values.Get("action_id")).Scan(&payload); e != nil || strings.Contains(payload, "secret") {
		t.Fatal("sensitive URL stored", e)
	}
}
func TestBOnlyCanLogoutAndScopedHTMLPostNeverEscapesToDefault(t *testing.T) {
	f := setup(t)
	credential, who, e := f.r.CreateCredential(context.Background(), "B only", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	p := f.create(t, who, "B")
	cookies := f.login(t, credential.Secret)
	req := httptest.NewRequest("POST", "http://127.0.0.1:8080/web/p/"+p.ID+"/tasks", strings.NewReader("type=epic&title=wrong"))
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	out := httptest.NewRecorder()
	f.handler.ServeHTTP(out, req)
	if out.Code != 406 || out.Header().Get("Location") != "" || strings.Contains(out.Body.String(), "<form") {
		t.Fatalf("HTML scope escape %d %s", out.Code, out.Body.String())
	}
	catalog := f.request("GET", "/web/projects", cookies, nil)
	var wire registryPage
	json.Unmarshal(catalog.Body.Bytes(), &wire)
	ended := f.request("POST", "/web/logout", cookies, url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["logout"]}})
	if ended.Code != 200 {
		t.Fatalf("B-only logout %d %s", ended.Code, ended.Body.String())
	}
	if replay := f.request("GET", "/web/p/"+p.ID+"/", cookies, nil); replay.Code != 401 {
		t.Fatalf("B logout replay %d", replay.Code)
	}
}
func TestNativeScopedOriginCannotBypassGlobalPerimeter(t *testing.T) {
	f := setup(t)
	for _, path := range []string{"/mcp/p/" + f.r.defaultID, "/api/v1/p/" + f.r.defaultID + "/tasks"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil)
		req.Header.Set("Origin", "http://127.0.0.1:8080")
		req.Header.Set("Authorization", "Bearer "+f.token.Secret)
		out := httptest.NewRecorder()
		f.handler.ServeHTTP(out, req)
		if out.Code != http.StatusForbidden {
			t.Fatalf("native Origin %s %d", path, out.Code)
		}
	}
}
func TestPendingCatalogActionEpochDoesNotReviveAfterRegrant(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "owner")
	credential, who, e := f.r.CreateCredential(ctx, "coowner", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "owner", "grant", 0); e != nil {
		t.Fatal(e)
	}
	cookies := f.login(t, credential.Secret)
	page := f.request("GET", "/web/catalog/projects/"+p.ID, cookies, nil)
	var wire registryPage
	json.Unmarshal(page.Body.Bytes(), &wire)
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 1); e != nil {
		t.Fatal(e)
	}
	if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "owner", "grant", 0); e != nil {
		t.Fatal(e)
	}
	out := f.request("POST", "/web/catalog/projects/"+p.ID+"/repositories", cookies, url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["repository:add"]}, "url": {"https://github.com/example/old-action.git"}})
	if out.Code != 409 {
		t.Fatalf("old unbound action revived %d %s", out.Code, out.Body.String())
	}
	var count int
	if e = f.r.db.QueryRow(`SELECT COUNT(*) FROM repositories`).Scan(&count); e != nil || count != 0 {
		t.Fatal("old action published repository", e)
	}
}

func TestExpiredRegistrySessionMetadataIsBoundedWithoutDeletingNativeEpochs(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	cookies := f.login(t, f.token.Secret)
	request := httptest.NewRequest("GET", "http://127.0.0.1:8080/web/projects", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	b, e := f.r.browser(request)
	if e != nil {
		t.Fatal(e)
	}
	expired := time.Now().Add(-time.Minute).UnixNano()
	until := time.Now().Add(time.Hour).UnixNano()
	for _, statement := range []string{`INSERT INTO sessions VALUES('expired','irrelevant','csrf',?)`, `INSERT INTO preauth VALUES('expired','csrf',?)`, `INSERT INTO session_tombstones VALUES('expired',?)`} {
		if _, e = f.r.db.Exec(statement, expired); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = f.r.db.Exec(`INSERT INTO actions(id,session_hash,operation,project_id,access_epoch,created) VALUES('old','expired','create','',0,0); INSERT INTO session_epochs VALUES('old-project','old-session','old-identity',1);`); e != nil {
		t.Fatal(e)
	}
	if _, e = f.r.db.Exec(`INSERT INTO session_tombstones VALUES('live-tombstone',?)`, until); e != nil {
		t.Fatal(e)
	}
	if _, e = f.r.issue(ctx, b, "create", ""); e != nil {
		t.Fatal(e)
	}
	for table := range map[string]bool{"sessions": true, "preauth": true, "session_tombstones": true} {
		var n int
		if e = f.r.db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE hash='expired'`).Scan(&n); e != nil || n != 0 {
			t.Fatalf("expired %s retained %d %v", table, n, e)
		}
	}
	var count int
	if e = f.r.db.QueryRow(`SELECT COUNT(*) FROM actions WHERE id='old'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("old action retained", e)
	}
	if e = f.r.db.QueryRow(`SELECT COUNT(*) FROM session_epochs WHERE session_id='old-session'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("native epoch replay mapping removed", e)
	}
	if e = f.r.db.QueryRow(`SELECT COUNT(*) FROM session_tombstones WHERE hash='live-tombstone'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("live logout tombstone removed", e)
	}
}

// T022-R2-10b: the tightened name rule applies to new names only; a project
// stored with a legacy name can still be archived and restored unchanged.
func TestLegacyProjectNameCanChangeStateButNotBeRenamedInvalid(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.r.BootstrapDefaultOwner(ctx, f.token.ID); e != nil {
		t.Fatal(e)
	}
	p := f.create(t, f.who, "legacy")
	legacy := "legacy\u0085name"
	if _, e := f.r.db.Exec(`UPDATE projects SET name=? WHERE id=?`, legacy, p.ID); e != nil {
		t.Fatal(e)
	}
	if validName(legacy) {
		t.Fatal("fixture name is valid under current rules")
	}
	archived, e := f.r.updateProject(ctx, f.who, p.ID, legacy, "archived", p.Version)
	if e != nil || archived.State != "archived" {
		t.Fatal("legacy-named project could not be archived", e)
	}
	restored, e := f.r.updateProject(ctx, f.who, p.ID, legacy, "active", archived.Version)
	if e != nil || restored.State != "active" {
		t.Fatal("legacy-named project could not be restored", e)
	}
	if _, e = f.r.updateProject(ctx, f.who, p.ID, "other\u0085name", "active", restored.Version); e == nil {
		t.Fatal("rename to an invalid name accepted")
	}
	if renamed, e := f.r.updateProject(ctx, f.who, p.ID, "clean name", "active", restored.Version); e != nil || renamed.Name != "clean name" {
		t.Fatal("rename to a valid name", e)
	}
}

func TestArchiveCapabilityAndCatalogCredentialRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.r.BootstrapDefaultOwner(ctx, f.token.ID); e != nil {
		t.Fatal(e)
	}
	p := f.create(t, f.who, "archive")
	list, e := f.r.List(ctx, f.who)
	if e != nil {
		t.Fatal(e)
	}
	for _, project := range list {
		if project.CanArchive != (project.ID != f.r.defaultID) {
			t.Fatalf("wrong archive capability %#v", project)
		}
	}
	if _, e = f.r.updateProject(ctx, f.who, f.r.defaultID, "existing-project", "archived", 1); e == nil {
		t.Fatal("default archived")
	}
	archived, e := f.r.updateProject(ctx, f.who, p.ID, p.Name, "archived", p.Version)
	if e != nil || archived.CanArchive || archived.CanWrite || !archived.CanManage {
		t.Fatalf("archived permissions %#v %v", archived, e)
	}
	credential, who, e := f.r.CreateCredential(ctx, "revoked", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	f.create(t, who, "credential-owned")
	cookies := f.login(t, credential.Secret)
	if e = f.r.RevokeCredential(ctx, who.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = f.r.Authenticate(ctx, credential.Secret); e == nil {
		t.Fatal("revoked catalog bearer accepted")
	}
	if out := f.request("GET", "/web/projects", cookies, nil); out.Code < 400 {
		t.Fatal("revoked catalog cookie accepted")
	}
}
