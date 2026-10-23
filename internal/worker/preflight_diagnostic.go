package worker

import "errors"

const preparationError = "cannot prepare isolated native MCP connection"

// Only fixed stage codes reach the private journal. The original cause remains
// available to errors.Is/As, but Error and reports never expose its contents.
type preparationFailure struct {
	stage string
	cause error
}

func (e *preparationFailure) Error() string { return preparationError }
func (e *preparationFailure) Unwrap() error { return e.cause }

func validPreparationStage(stage string) bool {
	switch stage {
	case "prepare_unspecified", "prepare_stage_invalid", "prepare_binding", "prepare_run_directory", "prepare_attempt_directory", "prepare_bridge_file", "prepare_self", "prepare_claude_file", "prepare_override_transport", "prepare_override_reserved", "prepare_journal", "prepare_fatal", "prepare_closed", "prepare_state", "prepare_deadline", "prepare_authorization", "prepare_pins", "prepare_launch_deadline", "prepare_launch_pins":
		return true
	}
	for _, context := range []string{"discovery", "verify"} {
		if stage == "prepare_"+context {
			return true
		}
		for _, step := range []string{"start", "read", "limit", "timeout", "wait", "json", "metadata", "transport", "binding"} {
			if stage == "prepare_"+context+"_"+step {
				return true
			}
		}
	}
	return false
}

func preparationContext(stage string, cause error) error {
	var classified *preparationFailure
	if errors.As(cause, &classified) {
		return cause
	}
	return preparationFailureAt(stage, cause)
}

func preparationFailureAt(stage string, cause error) error {
	if !validPreparationStage(stage) {
		stage = "prepare_stage_invalid"
	}
	return &preparationFailure{stage: stage, cause: cause}
}

func preparationDiagnostic(err error) string {
	stage := "prepare_unspecified"
	var failure *preparationFailure
	if errors.As(err, &failure) {
		stage = failure.stage
	}
	if !validPreparationStage(stage) {
		stage = "prepare_stage_invalid"
	}
	if ownership := ownershipDiagnostic(err); ownership != "" {
		if !validDiagnostic(ownership) {
			ownership = "ownership_diagnostic_invalid"
		}
		stage += "__" + ownership
	}
	// The allowlists currently produce at most 100 ASCII bytes. This bound
	// prevents a future code change from turning the private field into a log.
	if len(stage) > 128 {
		return "prepare_stage_invalid"
	}
	return stage
}
