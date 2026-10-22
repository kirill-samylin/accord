package worker

import (
	"errors"
	"strconv"
	"strings"
	"syscall"
)

// Diagnostics are private fixed stage codes, never native stderr, paths,
// process IDs, prompts, provider output, or credentials. Error() stays generic
// so the ordinary report/UI error cannot accidentally expose local details.
type ownershipFailure struct{ code string }

func (e *ownershipFailure) Error() string { return errOwnership.Error() }
func (e *ownershipFailure) Unwrap() error { return errOwnership }

func validDiagnostic(code string) bool {
	switch code {
	case "ownership_unspecified", "ownership_diagnostic_invalid", "unix_reap_census_leader_missing", "unix_exit_census_leader_missing", "unix_group_exit_timeout", "supervisor_helper_exit_failed", "supervisor_missing_nonce_proof", "supervisor_stop_proof_timeout", "supervisor_stop_helper_timeout", "supervisor_unproven_exit", "workspace_lock_release_failed", "stop_unclassified_failure":
		return true
	}
	for _, stage := range []string{"unix_waitid", "unix_reap_census", "unix_reap_kill", "unix_stop_interrupt", "unix_stop_force", "unix_exit_census"} {
		if code == stage+"_failed" {
			return true
		}
		if value, ok := strings.CutPrefix(code, stage+"_errno_"); ok {
			errno, err := strconv.Atoi(value)
			return err == nil && errno >= 0 && errno <= 4095 && strconv.Itoa(errno) == value
		}
	}
	return false
}

func ownershipFailureAt(code string) error {
	if !validDiagnostic(code) {
		code = "ownership_diagnostic_invalid"
	}
	return &ownershipFailure{code: code}
}

func ownershipSystemFailure(stage string, err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return ownershipFailureAt(stage + "_errno_" + strconv.FormatUint(uint64(errno), 10))
	}
	return ownershipFailureAt(stage + "_failed")
}

func ownershipDiagnostic(err error) string {
	var failure *ownershipFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	if errors.Is(err, errOwnership) {
		return "ownership_unspecified"
	}
	return ""
}
