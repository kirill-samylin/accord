package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestManagedRegistryMembershipEpochHostProofAndScopedIsolation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "managed")
	other := f.create(t, f.who, "other")
	owner := f.describe(t, p.ID, f.token.Secret, "owner", "human")
	foreign := f.describe(t, other.ID, f.token.Secret, "foreign", "human")
	path := "/api/v1/p/" + p.ID + "/managed/"
	cred, who, err := f.r.CreateCredential(ctx, "host-owner", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	worker := f.describe(t, p.ID, cred.Secret, "worker", "human")
	list := f.native(t, "POST", path+"list", f.token.Secret, owner.SessionID, `{}`)
	if list.Code != 200 {
		t.Fatal(list.Code, list.Body.String())
	}
	var page core.ManagedPage
	json.Unmarshal(list.Body.Bytes(), &page)
	enroll := core.ManagedHostEnrollInput{RequestID: "enroll", Name: "Windows", HostSecret: strings.Repeat("a", 64), Controllers: []string{page.ActorPrincipalID}, Workspaces: []string{"gui"}, Providers: []string{"claude"}}
	raw, _ := json.Marshal(enroll)
	created := f.native(t, "POST", path+"hosts/enroll", cred.Secret, worker.SessionID, string(raw))
	if created.Code != 200 {
		t.Fatal(created.Code, created.Body.String())
	}
	var host core.ManagedHost
	json.Unmarshal(created.Body.Bytes(), &host)
	if host.OwnerPrincipalID == page.ActorPrincipalID || strings.Contains(created.Body.String(), enroll.HostSecret) {
		t.Fatal("identity merged or secret exposed")
	}
	poll := core.ManagedPollInput{RequestID: "poll", HostID: host.ID, HostSecret: enroll.HostSecret, WorkerID: "native-worker"}
	raw, _ = json.Marshal(poll)
	out := f.native(t, "POST", path+"poll", cred.Secret, worker.SessionID, string(raw))
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	foreignRead := f.native(t, "POST", "/api/v1/p/"+other.ID+"/managed/poll", f.token.Secret, foreign.SessionID, string(raw))
	if foreignRead.Code != 404 {
		t.Fatal("foreign host", foreignRead.Code)
	}
	poll.HostSecret = strings.Repeat("0", 64)
	bad, _ := json.Marshal(poll)
	if out = f.native(t, "POST", path+"poll", cred.Secret, worker.SessionID, string(bad)); out.Code != 401 {
		t.Fatal("proof bypass replay", out.Code, out.Body.String())
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "viewer", "set", 1); err != nil {
		t.Fatal(err)
	}
	if out = f.native(t, "POST", path+"poll", cred.Secret, worker.SessionID, string(raw)); out.Code != 403 {
		t.Fatal("viewer worker", out.Code, out.Body.String())
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 2); err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	fresh := f.describe(t, p.ID, cred.Secret, "worker-fresh", "human")
	if out = f.native(t, "POST", path+"poll", cred.Secret, fresh.SessionID, string(raw)); out.Code != 401 {
		t.Fatal("old host survived owner epoch", out.Code, out.Body.String())
	}
}
