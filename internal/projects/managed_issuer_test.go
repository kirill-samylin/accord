package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestManagedQueuedIssuerCredentialAndEpochRevocationAcrossLegacyAuthority(t *testing.T) {
	for _, scenario := range []string{"catalog-revoke", "catalog-regrant", "legacy-revoke"} {
		t.Run(scenario, func(t *testing.T) {
			f := setup(t)
			ctx := context.Background()
			p := f.create(t, f.who, "managed")
			owner := f.describe(t, p.ID, f.token.Secret, "owner", "human")
			path := "/api/v1/p/" + p.ID + "/managed/"
			var cred core.TokenView
			var who Identity
			var err error
			if scenario == "legacy-revoke" {
				cred, err = f.store.CreateToken(ctx, core.TokenInput{Owner: "issuer", Machine: "machine", TTL: time.Hour})
				if err == nil {
					who, err = f.r.Authenticate(ctx, cred.Secret)
				}
			} else {
				cred, who, err = f.r.CreateCredential(ctx, "issuer", "machine", time.Hour)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); err != nil {
				t.Fatal(err)
			}
			issuer := f.describe(t, p.ID, cred.Secret, "issuer", "human")
			list := f.native(t, "POST", path+"list", cred.Secret, issuer.SessionID, `{}`)
			if list.Code != 200 {
				t.Fatal(list.Code, list.Body.String())
			}
			var page core.ManagedPage
			json.Unmarshal(list.Body.Bytes(), &page)
			enroll := core.ManagedHostEnrollInput{RequestID: "host", Name: "mac", HostSecret: strings.Repeat("c", 64), Controllers: []string{page.ActorPrincipalID}, Workspaces: []string{"gui"}, Providers: []string{"codex"}}
			raw, _ := json.Marshal(enroll)
			hostResponse := f.native(t, "POST", path+"hosts/enroll", f.token.Secret, owner.SessionID, string(raw))
			if hostResponse.Code != 200 {
				t.Fatal(hostResponse.Code, hostResponse.Body.String())
			}
			var host core.ManagedHost
			json.Unmarshal(hostResponse.Body.Bytes(), &host)
			profileResponse := f.native(t, "POST", "/api/v1/p/"+p.ID+"/profile_create", f.token.Secret, owner.SessionID, `{"request_id":"profile","name":"dev","description":"","role":"developer","instructions":"","provider":"unconfigured","model":""}`)
			if profileResponse.Code != 200 {
				t.Fatal(profileResponse.Code, profileResponse.Body.String())
			}
			var profile core.AgentProfile
			json.Unmarshal(profileResponse.Body.Bytes(), &profile)
			start := core.ManagedStartInput{RequestID: "start", HostID: host.ID, Workspace: "gui", Provider: "codex", ProfileID: profile.ID, ProfileVersion: 1, Prompt: "Read only"}
			raw, _ = json.Marshal(start)
			queued := f.native(t, "POST", path+"start", cred.Secret, issuer.SessionID, string(raw))
			if queued.Code != 200 {
				t.Fatal(queued.Code, queued.Body.String())
			}
			switch scenario {
			case "catalog-revoke":
				err = f.r.RevokeCredential(ctx, who.ID)
			case "legacy-revoke":
				err = f.store.RevokeToken(ctx, cred.ID)
			case "catalog-regrant":
				err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 1)
				if err == nil {
					err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			poll := core.ManagedPollInput{RequestID: "poll", HostID: host.ID, HostSecret: enroll.HostSecret, WorkerID: "owner-worker"}
			raw, _ = json.Marshal(poll)
			response := f.native(t, "POST", path+"poll", f.token.Secret, owner.SessionID, string(raw))
			if response.Code != 200 {
				t.Fatal(response.Code, response.Body.String())
			}
			var result core.ManagedPollResult
			json.Unmarshal(response.Body.Bytes(), &result)
			if result.Command != nil {
				t.Fatal("revoked/old epoch command dispatched")
			}
		})
	}
}
