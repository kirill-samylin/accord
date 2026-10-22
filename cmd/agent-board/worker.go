package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"agent-board/internal/worker"
)

func workerCommand(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: agent-board worker enroll|run --config PRIVATE")
	}
	operation := args[0]
	if operation == "supervise" {
		if len(args) != 1 {
			return errors.New("invalid local supervisor invocation")
		}
		return worker.Supervise(os.Stdin, out)
	}
	if operation != "enroll" && operation != "run" && operation != "bridge" && operation != "identity" && operation != "consent" {
		return errors.New("unknown local worker operation")
	}
	f := flags("worker " + operation)
	path := f.String("config", "", "private local worker configuration")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("worker requires --config PRIVATE")
	}
	if operation == "bridge" {
		config, token, err := worker.ReadBridgeConfig(*path)
		if err != nil {
			return err
		}
		// This isolated child process supplies the credential through its own
		// environment; Windows files were checked against their native DACL.
		if err = os.Setenv("AGENT_BOARD_TOKEN", token); err != nil {
			return errors.New("cannot initialize managed bridge credential")
		}
		return runMCPWithBinding([]string{"--server", config.Server, "--project", config.Project}, os.Stdin, out, config.ManagedRunID, config.ManagedGeneration)
	}
	config, err := worker.LoadConfig(*path)
	if err != nil {
		return err
	}
	engine, err := worker.Open(config)
	if err != nil {
		return err
	}
	defer engine.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.Signal(1))
	defer cancel()
	if operation == "identity" {
		id, err := engine.Identity(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Project-local controller ID:", id)
		return err
	}
	if operation == "consent" {
		host, err := engine.Consent(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Local host consent updated:", host.ID, "— controllers:", len(host.Controllers))
		return err
	}
	if operation == "enroll" {
		host, err := engine.Enroll(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Local host enrolled:", host.ID, "— owner controller ID:", host.OwnerPrincipalID, "— explicitly allowed additional controllers:", len(host.Controllers))
		return err
	}
	_, _ = io.WriteString(out, "Accord worker is running; native authentication and permission prompts remain local.\n")
	err = engine.Run(ctx)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
