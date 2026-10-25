package worker

import "testing"

// The server sends core.Fail codes upper-cased; recovery classification must
// not depend on case, and a live previous lease stays a transient refusal.
func TestRecoveryCodesMatchServerUpperCase(t *testing.T) {
	for _, code := range []string{"STALE_GENERATION", "STALE_DISPATCH", "RUN_ARCHIVED", "COMMAND_TERMINAL"} {
		if !definitiveRecoveryRejection(statusError{code: 409, server: code, message: "x"}) {
			t.Fatalf("%s not treated as settled", code)
		}
	}
	if !definitiveRecoveryRejection(statusError{code: 404, server: "RUN_ARCHIVED"}) || !definitiveRecoveryRejection(statusError{code: 404, server: "NOT_FOUND"}) {
		t.Fatal("Accord 404 for an archived/unknown run must settle the report")
	}
	if definitiveRecoveryRejection(statusError{code: 404}) {
		t.Fatal("a bare 404 without an Accord code must be retried, not dropped")
	}
	if definitiveRecoveryRejection(statusError{code: 409, server: "LEASE_HELD", message: "previous dispatch lease is live"}) {
		t.Fatal("LEASE_HELD must stay transient")
	}
	if !recoveryConflict(statusError{code: 409, server: "MANAGED_UNCERTAIN", message: "x"}) {
		t.Fatal("MANAGED_UNCERTAIN poll refusal not recognised")
	}
}
