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
	if len(args) == 0 {
		return errors.New("usage: agent-board serve | token create | call <operation>")
	}
	switch args[0] {
	case "serve":
		return serve(args[1:], out)
	case "token":
		if len(args) < 2 {
			return errors.New("usage: agent-board token create | revoke")
		}
		switch args[1] {
		case "create":
			return createToken(args[2:], out)
		case "revoke":
			return revokeToken(args[2:], out)
		default:
			return errors.New("usage: agent-board token create | revoke")
		}
	case "call":
		return call(args[1:], in, out)
	default:
		return errors.New("unknown command; use serve, token create, or call")
	}
}
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
	server := &http.Server{Handler: transport.NewHandler(s, s.Ready), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: time.Minute, MaxHeaderBytes: 16 << 10}
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
