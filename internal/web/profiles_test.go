package web

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func profileFields() url.Values {
	return url.Values{"name": {"Developer"}, "description": {"Shared"}, "role": {"developer"}, "instructions": {"Read the specification."}, "provider": {"unconfigured"}, "model": {""}}
}
func TestProfileBrowserCRUDValidationRecoveryAndProjection(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "profiles")
	cookie := f.login(t, token)
	for _, path := range []string{"/web/profiles?%zz=x", "/web/profiles?state=active;unknown=x"} {
		decodeBrowser(t, jsonBrowserRequest(f.s, "GET", path, nil, cookie), 400)
	}
	page := jsonBrowserPage(t, f, cookie, "/web/profiles")
	if page.Name != "profiles" || !page.CanManageProfiles || page.Actions["profile:create"] == "" {
		t.Fatal("profile create capability")
	}
	invalid := jsonBrowserFields(page, "profile:create", profileFields())
	invalid.Set("name", "")
	failed := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/profiles", invalid, cookie, "https://accord.test"), 400)
	if failed.Draft["instructions"] != "Read the specification." || failed.Actions["profile:create"] == invalid.Get("action_id") || failed.RetrySameAction {
		t.Fatal("invalid draft or capability lost")
	}
	page.Actions = failed.Actions
	fields := jsonBrowserFields(page, "profile:create", profileFields())
	created := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/profiles", fields, cookie, "https://accord.test"), 200)
	detail := jsonBrowserPage(t, f, cookie, created.Redirect)
	if detail.Profile == nil || detail.Profile.CreatedBy != page.Session.SessionID || detail.Profile.Version != 1 {
		t.Fatal("profile provenance")
	}
	if strings.Contains(func() string { raw, _ := json.Marshal(detail); return string(raw) }(), token.ID) {
		t.Fatal("credential ID projected")
	}
	edit := jsonBrowserFields(detail, "profile:update", profileFields())
	edit.Set("instructions", "Changed instructions")
	edit.Set("version", "1")
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", created.Redirect+"/update", edit, cookie, "https://accord.test"), 200)
	if _, err := f.s.db.Exec(`UPDATE web_actions SET result='' WHERE id=?`, edit.Get("action_id")); err != nil {
		t.Fatal(err)
	}
	// A committed native profile edit survives a lost browser result and expiry.
	f.now = f.now.Add(2 * time.Minute)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", created.Redirect+"/update", edit, cookie, "https://accord.test"), 200)
	if _, err := f.s.db.Exec(`UPDATE web_actions SET result='' WHERE id=?`, edit.Get("action_id")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.db.Exec(`UPDATE browser_sessions SET board_session=json_set(board_session,'$.expires_at',?)`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	expired := jsonBrowserPage(t, f, cookie, created.Redirect)
	if !expired.CanContinue || expired.CanManageProfiles {
		t.Fatal("expired profile controls")
	}
	cont := jsonBrowserFields(expired, "continue", nil)
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/session/continue", cont, cookie, "https://accord.test"), 200)
	current := jsonBrowserPage(t, f, cookie, created.Redirect)
	if current.Profile.Version != 2 || current.Profile.Instructions != "Changed instructions" {
		t.Fatal("recovery duplicated update")
	}
	historical := jsonBrowserPage(t, f, cookie, created.Redirect+"?version=1")
	if historical.CanManageProfiles || historical.Profile.Instructions != "Read the specification." {
		t.Fatal("revision overwritten")
	}
	archive := jsonBrowserFields(current, "profile:archive", url.Values{"version": {strconv.FormatInt(current.Profile.Version, 10)}, "state": {"archived"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", created.Redirect+"/archive", archive, cookie, "https://accord.test"), 200)
	list := jsonBrowserPage(t, f, cookie, "/web/profiles?state=archived")
	if len(list.Profiles) != 1 || list.Profiles[0].Instructions != "" {
		t.Fatal("archive/list summary")
	}
	unknown := jsonBrowserFields(current, "profile:update", profileFields())
	unknown.Set("version", "3")
	unknown.Set("command", "unsafe")
	if got := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", created.Redirect+"/update", unknown, cookie, "https://accord.test"), 400); got.Error.Code != "VALIDATION" {
		t.Fatal("unknown form field")
	}
}
