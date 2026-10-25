package worker

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/core"
)

func (e *Engine) executeAutomationCheck(id string, c *automationCheck, p CheckPreset) {
	defer e.wg.Done()
	e.mu.Lock()
	active := e.active[id]
	run := e.journal.Runs[id]
	rt := active.automation
	consent := e.config.Automation.Workspaces[run.Workspace]
	directory := e.config.Workspaces[run.Workspace].Path
	baseline := run.Automation.Baseline
	role := run.Automation.Run.Profile.Role
	target := run.Automation.Run.Automation.TargetSHA
	e.mu.Unlock()
	defer e.finishCheckCount(id, rt)
	deadline := minDeadline(rt.deadline, time.Now().Add(time.Duration(p.TimeoutSeconds)*time.Second))
	ctx, cancel := context.WithDeadline(rt.ctx, deadline)
	defer cancel()
	var holder core.Task
	if err := e.automationRead(ctx, "/tasks/"+c.Request.TaskKey, &holder); err != nil || holder.Lease == nil || holder.Key != c.Request.TaskKey || holder.Lease.SessionID != c.Request.SessionID || holder.Lease.Generation != c.Request.TaskGeneration || holder.Generation != c.Request.TaskGeneration || !holder.Lease.ExpiresAt.After(time.Now()) {
		e.failCheck(id, c, "lease_checkpoint_unavailable", false)
		return
	}
	if verifyExecutable(p.Executable, p.ExecutableSHA256) != nil {
		e.failCheck(id, c, "preset_changed", false)
		return
	}
	before, err := e.observeAutomationGit(ctx, consent, directory)
	if err != nil || unchangedProof(baseline, before, role != "developer") != nil {
		e.failCheck(id, c, "source_proof_unavailable", errors.Is(err, errOwnership))
		return
	}
	if role == "tester" && before.Head != target {
		e.failCheck(id, c, "reviewed_sha_changed", false)
		return
	}
	cwd, err := canonicalCheckDirectory(directory, p.Directory)
	if err != nil {
		e.failCheck(id, c, "preset_directory_changed", false)
		return
	}
	if p.ReportPath != "" {
		if err = prepareReport(directory, p.ReportPath); err != nil {
			e.failCheck(id, c, "report_path_unavailable", false)
			return
		}
	}
	reserved := e.checkAttestation(id, c, "check_reserved", c.ReservedRequestID)
	response, err := e.sendAttestation(ctx, reserved)
	if err != nil {
		e.failCheck(id, c, "reservation_unavailable", false)
		return
	}
	if response.MaxRuntimeSeconds < 1 || response.MaxRuntimeSeconds > 7200 || response.ReservationTTLSeconds < 1 || response.ReservationTTLSeconds > 30 {
		e.failCheck(id, c, "reservation_expired", false)
		return
	}
	deadline = minDeadline(deadline, time.Now().Add(time.Duration(response.MaxRuntimeSeconds)*time.Second))
	reservationUntil := time.Now().Add(time.Duration(response.ReservationTTLSeconds) * time.Second)
	e.mu.Lock()
	if !e.checkLaunchAllowed(id, rt) || !time.Now().Before(reservationUntil) {
		e.mu.Unlock()
		e.failCheck(id, c, "check_cancelled", false)
		return
	}
	c.Status = "launching"
	c.StartedAt = time.Now()
	err = e.save()
	e.mu.Unlock()
	if err != nil {
		return
	}
	child, err := e.start(launchSpec{executable: p.Executable, directory: cwd, args: append([]string{}, p.Args...), env: providerEnv(), onGroup: e.groupRecorder(id)})
	if err != nil {
		e.failCheck(id, c, "check_not_started", false)
		return
	}
	e.mu.Lock()
	e.recordSupervisorLocked(id, child)
	rt.checks[c.ID] = child
	allowed := e.checkLaunchAllowed(id, rt)
	e.mu.Unlock()
	if !allowed {
		_ = child.Stop()
	}
	done := make(chan struct{})
	go func() {
		timer := time.NewTimer(max(0, time.Until(deadline)))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			_ = child.Stop()
		case <-timer.C:
			_ = child.Stop()
		case <-done:
		}
	}()
	// Drain concurrently with the started receipt. A verbose executable must not
	// block a pipe while its owner renews authority or waits for HTTP.
	drained := make(chan bool, 1)
	go func() {
		_, truncated, copyErr := captureProcess(child, 1<<20, false)
		drained <- truncated || copyErr != nil
	}()
	started := e.checkAttestation(id, c, "check_started", c.StartedRequestID)
	_, startErr := e.sendAttestation(ctx, started)
	if startErr == nil {
		e.mu.Lock()
		c.Status = "running"
		startErr = e.save()
		e.mu.Unlock()
	}
	var stopErr error
	if startErr != nil {
		stopErr = child.Stop()
	}
	truncated := <-drained
	waitErr := child.Wait()
	close(done)
	e.mu.Lock()
	delete(rt.checks, c.ID)
	e.mu.Unlock()
	if errors.Is(waitErr, errOwnership) || errors.Is(stopErr, errOwnership) {
		e.failCheck(id, c, "check_delivery_unknown", true)
		return
	}
	if startErr != nil {
		// A definitive 4xx (e.g. a human pause changed the control epoch)
		// means the server never recorded this check as running; the owned
		// process is stopped and reaped, so it is simply not run. Transport
		// errors or a failed save keep the uncertain path.
		var rejected statusError
		if errors.As(startErr, &rejected) && rejected.code >= 400 && rejected.code < 500 {
			e.failCheck(id, c, "check_cancelled", false)
			return
		}
		e.failCheck(id, c, "check_delivery_unknown", true)
		return
	}
	code, known := processExitCode(child)
	if !known {
		e.failCheck(id, c, "check_exit_unproven", true)
		return
	}
	proofCtx, proofCancel := context.WithTimeout(context.Background(), 20*time.Second)
	after, proofErr := e.observeAutomationGit(proofCtx, consent, directory)
	proofCancel()
	if errors.Is(proofErr, errOwnership) {
		e.failCheck(id, c, "git_observation_ownership_uncertain", true)
		return
	}
	clean := proofErr == nil && unchangedProof(before, after, true) == nil
	observations := core.WorkflowObservations{ActualHead: after.Head, TargetSHA: target, SourceClean: &clean, ManifestSHA256: after.Manifest, CheckExitCode: &code, OutcomeBytesTruncated: truncated}
	if role == "developer" {
		observations.TargetSHA = before.Head
	}
	result := &automationipc.Result{ExitCode: &code, ActualHead: after.Head, ManifestSHA256: after.Manifest}
	errorCode := ""
	if p.ReportFormat != "" {
		path, pathErr := safeRelativeFile(directory, p.ReportPath)
		var raw []byte
		var hash string
		if pathErr == nil {
			raw, hash, pathErr = readArtifact(path, 4<<20)
		}
		if pathErr == nil {
			counts, parseErr := parseTestReport(p.ReportFormat, raw)
			if parseErr == nil {
				if persistErr := e.persistCheckArtifact(id, c.ID, p.ReportFormat, raw); persistErr != nil {
					e.failCheck(id, c, "check_artifact_not_durable", true)
					return
				}
				observations.ReportKind = p.ReportFormat
				observations.ReportSHA256 = hash
				observations.ChecksPassed = counts.Passed
				observations.ChecksFailed = counts.Failed
				observations.ChecksSkipped = counts.Skipped
				observations.ArtifactRefs = []string{"sha256:" + hash}
				result.Passed, result.Failed, result.Skipped = counts.Passed, counts.Failed, counts.Skipped
				result.ReportKind, result.ReportSHA256 = p.ReportFormat, hash
				result.ArtifactRefs = observations.ArtifactRefs
			} else {
				pathErr = parseErr
			}
		}
		if pathErr != nil {
			errorCode = "invalid_test_report"
		}
	} else {
		observations.ReportKind = "exit"
		result.ReportKind = "exit"
	}
	if !clean {
		errorCode = "source_changed"
	}
	if truncated {
		errorCode = "check_output_truncated"
	}
	if ctx.Err() != nil || !time.Now().Before(deadline) {
		errorCode = "check_deadline_or_stop"
		observations.CheckAborted = true
	}
	e.mu.Lock()
	c.ErrorCode = errorCode
	c.Observations = observations
	c.Result = result
	c.Status = "report_pending"
	err = e.save()
	e.mu.Unlock()
	if err != nil {
		return
	}
	finished := e.checkAttestation(id, c, "check_finished", c.FinishedRequestID)
	if err = e.queueAutomationAttestation(id, finished); err != nil {
		e.failCheck(id, c, "check_proof_not_durable", true)
	}
}
func minDeadline(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func (e *Engine) checkLaunchAllowed(id string, rt *automationRuntime) bool {
	r, a := e.journal.Runs[id], e.active[id]
	return r != nil && a != nil && a.automation == rt && !a.stop && !a.uncertain && !rt.closing && rt.ctx.Err() == nil && !r.Automation.NativeExited && time.Now().Before(e.deadline) && time.Now().Before(rt.deadline) && e.fatal == nil && !e.closed && e.checkPins(r.Provider, r.Workspace) == nil
}
func canonicalCheckDirectory(directory, relative string) (string, error) {
	path := filepath.Join(directory, relative)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return "", consentError()
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", consentError()
	}
	return path, nil
}
func prepareReport(directory, relative string) error {
	path := filepath.Join(directory, relative)
	parent := filepath.Dir(relative)
	if parent != "." {
		if err := createArtifactParents(directory, parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return consentError()
	}
	return os.Remove(path)
}
func readBoundedReport(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 4<<20+1))
	if err != nil || len(raw) > 4<<20 {
		return nil, errors.New("report too large")
	}
	return raw, nil
}

func createArtifactParents(directory, relative string) error {
	current := directory
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err = os.Mkdir(current, 0700); err != nil {
				return err
			}
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return consentError()
		}
	}
	return nil
}
