package transport

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"agent-board/internal/core"
)

// metrics is registered behind the same bearer authentication as API/MCP.
// All series are fixed and unlabelled: no token, owner, session or task IDs.
// Runtime Sys describes Go's runtime allocation; it is deliberately not RSS.
func (a *admission) metrics(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" || r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
		writeError(w, core.Fail("VALIDATION", "metrics does not accept a body or query parameters"))
		return
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# TYPE accord_uptime_seconds gauge\naccord_uptime_seconds %.3f\n", time.Since(a.started).Seconds())
	fmt.Fprintf(w, "# TYPE accord_http_requests_total counter\naccord_http_requests_total %d\n", a.requests.Load())
	fmt.Fprintf(w, "# TYPE accord_http_rate_limited_total counter\naccord_http_rate_limited_total %d\n", a.limited.Load())
	fmt.Fprintf(w, "# TYPE accord_http_concurrency_rejected_total counter\naccord_http_concurrency_rejected_total %d\n", a.saturated.Load())
	fmt.Fprintf(w, "# TYPE accord_http_active_requests gauge\naccord_http_active_requests %d\n", a.active.Load())
	fmt.Fprintf(w, "# TYPE accord_runtime_heap_bytes gauge\naccord_runtime_heap_bytes %d\n", memory.HeapAlloc)
	fmt.Fprintf(w, "# TYPE accord_runtime_sys_bytes gauge\naccord_runtime_sys_bytes %d\n", memory.Sys)
}
