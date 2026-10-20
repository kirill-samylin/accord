package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProfileProjectACLIsolationEpochAndArchivedReplay(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	a := f.create(t, f.who, "A")
	b := f.create(t, f.who, "B")
	owner := f.describe(t, a.ID, f.token.Secret, "owner", "human")
	other := f.describe(t, b.ID, f.token.Secret, "other", "human")
	cred, who, err := f.r.CreateCredential(ctx, "editor", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	editor := f.describe(t, a.ID, cred.Secret, "editor", "human")
	path := "/api/v1/p/" + a.ID
	body := `{"request_id":"profile-create","name":"Same name","description":"public","role":"developer","instructions":"Immutable","provider":"unconfigured","model":""}`
	created := f.native(t, "POST", path+"/profile_create", cred.Secret, editor.SessionID, body)
	if created.Code != 200 {
		t.Fatal(created.Code, created.Body.String())
	}
	var profile core.AgentProfile
	json.Unmarshal(created.Body.Bytes(), &profile)
	if out := f.native(t, "GET", "/api/v1/p/"+b.ID+"/profile?profile_id="+profile.ID, f.token.Secret, other.SessionID, ""); out.Code != 404 {
		t.Fatal("foreign profile read", out.Code)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "viewer", "set", 1); err != nil {
		t.Fatal(err)
	}
	if out := f.native(t, "POST", path+"/profile_create", cred.Secret, editor.SessionID, body); out.Code != 403 {
		t.Fatal("viewer replay", out.Code)
	}
	if out := f.native(t, "GET", path+"/profile?profile_id="+profile.ID, cred.Secret, editor.SessionID, ""); out.Code != 200 {
		t.Fatal("viewer read", out.Code)
	}
	fake := `{"request_id":"fake-human","role":"human","run_id":"x","machine":"machine","profile_id":"` + profile.ID + `","profile_version":1}`
	if out := f.native(t, "POST", path+"/session", cred.Secret, "", fake); out.Code != 400 {
		t.Fatal("human role bypass", out.Code, out.Body.String())
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "", "revoke", 2); err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	if out := f.native(t, "GET", path+"/profile?profile_id="+profile.ID, cred.Secret, editor.SessionID, ""); out.Code != 401 {
		t.Fatal("old epoch revived", out.Code)
	}
	fresh := f.describe(t, a.ID, cred.Secret, "editor", "human")
	if fresh.SessionID == editor.SessionID {
		t.Fatal("old describe revived")
	}
	if out := f.native(t, "GET", path+"/profile?profile_id="+profile.ID, cred.Secret, fresh.SessionID, ""); out.Code != 200 {
		t.Fatal("new epoch lost data", out.Code)
	}
	if _, err = f.r.updateProject(ctx, f.who, a.ID, a.Name, "archived", a.Version); err != nil {
		t.Fatal(err)
	}
	if out := f.native(t, "POST", path+"/profile_create", f.token.Secret, owner.SessionID, strings.Replace(body, "profile-create", "archived-write", 1)); out.Code != 403 {
		t.Fatal("archived write", out.Code)
	}
	if out := f.native(t, "POST", path+"/profile_create", cred.Secret, fresh.SessionID, body); out.Code != 403 {
		t.Fatal("archived replay", out.Code)
	}
	if out := f.native(t, "GET", path+"/profile?profile_id="+profile.ID, f.token.Secret, owner.SessionID, ""); out.Code != 200 {
		t.Fatal("archived read", out.Code)
	}
}
