// Package bridge exposes one lease-maintaining board session through stdio MCP.
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/client"
	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options describes a native client; TokenFile's contents are never protocol output.
type Options struct {
	Automation            *automationipc.Config
	ExecutionCapabilityID string
	ManagedRunID          string
	ManagedGeneration     int64
	Server                string
	ProjectID             string
	TokenFile             string
	HeartbeatInterval     time.Duration
	Stderr                io.Writer
}

type bridge struct {
	automation            *automationipc.Config
	executionCapabilityID string
	managedRunID          string
	managedGeneration     int64
	mu                    sync.Mutex
	upstream              *mcp.ClientSession
	// dial opens a fresh upstream session; used only before the first board
	// session, when a broken transport can be replaced without losing ownership.
	dial            func(context.Context) (*mcp.ClientSession, error)
	server          *mcp.Server
	session         core.SessionView
	task            *core.Lease
	taskKey         string
	sessionDeadline time.Time
	taskDeadline    time.Time
	lastHeartbeat   time.Time
	interval        time.Duration
	rpcTimeout      time.Duration
	override        time.Duration
	quarantined     bool
	successful      map[string]bool
	readOnly        map[string]bool
	wake            chan struct{}
	stderr          io.Writer
}

// Serve runs exactly one local MCP connection and stops renewing on EOF/cancel.
// It never releases a lease implicitly or recovers ownership after uncertainty.
func Serve(ctx context.Context, options Options, transport mcp.Transport) error {
	if (options.ManagedRunID == "") != (options.ManagedGeneration == 0) || options.ManagedGeneration < 0 {
		return errors.New("managed bridge requires an exact run and positive generation")
	}
	if (options.Automation != nil) != (options.ExecutionCapabilityID != "") || (options.Automation != nil && (options.ManagedRunID == "" || options.Automation.Binding.RunID != options.ManagedRunID || options.Automation.Binding.Generation != options.ManagedGeneration)) {
		return errors.New("automation bridge requires an exact private attempt binding")
	}
	if options.HeartbeatInterval < 0 {
		return errors.New("heartbeat interval must be positive")
	}
	if err := client.Project(options.ProjectID); err != nil {
		return err
	}
	origin, err := client.Origin(options.Server)
	if err != nil {
		return err
	}
	token, err := client.ReadToken(options.TokenFile)
	if err != nil {
		return err
	}
	if options.Stderr == nil {
		options.Stderr = io.Discard
	}
	silent := slog.New(slog.NewTextHandler(io.Discard, nil))
	upstream := mcp.NewClient(&mcp.Implementation{Name: "accord-bridge", Version: "0.2.0"}, &mcp.ClientOptions{Logger: silent})
	endpoint := origin.String() + "/mcp"
	if options.ProjectID != "" {
		endpoint += "/p/" + options.ProjectID
	}
	httpClient := client.HTTP(origin, token, 30*time.Second)
	dial := func(ctx context.Context) (*mcp.ClientSession, error) {
		connecting, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return upstream.Connect(connecting, &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: httpClient, MaxRetries: -1, DisableStandaloneSSE: true, MaxEventSize: 1 << 20}, nil)
	}
	remote, err := dial(ctx)
	if err != nil {
		return errors.New("cannot connect to Accord; check HTTPS origin and private token")
	}
	b := &bridge{automation: options.Automation, executionCapabilityID: options.ExecutionCapabilityID, managedRunID: options.ManagedRunID, managedGeneration: options.ManagedGeneration, upstream: remote, dial: dial, override: options.HeartbeatInterval, wake: make(chan struct{}, 1), successful: make(map[string]bool), readOnly: make(map[string]bool), stderr: options.Stderr}
	defer func() {
		b.mu.Lock()
		_ = b.upstream.Close()
		b.mu.Unlock()
	}()
	b.server = mcp.NewServer(&mcp.Implementation{Name: "accord-bridge", Version: "0.2.0"}, &mcp.ServerOptions{Logger: silent,
		Instructions: "Use describe once with a fresh request_id per process, role and run_id; omit machine (the server uses the token's machine) or pass exactly the token's machine. Session IDs are injected. This bridge renews ownership while alive. LEASE_UNCERTAIN or LEASE_LOST means stop local edits immediately and restart with a fresh describe; the bridge cannot stop your shell/file tools. Closing stdin stops heartbeat without releasing work. For workflow steps read accord://workflow and use workflow_run_check with fixed preset IDs. A queued check is not evidence: poll accord://workflow-check, or repeat workflow_run_check with the same request_id and preset_id, until completed before transitioning the task."})
	b.addCheckTool()
	listing, stopListing := context.WithTimeout(ctx, 10*time.Second)
	err = b.mirror(listing)
	if err == nil {
		err = b.mirrorResources(listing)
	}
	stopListing()
	if err != nil {
		return errors.New("cannot read Accord MCP tool catalogue")
	}
	running, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); b.pulse(running) }()
	err = b.server.Run(running, transport)
	stop()
	<-done
	if err != nil && !errors.Is(err, context.Canceled) {
		return errors.New("stdio MCP connection stopped")
	}
	return nil
}

func (b *bridge) mirror(ctx context.Context) error {
	count := 0
	for tool, err := range b.upstream.Tools(ctx, nil) {
		if err != nil {
			return err
		}
		count++
		if !b.allowedTool(tool.Name) {
			continue
		}
		if count > 200 {
			return errors.New("too many tools")
		}
		// Keep upstream schemas/descriptions, including future tools. Only local
		// session_id becomes optional; the authoritative service still validates it.
		copyTool, err := localTool(tool)
		if err != nil {
			return err
		}
		b.readOnly[copyTool.Name] = copyTool.Annotations != nil && copyTool.Annotations.ReadOnlyHint
		b.server.AddTool(copyTool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(ctx, req.Params.Name, req.Params.Arguments)
		})
	}
	if count == 0 {
		return errors.New("empty tool catalogue")
	}
	return nil
}

func (b *bridge) quarantine() {
	if b.quarantined {
		return
	}
	b.quarantined = true
	const message = "LEASE_UNCERTAIN: stop local edits; ownership is uncertain. Restart the bridge with a fresh describe; never resume stale generations."
	_, _ = io.WriteString(b.stderr, message+"\n")
	for session := range b.server.Sessions() {
		notification, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_ = session.Log(notification, &mcp.LoggingMessageParams{Level: "error", Logger: "accord-lease", Data: message})
		cancel()
	}
}

func failure(code, message string) *mcp.CallToolResult {
	value := map[string]string{"code": code, "message": message}
	encoded, _ := json.Marshal(value)
	return &mcp.CallToolResult{IsError: true, StructuredContent: value, Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}}
}

// reconnectLocked replaces a broken upstream transport. It is only called
// before the first board session, so no lease or generation is affected.
func (b *bridge) reconnectLocked(ctx context.Context) {
	if b.dial == nil {
		return
	}
	fresh, err := b.dial(ctx)
	if err != nil {
		return
	}
	old := b.upstream
	b.upstream = fresh
	_ = old.Close()
}

func (b *bridge) uncertain() *mcp.CallToolResult {
	b.quarantine()
	return failure("LEASE_UNCERTAIN", "Stop local edits. Restart the bridge with a fresh describe and verify ownership before continuing.")
}
