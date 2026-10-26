package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"agent-board/internal/worker"
)

// workerSetup runs the interactive onboarding. Every answer comes from the
// owner's terminal; the only argument is the private configuration path, so
// no token or origin ever appears in a command line or shell history.
func workerSetup(args []string, out io.Writer) error {
	f := flags("worker setup")
	path := f.String("config", "", "private local worker configuration to create or edit")
	if err := parse(f, args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("worker setup requires --config PRIVATE_JSON")
	}
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return errors.New("worker setup is interactive: run it in a terminal, answers are not accepted from a pipe or arguments")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return worker.Setup(ctx, worker.SetupOptions{ConfigPath: *path, In: os.Stdin, Out: out})
}
