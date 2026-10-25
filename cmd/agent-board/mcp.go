package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/bridge"
)

func runMCP(args []string, in io.Reader, out io.Writer) error {
	return runMCPWithBinding(args, in, out, "", 0)
}
func runMCPWithBinding(args []string, in io.Reader, out io.Writer, managedRunID string, managedGeneration int64) error {
	return runMCPWithAutomationBinding(args, in, out, managedRunID, managedGeneration, nil, "")
}
func runMCPWithAutomationBinding(args []string, in io.Reader, out io.Writer, managedRunID string, managedGeneration int64, automation *automationipc.Config, capabilityID string) error {
	f := flags("mcp")
	server := f.String("server", "http://127.0.0.1:8080", "server origin")
	project := f.String("project", "", "opaque project ID")
	tokenFile := f.String("token-file", "", "private token file; otherwise AGENT_BOARD_TOKEN")
	interval := f.Duration("heartbeat-interval", 0, "optional interval, may only reduce server-selected interval")
	if err := parse(f, args); err != nil {
		return err
	}
	if *interval < 0 || *interval > 24*time.Hour {
		return errors.New("heartbeat interval must be positive and at most 24h")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reader, ok := in.(io.ReadCloser)
	if !ok {
		reader = io.NopCloser(in)
	}
	return bridge.Serve(ctx, bridge.Options{Automation: automation, ExecutionCapabilityID: capabilityID, ManagedRunID: managedRunID, ManagedGeneration: managedGeneration, Server: *server, ProjectID: *project, TokenFile: *tokenFile, HeartbeatInterval: *interval, Stderr: os.Stderr}, bridge.StdioTransport(reader, writerNoClose{out}, 64<<10))
}

type writerNoClose struct{ io.Writer }

func (writerNoClose) Close() error { return nil }
