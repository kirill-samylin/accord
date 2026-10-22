package bridge

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestManagedBridgeInjectsFixedAttemptAndBindsArchivedSnapshot(t *testing.T) {
	f := setupConfigured(t, 0, core.Config{SessionTTL: time.Minute, LeaseTTL: time.Minute})
	ctx := context.Background()
	p, err := f.store.Authenticate(ctx, f.token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	human, err := f.store.Describe(ctx, p, core.OpenSession{RequestID: fmtID(), Role: "human", RunID: "management", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	a := core.Actor{Principal: p, SessionID: human.SessionID}
	profile, err := f.store.ProfileCreate(ctx, a, core.ProfileCreateInput{RequestID: fmtID(), ProfileSettings: core.ProfileSettings{Name: "managed", Role: "developer", Provider: "unconfigured", Instructions: "Pinned original"}})
	if err != nil {
		t.Fatal(err)
	}
	secret := strings.Repeat("e", 64)
	host, err := f.store.ManagedHostEnroll(ctx, a, core.ManagedHostEnrollInput{RequestID: fmtID(), Name: "managed", HostSecret: secret, Controllers: []string{}, Workspaces: []string{"gui"}, Providers: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.store.ManagedStart(ctx, a, core.ManagedStartInput{RequestID: fmtID(), HostID: host.ID, Provider: "codex", Workspace: "gui", ProfileID: profile.ID, ProfileVersion: 1, Prompt: "Read"})
	if err != nil {
		t.Fatal(err)
	}
	poll := core.ManagedPollInput{RequestID: fmtID(), HostID: host.ID, HostSecret: secret, WorkerID: "managed-test"}
	dispatch, err := f.store.ManagedPoll(ctx, a, poll)
	if err != nil {
		t.Fatal(err)
	}
	poll.RequestID = fmtID()
	poll.WorkerEpoch = dispatch.WorkerEpoch
	_, err = f.store.ManagedReport(ctx, a, core.ManagedReportInput{ManagedPollInput: poll, CommandID: dispatch.Command.ID, DispatchID: dispatch.Command.DispatchID, RunID: created.Run.ID, Generation: 1, Phase: "accepted"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.ProfileArchive(ctx, a, core.ProfileArchiveInput{RequestID: fmtID(), ProfileID: profile.ID, Version: 1, State: "archived"}); err != nil {
		t.Fatal(err)
	}
	running, cancel := context.WithCancel(ctx)
	defer cancel()
	clientReader, bridgeWriter := io.Pipe()
	bridgeReader, clientWriter := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Serve(running, Options{Server: f.http.URL, TokenFile: f.path, ManagedRunID: created.Run.ID, ManagedGeneration: 1}, &mcp.IOTransport{Reader: bridgeReader, Writer: bridgeWriter})
	}()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "managed-binding-test", Version: "1"}, nil)
	connecting, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()
	client, err := consumer.Connect(connecting, &mcp.IOTransport{Reader: clientReader, Writer: clientWriter}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "describe" {
			raw, _ := json.Marshal(tool.InputSchema)
			if strings.Contains(string(raw), "managed_generation") || strings.Contains(string(raw), "managed_run_id") {
				t.Fatal("model-controlled managed binding exposed")
			}
		}
	}
	bad := core.OpenSession{RequestID: fmtID(), Role: "developer", RunID: "foreign", Machine: p.Machine, ProfileID: profile.ID, ProfileVersion: 1}
	result, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "describe", Arguments: bad})
	if err != nil || !result.IsError {
		t.Fatal("bridge accepted foreign run binding", err)
	}
	bad.RequestID = fmtID()
	bad.RunID = created.Run.ID
	result, err = client.CallTool(ctx, &mcp.CallToolParams{Name: "describe", Arguments: bad})
	if err != nil {
		t.Fatal(err)
	}
	view := value[core.SessionView](t, result)
	if view.ManagedRunID != created.Run.ID || view.ManagedGeneration != 1 || view.Profile.Instructions != "Pinned original" {
		t.Fatal("fixed bridge did not bind authoritative original snapshot")
	}
	cancel()
	client.Close()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("managed bridge did not close")
	}
}
