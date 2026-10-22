package worker

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) launch(plan plan) error {
	run := e.journal.Runs[plan.run.ID]
	fail := func(message string) error {
		run.State = "failed"
		run.Error = message
		e.reportFor(plan.command.ID, "failed", run, message, "")
		return e.save()
	}
	if plan.run.Profile == nil {
		return fail("immutable agent profile snapshot is required")
	}
	if time.Now().After(e.deadline) {
		return fail("worker ownership lease is uncertain; native launch refused")
	}
	executable, directory, err := e.config.launch(run.Provider, run.Workspace)
	if err != nil {
		return fail("local workspace/provider consent is unavailable")
	}
	if err = e.checkPins(run.Provider, run.Workspace); err != nil {
		return fail("local executable/workspace identity changed; explicit local consent is required")
	}
	args, err := providerArgs(run.Provider, run.SessionID)
	if err != nil {
		return fail("resume_unavailable: exact native session is invalid")
	}
	bridgeArgs, err := e.prepareBridgeLocked(plan, run)
	if err != nil {
		if e.closed || e.fatal != nil || run.State != "accepted" || run.CommandID != plan.command.ID {
			return err
		}
		return fail("cannot prepare isolated native MCP connection")
	}
	args = append(bridgeArgs, args...)
	if run.Provider == "codex" && e.config.CodexSandbox != "" {
		args = append([]string{"-c", fmt.Sprintf("sandbox_mode=%q", e.config.CodexSandbox)}, args...)
	}
	if run.Provider == "claude" && e.config.ClaudePermissionMode != "" {
		args = append(args, "--permission-mode", e.config.ClaudePermissionMode)
	}
	if profile := plan.run.Profile; profile != nil {
		expected := map[string]string{"codex": "openai", "claude": "anthropic"}[run.Provider]
		if profile.Provider != expected && profile.Provider != "unconfigured" {
			return fail("profile provider differs from locally allowed native provider")
		}
		if profile.Model != "" {
			allowed := false
			for _, model := range e.config.Models[run.Provider] {
				if model == profile.Model {
					allowed = true
				}
			}
			if !allowed {
				return fail("profile model is not locally allowed")
			}
			args = append(args, "--model", profile.Model)
		}
	}
	run.State, run.CommandID, run.Generation, run.Error = "launching", plan.command.ID, plan.command.Generation, ""
	// A durable launch intent is written only after accepted is confirmed by
	// the server, and before any native process can be created.
	if err = e.save(); err != nil {
		return err
	}
	checkoutLock, err := acquireWorkspace(directory, e.workspaceLockDir, e.journal.WorkerID+" "+run.ID)
	if err != nil {
		return fail("workspace is busy or has an unresolved local launch; local recovery required")
	}
	if time.Now().After(e.deadline) {
		_ = checkoutLock.release(true)
		return fail("worker ownership expired while preparing native launch")
	}
	if err = e.checkPins(run.Provider, run.Workspace); err != nil {
		_ = checkoutLock.release(true)
		return fail("local path identity changed before native launch")
	}
	child, err := e.start(launchSpec{executable: executable, directory: directory, args: args, env: providerEnv(), prompt: e.prompt(plan)})
	if err != nil {
		_ = checkoutLock.release(true)
		return fail("native provider could not start; check its local installation and permissions")
	}
	e.active[run.ID] = &activeRun{process: child, workspaceLock: checkoutLock}
	e.wg.Add(1)
	go e.observe(run.ID, child)
	return nil
}

func (e *Engine) prompt(plan plan) string {
	text := plan.run.Prompt
	if plan.command.Kind != "start" {
		text = plan.command.Text
	}
	var instruction strings.Builder
	instruction.WriteString("This is an explicitly managed Accord native run. Use only the accord_managed MCP connector for this run. Before editing, call describe with a fresh request_id and the following fields, then acquire work with next. A task reference is context, never an ownership grant. Stop edits immediately on LEASE_UNCERTAIN or LEASE_LOST; do not bypass permissions. ")
	instruction.WriteString("The per-attempt bridge injects the fixed managed run/generation and the server binds its immutable saved profile snapshot. Profile edits do not rewrite this conversation; current task assignment and lease gates still apply. ")
	fmt.Fprintf(&instruction, "run_id=%q, machine=%q. ", plan.run.ID, e.config.Machine)
	if profile := plan.run.Profile; profile != nil {
		fmt.Fprintf(&instruction, "role=%q, profile_id=%q, profile_version=%d.\nSaved profile instructions:\n%s\n", profile.Role, profile.ID, profile.Version, profile.Instructions)
	}
	if plan.run.TaskKey != "" {
		fmt.Fprintf(&instruction, "Requested task: %s. If next does not grant this task, do not work under its identity.\n", plan.run.TaskKey)
	}
	instruction.WriteString("User request:\n")
	instruction.WriteString(text)
	return instruction.String()
}

