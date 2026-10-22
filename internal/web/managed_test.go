package web

import (
	"agent-board/internal/core"
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestManagedBrowserActionsProjectionAndCachedControllerRevocation(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "managed")
	cookie := f.login(t, token)
	r := jsonBrowserRequest(f.s, "GET", "/web/managed", nil, cookie)
	page := decodeBrowser(t, r, 200).Page
	if page.Managed == nil || page.Managed.ActorPrincipalID == "" {
		t.Fatal("missing current control ID")
	}
	if len(page.Actions) != 0 && page.Actions["managed:start"] != "" {
		t.Fatal("no host control should be offered")
	}
	principal, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	actor := core.Actor{Principal: principal, SessionID: page.Session.SessionID}
	h, err := f.board.ManagedHostEnroll(context.Background(), actor, core.ManagedHostEnrollInput{RequestID: "local-enroll", Name: "local", HostSecret: strings.Repeat("a", 64), Workspaces: []string{"gui"}, Providers: []string{"codex"}, Controllers: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := f.board.ProfileCreate(context.Background(), actor, core.ProfileCreateInput{RequestID: "profile", ProfileSettings: core.ProfileSettings{Name: "dev", Role: "developer", Provider: "unconfigured"}})
	if err != nil {
		t.Fatal(err)
	}
	page = jsonBrowserPage(t, f, cookie, "/web/managed")
	fields := jsonBrowserFields(page, "managed:start", url.Values{"host_id": {h.ID}, "workspace": {"gui"}, "provider": {"codex"}, "profile_id": {profile.ID}, "profile_version": {"1"}, "prompt": {"Read only"}})
	started := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/start", fields, cookie, "https://accord.test"), 200)
	detail := jsonBrowserPage(t, f, cookie, started.Redirect)
	if detail.Name != "managed-run" || detail.Managed == nil || detail.ManagedRun == nil || len(detail.ManagedRun.Commands) != 1 {
		t.Fatal("detail projection")
	}
	if strings.Contains(started.Redirect, "secret") {
		t.Fatal("unsafe response")
	}
	_, err = f.board.ManagedCommand(context.Background(), actor, core.ManagedCommandInput{RequestID: "native-clarify", RunID: detail.ManagedRun.Run.ID, Version: detail.ManagedRun.Run.Version, Kind: "clarify", Text: "Latest"})
	if err != nil {
		t.Fatal(err)
	}
	stale := jsonBrowserFields(detail, "managed:command", url.Values{"run_id": {detail.ManagedRun.Run.ID}, "version": {strconv.FormatInt(detail.ManagedRun.Run.Version, 10)}, "kind": {"stop"}})
	conflict := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/command", stale, cookie, "https://accord.test"), 409)
	if conflict.Page.ManagedRun == nil || conflict.Page.ManagedRun.Run.Version <= detail.ManagedRun.Run.Version || conflict.Page.Actions["managed:command"] == stale.Get("action_id") {
		t.Fatal("conflict omitted fresh authoritative run/version/capability")
	}
	invalid := jsonBrowserFields(detail, "managed:command", url.Values{"run_id": {detail.ManagedRun.Run.ID}, "version": {strconv.FormatInt(detail.ManagedRun.Run.Version, 10)}, "kind": {"stop"}, "host_secret": {"not-from-browser"}})
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/command", invalid, cookie, "https://accord.test"), 400)
	_, err = f.board.ManagedHostDisable(context.Background(), actor, core.ManagedHostDisableInput{RequestID: "local-disable", HostID: h.ID, Version: h.Version})
	if err != nil {
		t.Fatal(err)
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/start", fields, cookie, "https://accord.test"), 403)
}
func TestManagedCachedRejectedActionCannotLaterCreateAHiddenRun(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "cached-managed")
	cookie := f.login(t, token)
	page := jsonBrowserPage(t, f, cookie, "/web/managed")
	p, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	a := core.Actor{Principal: p, SessionID: page.Session.SessionID}
	h, err := f.board.ManagedHostEnroll(context.Background(), a, core.ManagedHostEnrollInput{RequestID: "host", Name: "local", HostSecret: strings.Repeat("d", 64), Workspaces: []string{"gui"}, Providers: []string{"codex"}, Controllers: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := f.board.ProfileCreate(context.Background(), a, core.ProfileCreateInput{RequestID: "profile", ProfileSettings: core.ProfileSettings{Name: "dev", Role: "developer", Provider: "unconfigured"}})
	if err != nil {
		t.Fatal(err)
	}
	values := func() url.Values {
		return url.Values{"host_id": {h.ID}, "workspace": {"gui"}, "provider": {"codex"}, "profile_id": {profile.ID}, "profile_version": {"1"}, "prompt": {"Read"}}
	}
	page = jsonBrowserPage(t, f, cookie, "/web/managed")
	first := decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/start", jsonBrowserFields(page, "managed:start", values()), cookie, "https://accord.test"), 200)
	live := jsonBrowserPage(t, f, cookie, first.Redirect)
	page = jsonBrowserPage(t, f, cookie, "/web/managed")
	rejected := jsonBrowserFields(page, "managed:start", values())
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/start", rejected, cookie, "https://accord.test"), 409)
	if _, err = f.board.ManagedCommand(context.Background(), a, core.ManagedCommandInput{RequestID: "stop", RunID: live.ManagedRun.Run.ID, Version: live.ManagedRun.Run.Version, Kind: "stop"}); err != nil {
		t.Fatal(err)
	}
	decodeBrowser(t, jsonBrowserRequest(f.s, "POST", "/web/managed/start", rejected, cookie, "https://accord.test"), 409)
	after, err := f.board.ManagedList(context.Background(), a, core.ManagedListInput{})
	if err != nil || len(after.Runs) != 1 || after.Runs[0].State != "stopped" {
		t.Fatal("cached error launched a new hidden run", err)
	}
}
