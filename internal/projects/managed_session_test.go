package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestManagedDescribeUsesHostOwnerEpochAndArchivedSnapshotWithCurrentProjectACL(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	project := f.create(t, f.who, "binding")
	cred, who, err := f.r.CreateCredential(ctx, "local-owner", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, project.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	session := f.describe(t, project.ID, cred.Secret, "management", "human")
	base := "/api/v1/p/" + project.ID + "/"
	post := func(operation string, input, out any) {
		t.Helper()
		raw, _ := json.Marshal(input)
		response := f.native(t, "POST", base+operation, cred.Secret, session.SessionID, string(raw))
		if response.Code != 200 {
			t.Fatal(operation, response.Code, response.Body.String())
		}
		if err := json.Unmarshal(response.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	var profile core.AgentProfile
	post("profile_create", core.ProfileCreateInput{RequestID: "profile", ProfileSettings: core.ProfileSettings{Name: "dev", Description: "", Role: "developer", Provider: "unconfigured", Model: "", Instructions: "Pinned owner epoch"}}, &profile)
	secret := strings.Repeat("f", 64)
	var host core.ManagedHost
	post("managed/hosts/enroll", core.ManagedHostEnrollInput{RequestID: "host", Name: "local", HostSecret: secret, Controllers: []string{}, Workspaces: []string{"gui"}, Providers: []string{"codex"}}, &host)
	if host.OwnerAccessEpoch == 0 {
		t.Fatal("fixture must cover nonlegacy project epoch")
	}
	var run core.ManagedResult
	post("managed/start", core.ManagedStartInput{RequestID: "start", HostID: host.ID, Workspace: "gui", Provider: "codex", ProfileID: profile.ID, ProfileVersion: 1, Prompt: "Read"}, &run)
	poll := core.ManagedPollInput{RequestID: "poll", HostID: host.ID, HostSecret: secret, WorkerID: "worker-binding"}
	var dispatch core.ManagedPollResult
	post("managed/poll", poll, &dispatch)
	poll.RequestID = "accepted"
	poll.WorkerEpoch = dispatch.WorkerEpoch
	var receipt core.ManagedResult
	post("managed/report", core.ManagedReportInput{ManagedPollInput: poll, CommandID: dispatch.Command.ID, DispatchID: dispatch.Command.DispatchID, RunID: run.Run.ID, Generation: 1, Phase: "accepted"}, &receipt)
	post("profile_archive", core.ProfileArchiveInput{RequestID: "archive", ProfileID: profile.ID, Version: 1, State: "archived"}, &profile)
	in := core.OpenSession{RequestID: "bound", Role: "developer", RunID: run.Run.ID, Machine: "machine", ProfileID: profile.ID, ProfileVersion: 1, ManagedRunID: run.Run.ID, ManagedGeneration: 1}
	var bound core.SessionView
	post("session", in, &bound)
	if bound.Profile == nil || bound.Profile.Version != 1 || bound.Profile.Instructions != "Pinned owner epoch" {
		t.Fatal("registry did not inject authoritative owner epoch")
	}
	raw, _ := json.Marshal(in)
	foreign := f.native(t, "POST", base+"session", f.token.Secret, "", string(raw))
	if foreign.Code != 403 {
		t.Fatal("project owner impersonated local host owner", foreign.Code, foreign.Body.String())
	}
	ordinary := in
	ordinary.RequestID = "ordinary"
	ordinary.ManagedRunID = ""
	ordinary.ManagedGeneration = 0
	raw, _ = json.Marshal(ordinary)
	if response := f.native(t, "POST", base+"session", cred.Secret, "", string(raw)); response.Code != 409 {
		t.Fatal("ordinary archive rule weakened", response.Code, response.Body.String())
	}
	if err = f.r.changeMember(ctx, f.who, project.ID, who.ID, "", "revoke", 1); err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, project.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	in.RequestID = "new-epoch"
	raw, _ = json.Marshal(in)
	if response := f.native(t, "POST", base+"session", cred.Secret, "", string(raw)); response.Code != 403 {
		t.Fatal("old host ownership survived owner epoch", response.Code, response.Body.String())
	}
}
