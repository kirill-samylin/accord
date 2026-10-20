package projects

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnknownCatalogCreatePreservesSameActionAndResumesOneProject(t *testing.T) {
	f := setup(t)
	cookies := f.login(t, f.token.Secret)
	catalog := f.request("GET", "/web/projects", cookies, nil)
	var wire registryPage
	json.Unmarshal(catalog.Body.Bytes(), &wire)
	values := url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["create"]}, "name": {"resumable"}}
	blocked := filepath.Join(f.r.options.DataDir, "projects")
	if e := os.WriteFile(blocked, []byte("injected fixture obstruction"), 0600); e != nil {
		t.Fatal(e)
	}
	first := f.request("POST", "/web/catalog/projects", cookies, values)
	if first.Code != 503 || !strings.Contains(first.Body.String(), `"retry_same_action":true`) {
		t.Fatalf("unknown create discarded action %d %s", first.Code, first.Body.String())
	}
	pid := "p_" + f.r.actionProjectID(values.Get("action_id"))
	var state string
	if e := f.r.db.QueryRow(`SELECT state FROM projects WHERE id=?`, pid).Scan(&state); e != nil || state != "provisioning" {
		t.Fatal("fixture did not reserve pending project", e)
	}
	if e := os.Remove(blocked); e != nil {
		t.Fatal(e)
	}
	resumed := f.request("POST", "/web/catalog/projects", cookies, values)
	if resumed.Code != 200 {
		t.Fatalf("same create retry %d %s", resumed.Code, resumed.Body.String())
	}
	var count int
	if e := f.r.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count); e != nil || count != 2 {
		t.Fatal("same create retry duplicated or lost project", count, e)
	}
	if e := f.r.db.QueryRow(`SELECT state FROM projects WHERE id=?`, pid).Scan(&state); e != nil || state != "active" {
		t.Fatal("original pending project not resumed", e)
	}
}
func TestUnknownSwitchRetriesOriginalPendingDescribeWithoutNewSession(t *testing.T) {
	f := setup(t)
	p := f.create(t, f.who, "switch recovery")
	cookies := f.login(t, f.token.Secret)
	catalog := f.request("GET", "/web/projects", cookies, nil)
	var wire registryPage
	json.Unmarshal(catalog.Body.Bytes(), &wire)
	if _, e := f.r.db.Exec(`CREATE TRIGGER injected_session_tag_failure BEFORE INSERT ON session_epochs WHEN NEW.project_id='` + p.ID + `' BEGIN SELECT RAISE(ABORT,'fixture tag failure'); END`); e != nil {
		t.Fatal(e)
	}
	values := url.Values{"csrf": {wire.Page.Registry.CSRF}, "action_id": {wire.Page.Registry.Actions["switch:"+p.ID]}, "project_id": {p.ID}}
	first := f.request("POST", "/web/catalog/switch", cookies, values)
	if first.Code != 503 || !strings.Contains(first.Body.String(), `"retry_same_action":true`) {
		t.Fatalf("unknown switch discarded action %d %s", first.Code, first.Body.String())
	}
	runtime, e := f.r.runtime(p.ID)
	if e != nil {
		t.Fatal(e)
	}
	sessions, e := runtime.store.ListSessions(context.Background())
	if e != nil || len(sessions) != 1 {
		t.Fatal("fixture core Describe did not commit", e)
	}
	if _, e = f.r.db.Exec(`DROP TRIGGER injected_session_tag_failure`); e != nil {
		t.Fatal(e)
	}
	resumed := f.request("POST", "/web/catalog/switch", cookies, values)
	if resumed.Code != 200 {
		t.Fatalf("same switch retry %d %s", resumed.Code, resumed.Body.String())
	}
	after, e := runtime.store.ListSessions(context.Background())
	if e != nil || len(after) != 1 || after[0].SessionID != sessions[0].SessionID {
		t.Fatal("Describe recovery created another session", e)
	}
}
