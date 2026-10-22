package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
)

func TestOwnershipDiagnosticsAreFixedAndPrivate(t *testing.T) {
	err := ownershipSystemFailure("unix_reap_kill", syscall.EPERM)
	if !errors.Is(err, errOwnership) || ownershipDiagnostic(err) != "unix_reap_kill_errno_1" {
		t.Fatal("lost ownership stage or sentinel")
	}
	if err.Error() != errOwnership.Error() {
		t.Fatal("private diagnostic leaked to public error")
	}
	for _, unsafe := range []string{"plainsecret", "unix_reap_kill_errno_123456789", "https://user:secret@example.invalid", "unix_reap_kill_errno_01"} {
		if got := ownershipDiagnostic(ownershipFailureAt(unsafe)); got != "ownership_diagnostic_invalid" {
			t.Fatal("arbitrary diagnostic data accepted")
		}
	}
	state := RunState{ID: "synthetic", LocalDiagnostic: ownershipDiagnostic(fmt.Errorf("wrapped: %w", err))}
	data, _ := json.Marshal(state)
	if !strings.Contains(string(data), `"local_diagnostic":"unix_reap_kill_errno_1"`) {
		t.Fatal("private journal lost stage")
	}
	receipt := Receipt{CommandID: "synthetic", Error: err.Error()}
	data, _ = json.Marshal(receipt)
	if strings.Contains(string(data), "local_diagnostic") || strings.Contains(string(data), "unix_reap_kill") {
		t.Fatal("diagnostic reached public report fields")
	}
}
