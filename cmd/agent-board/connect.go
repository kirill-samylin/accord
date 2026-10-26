package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"agent-board/internal/connect"
)

// connectCommand registers Accord in the owner's CLIs with no dialogue. Flags
// carry only a path and identifiers; the token stays in its file.
func connectCommand(args []string, out io.Writer) error {
	f := flags("connect")
	tokenFile := f.String("token-file", "", "private token file issued for this person")
	server := f.String("server", "", "HTTPS origin; defaults to connection.json beside the token")
	project := f.String("project", "", "project ID; defaults to connection.json beside the token")
	if err := parse(f, args); err != nil {
		return err
	}
	if *tokenFile == "" {
		return errors.New("usage: agent-board connect --token-file PRIVATE [--server ORIGIN --project ID]")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	report, err := connect.Connect(ctx, connect.Options{TokenFile: *tokenFile, Server: *server, Project: *project, Out: out})
	if err != nil {
		return err
	}
	report.Print(out)
	return nil
}
