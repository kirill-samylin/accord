package transport

import (
	"net/http"
	"testing"

	"agent-board/internal/core"
)

// The worker's managed contract codes: archived runs are definitive 404s, the
// others are conflicts with their own code (LEASE_HELD stays transient).
func TestManagedContractCodesKeepTheirHTTPStatus(t *testing.T) {
	for code, status := range map[string]int{"run_archived": http.StatusNotFound, "not_found": http.StatusNotFound, "stale_generation": http.StatusConflict, "stale_dispatch": http.StatusConflict, "command_terminal": http.StatusConflict, "managed_uncertain": http.StatusConflict, "lease_held": http.StatusConflict} {
		body, got := publicError(core.Fail(code, "message kept"))
		if got != status || body.Error.Message != "message kept" {
			t.Fatalf("%s: %d %+v, want %d", code, got, body.Error, status)
		}
	}
}