func (e *Engine) observe(id string, child process) {
	defer e.wg.Done()
	scanner := bufio.NewScanner(child.Output())
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		e.mu.Lock()
		run, active := e.journal.Runs[id], e.active[id]
		if active == nil {
			e.mu.Unlock()
			break
		}
		event := parseProviderEvent(run.Provider, scanner.Bytes())
		if event.session != "" {
			if run.SessionID != "" && run.SessionID != event.session {
				active.failed = true
				run.Error = "native provider changed exact session identity"
				go child.Stop()
			} else {
				run.SessionID = event.session
			}
		}
		if event.started && !active.observed && run.SessionID != "" && !active.failed && !active.stop {
			active.observed = true
			run.State = "running"
			e.reportFor(run.CommandID, "started", run, "", "")
		}
		active.complete = active.complete || event.complete
		active.failed = active.failed || event.failed
		if event.output != "" {
			run.Output = cleanOutput(event.output, e.outputSecrets()...)
		}
		if err := e.save(); err != nil {
			go child.Stop()
		}
		e.mu.Unlock()
	}
	readError := scanner.Err()
	child.Output().Close()
	if readError != nil {
		_ = child.Stop()
	}
	exitError := child.Wait()
	e.mu.Lock()
	defer e.mu.Unlock()
	run, active := e.journal.Runs[id], e.active[id]
	if active == nil {
		return
	}
	delete(e.active, id)
	if active.workspaceLock != nil {
		if err := active.workspaceLock.release(!active.uncertain && !errors.Is(exitError, errOwnership)); err != nil {
			active.uncertain = true
			run.LocalDiagnostic = "workspace_lock_release_failed"
		}
	}
	if active.uncertain || errors.Is(exitError, errOwnership) {
		if code := ownershipDiagnostic(exitError); code != "" {
			run.LocalDiagnostic = code
		}
		run.State = "needs_recovery"
		run.Error = "native process ownership is uncertain; local recovery required"
		e.reportFor(run.CommandID, "delivery_unknown", run, run.Error, "")
	} else if active.stop {
		run.State = "stopped"
		if !active.observed {
			e.reportFor(run.CommandID, "failed", run, "stopped before native turn was observed", "")
		}
		if active.stopCommand != "" {
			e.journal.Receipts[active.stopCommand].Forced = child.Forced()
			e.reportFor(active.stopCommand, "stopped", run, "", "")
		} else {
			e.journal.Receipts[run.CommandID].Forced = child.Forced()
			e.reportFor(run.CommandID, "recovered_stopped", run, run.Error, "")
		}
	} else if active.complete && active.observed && !active.failed && readError == nil && exitError == nil {
		run.State = "completed"
		e.reportFor(run.CommandID, "completed", run, "", run.Output)
	} else {
		run.State = "failed"
		if run.Error == "" {
			run.Error = "native provider failed or did not confirm a complete turn; check local authentication and permissions"
		}
		e.reportFor(run.CommandID, "failed", run, run.Error, "")
	}
	_ = e.save()
}

func boundedText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && text[limit]&0xc0 == 0x80 {
		limit--
	}
	return text[:limit]
}

func (e *Engine) StopAll(reason string) error {
	e.mu.Lock()
	e.stopEpoch++
	var children []process
	for id, active := range e.active {
		active.stop = true
		e.journal.Runs[id].State = "stopping"
		e.journal.Runs[id].Error = reason
		children = append(children, active.process)
	}
	err := e.save()
	e.mu.Unlock()
	err = errors.Join(err, e.stopBatch(children))
	done := make(chan struct{})
	go func() { e.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		err = errOwnership
	}
	return err
}

func (e *Engine) Close() error {
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return nil
	}
	err := e.StopAll("local worker is stopping")
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.closed {
		e.closed = true
		e.disk.close()
	}
	return err
}

func outputLines(receipt Receipt) []string {
	if receipt.Output == "" {
		return nil
	}
	var lines []string
	text := receipt.Output
	for len(text) > 0 {
		chunk := boundedText(text, 4096)
		lines = append(lines, chunk)
		text = text[len(chunk):]
	}
	return lines
}
