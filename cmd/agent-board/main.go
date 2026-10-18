// Command agent-board runs the local pilot and provides a JSON API client.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "agent-board:", err)
		os.Exit(1)
	}
}
func run(args []string, in io.Reader, out io.Writer) error {
	if helpRequested(args) {
		_, err := io.WriteString(out, commandHelp)
		return err
	}
	if len(args) == 0 {
		return errors.New("usage: agent-board serve | token | sessions list | backup | maintenance reset | mcp | call <operation>")
	}
	switch args[0] {
	case "serve":
		return serve(args[1:], out)
	case "mcp":
		return runMCP(args[1:], in, out)
	case "backup":
		return backup(args[1:], out)
	case "sessions":
		if len(args) < 2 || args[1] != "list" {
			return errors.New("usage: agent-board sessions list")
		}
		return listSessions(args[2:], out)
	case "maintenance":
		if len(args) < 2 || args[1] != "reset" {
			return errors.New("usage: agent-board maintenance reset")
		}
		return recoverState(args[2:], out)
	case "token":
		if len(args) < 2 {
			return errors.New("usage: agent-board token create | list | revoke | revoke-all")
		}
		switch args[1] {
		case "create":
			return createToken(args[2:], out)
		case "revoke":
			return revokeToken(args[2:], out)
		case "list":
			return listTokens(args[2:], out)
		case "revoke-all":
			return revokeAllTokens(args[2:], out)
		default:
			return errors.New("usage: agent-board token create | list | revoke | revoke-all")
		}
	case "call":
		return call(args[1:], in, out)
	default:
		return errors.New("unknown command; use --help")
	}
}

func helpRequested(args []string) bool {
	if len(args) == 1 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		return true
	}
	if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		switch args[0] {
		case "serve", "mcp", "call", "backup", "token", "sessions", "maintenance":
			return true
		}
	}
	return len(args) == 3 && (args[0] == "token" || args[0] == "sessions" || args[0] == "maintenance") && (args[2] == "-h" || args[2] == "--help")
}

const commandHelp = `Accord — authenticated coordination server and native client

agent-board serve --db PATH --listen 127.0.0.1:8080 [--external-host DNS:PORT] [--task-ttl 15m --session-ttl 15m]
agent-board mcp --server ORIGIN --token-file PRIVATE [--heartbeat-interval 1m]
agent-board call OPERATION --server ORIGIN --token-file PRIVATE < request.json
agent-board token create --db PATH --owner NAME --machine NAME --output NEW_PRIVATE [--days 30]
agent-board token list --db PATH
agent-board token revoke --db PATH --id TOKEN_ID
agent-board token revoke-all --db PATH --reason REASON
agent-board sessions list --db PATH
agent-board backup --db PATH --output NEW_PRIVATE
agent-board maintenance reset --db PATH --reason REASON

Remote origins require HTTPS; HTTP accepts literal loopback only. No redirects.
Tokens are read from a private file or AGENT_BOARD_TOKEN, never a secret argument.
The mcp command writes protocol only to stdout and renews the current lease.
On LEASE_UNCERTAIN/LEASE_LOST stop local edits; restart with a fresh describe.
EOF stops heartbeat without releasing work. Administrative commands run locally.
Recovery commands require the old server to be stopped; see docs/deployment.md.
`

func flags(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	return f
}
func parse(f *flag.FlagSet, args []string) error {
	if err := f.Parse(args); err != nil {
		return errors.New("invalid flags for " + f.Name())
	}
	if f.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	return nil
}
func openDB(path string, taskTTL, sessionTTL time.Duration) (*store.Store, error) {
	s, err := store.Open(path, core.Config{LeaseTTL: taskTTL, SessionTTL: sessionTTL, Project: "agent-board"}, nil)
	if err != nil {
		return nil, errors.New("cannot open board database")
	}
	return s, nil
}
func serve(args []string, out io.Writer) error {
	f := flags("serve")
	db := f.String("db", ".local/board.db", "database path")
	address := f.String("listen", "127.0.0.1:8080", "loopback IP:port")
	externalHost := f.String("external-host", os.Getenv("ACCORD_EXTERNAL_HOST"), "exact advertised DNS:port; defaults to ACCORD_EXTERNAL_HOST")
	taskTTL := f.Duration("task-ttl", 15*time.Minute, "task lease TTL")
	sessionTTL := f.Duration("session-ttl", 15*time.Minute, "session and singleton role TTL")
	if err := parse(f, args); err != nil {
		return err
	}
	if *taskTTL < time.Second || *taskTTL > 24*time.Hour || *sessionTTL < time.Second || *sessionTTL > 24*time.Hour {
		return errors.New("TTLs must be between 1s and 24h")
	}
	if err := transport.ValidateListen(*address); err != nil {
		return err
	}
	if err := transport.ValidateExternalHost(*externalHost); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return errors.New("cannot listen on requested loopback address")
	}
	defer listener.Close()
	s, err := openDB(*db, *taskTTL, *sessionTTL)
	if err != nil {
		return err
	}
	defer s.Close()
	server := &http.Server{Handler: transport.NewHandlerWithOptions(s, s.Ready, transport.HandlerOptions{ExternalHost: *externalHost}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: time.Minute, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
		case <-done:
		}
	}()
	fmt.Fprintln(out, "agent-board listening on http://"+listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("HTTP server stopped unexpectedly")
	}
	return nil
}
