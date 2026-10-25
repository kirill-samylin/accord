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
		return errors.New("usage: agent-board worker enroll|run|identity|consent|recover|automation-preview --config PRIVATE [--run RUN_ID [--unverified]]")
	}
	operation := args[0]
	if operation == "automation-preview" {
		return workerAutomationPreview(args[1:], out)
	}
	if operation == "supervise" {
		if len(args) != 1 {
			return errors.New("invalid local supervisor invocation")
		}
		return worker.Supervise(os.Stdin, out)
	}
	if operation != "enroll" && operation != "run" && operation != "bridge" && operation != "identity" && operation != "consent" && operation != "recover" {
		return errors.New("unknown local worker operation")
	}
	f := flags("worker " + operation)
	path := f.String("config", "", "private local worker configuration")
	runID := f.String("run", "", "local run ID to recover (worker recover only)")
	unverified := f.Bool("unverified", false, "recover a run without recorded process groups; the owner confirms no agent of the run is still running (worker recover only)")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("worker requires --config PRIVATE")
	}
	if (operation == "recover") != (*runID != "") {
		return errors.New("--run RUN_ID is required for worker recover and only allowed there")
	}
	if *unverified && operation != "recover" {
		return errors.New("--unverified is only allowed for worker recover")
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
		return runMCPWithAutomationBinding([]string{"--server", config.Server, "--project", config.Project}, os.Stdin, out, config.ManagedRunID, config.ManagedGeneration, config.Automation, config.ExecutionCapabilityID)
	}
	config, err := worker.LoadConfig(*path)
	if err != nil {
		return err
	}
	// Open takes the exclusive private state lock: recover refuses while a
	// running worker process holds it.
	engine, err := worker.Open(config)
	if err != nil {
		return err
	}
	defer engine.Close()
	engine.SetLogger(func(line string) { fmt.Fprintln(os.Stderr, "accord worker:", line) })
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
	if operation == "recover" {
		if *unverified {
			fmt.Fprintln(os.Stderr, "accord worker: WARNING: --unverified recovery. No recorded process group of this run can be checked; the worker only scans local processes for native provider processes of this workspace or native session. By continuing you confirm that no agent process of run", *runID, "is still running. / ВНИМАНИЕ: восстановление без проверки групп процессов; вы подтверждаете, что ни один агент этого запуска больше не работает.")
		}
		if err = engine.RecoverRun(ctx, *runID, worker.RecoverOptions{Unverified: *unverified}); err != nil {
			return err
		}
		message := "recovered: no owned process is alive; reported recovered_stopped to Accord."
		if *unverified {
			message = "recovered without process-group verification (owner confirmed); no native provider process of this run was found; reported recovered_stopped to Accord."
		}
		_, err = fmt.Fprintln(out, "Local run", *runID, message)
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
