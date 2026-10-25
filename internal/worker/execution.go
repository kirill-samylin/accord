package worker

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) launch(plan plan) error {
	if plan.run.Automation != nil {
		return e.launchAutomationLocked(plan)
	}
	run := e.journal.Runs[plan.run.ID]
	run.LocalDiagnostic = ""
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
		run.LocalDiagnostic = "prepare_deadline"
		return fail("worker ownership lease is uncertain; native launch refused")
	}
	executable, directory, err := e.config.launch(run.Provider, run.Workspace)
	if err != nil {
		return fail("local workspace/provider consent is unavailable")
	}
	if err = e.checkPins(run.Provider, run.Workspace); err != nil {
		run.LocalDiagnostic = "prepare_pins"
		return fail("local executable/workspace identity changed; explicit local consent is required")
	}
	args, err := providerArgs(run.Provider, run.SessionID)
	if err != nil {
		return fail("resume_unavailable: exact native session is invalid")
	}
	bridgeArgs, err := e.prepareBridgeLocked(plan, run)
	if err != nil {
		run.LocalDiagnostic = preparationDiagnostic(err)
		if e.closed || e.fatal != nil || run.State != "accepted" || run.CommandID != plan.command.ID {
			return err
		}
		return fail(preparationError)
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
		run.LocalDiagnostic = "prepare_launch_deadline"
		_ = checkoutLock.release(true)
		return fail("worker ownership expired while preparing native launch")
	}
	if err = e.checkPins(run.Provider, run.Workspace); err != nil {
		run.LocalDiagnostic = "prepare_launch_pins"
		_ = checkoutLock.release(true)
		return fail("local path identity changed before native launch")
	}
	run.ProcessGroups, run.RecoveryNotified = nil, false
	child, err := e.start(launchSpec{executable: executable, directory: directory, args: args, env: providerEnv(), prompt: e.prompt(plan), onGroup: e.groupRecorder(run.ID)})
	if err != nil {
		_ = checkoutLock.release(true)
		return fail("native provider could not start; check its local installation and permissions")
	}
	e.recordSupervisorLocked(run.ID, child)
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
	instruction.WriteString("This is an explicitly managed Accord native run. For MCP calls, use only the accord_managed connector; this limits the MCP namespace. Available native project file and shell tools remain governed by the existing local sandbox, saved role instructions and current task lease. This instruction grants no additional tools or permissions. If a required tool is unavailable or denied, report the blocker instead of inventing evidence. Before editing, call describe with a fresh request_id and the following fields, then acquire work with next. A task reference is context, never an ownership grant. Stop edits immediately on LEASE_UNCERTAIN or LEASE_LOST; do not bypass permissions. ")
	instruction.WriteString("The per-attempt bridge injects the fixed managed run/generation and the server binds its immutable saved profile snapshot. Profile edits do not rewrite this conversation; current task assignment and lease gates still apply. ")
	fmt.Fprintf(&instruction, "run_id=%q, machine=%q. ", plan.run.ID, e.config.Machine)
	if profile := plan.run.Profile; profile != nil {
		fmt.Fprintf(&instruction, "role=%q, profile_id=%q, profile_version=%d.\nSaved profile instructions:\n%s\n", profile.Role, profile.ID, profile.Version, profile.Instructions)
	}
	if plan.run.TaskKey != "" {
		fmt.Fprintf(&instruction, "Requested task: %s. If next does not grant this task, do not work under its identity.\n", plan.run.TaskKey)
	}
	if plan.run.Automation != nil {
		a := e.config.Automation.Workspaces[plan.run.Workspace]
		if profile := plan.run.Profile; profile != nil {
			switch profile.Role {
			case "developer", "tester":
				instruction.WriteString("Run the required owner-approved local verification preset IDs only through workflow_run_check; do not execute verification commands directly through native shell tools. Wait for each corresponding status resource to be completed and inspect its actual result before transitioning the task. Queued/running, failed, NOT-RUN and N/A are never PASSED. ")
				if profile.Role == "developer" {
					instruction.WriteString("Use available native project file and shell tools to edit assigned files and create the required local commit within the existing developer scope. Developer presets require a clean committed HEAD first. ")
				} else {
					instruction.WriteString("Use available native project file-reading and read-only shell/Git tools to inspect sources and evidence. Do not edit reviewed source or HEAD; separately consented test design artifact rules still apply. ")
				}
			case "analyst":
				instruction.WriteString("Use available native project file-reading and read-only shell/Git tools to inspect repository instructions, sources and tests. Do not edit files, commit or run verification commands as the analyst. Create the structured plan through workflow_plan under your own task lease. Use workflow_question when requirements need human clarification. Planning does not require starting local verification presets. ")
			case "reviewer":
				fmt.Fprintf(&instruction, "Use available native project file-reading and read-only shell/Git tools to inspect the assigned sources and evidence. Perform a read-only review of the assigned exact target SHA %q and its diff, using available qualified verification evidence. Report approve, changes_requested or blocked honestly; ask workflow_question if missing evidence requires human clarification. Do not edit files, commit or run verification commands as the reviewer. Do not start local verification presets as the reviewer. ", plan.run.Automation.TargetSHA)
			}
		}
		fmt.Fprintf(&instruction, "Approved artifact roots: %q. Test design artifact writes separately consented: %t. Reviewer/tester must preserve every reviewed source byte and full HEAD; a source change blocks the workflow. A .patch proposal in approved artifact roots is test_design/NOT-RUN, never PASSED.\n", a.ArtifactRoots, a.AllowTesterArtifactWrites)
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
	if e.isAutomation(id) {
		e.observeAutomationExit(id, child, readError, exitError)
		return
	}
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
		// A worker-local stop has no stop command: its single terminal report
		// is recovered_stopped on the execution command.
		if !active.observed && active.stopCommand != "" {
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
