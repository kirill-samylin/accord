package transport

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"agent-board/internal/core"
)

// HandlerOptions bounds admission globally, before authentication or storage.
// Nonpositive/nonfinite values use defaults; limits cannot be disabled. There
// are no buckets keyed by attacker-controlled bearer tokens or proxy headers.
type HandlerOptions struct {
	RequestsPerSecond float64
	Burst             int
	MaxConcurrent     int
}

type admission struct {
	mu        sync.Mutex
	tokens    float64
	last      time.Time
	rate      float64
	burst     float64
	inflight  chan struct{}
	started   time.Time
	requests  atomic.Uint64
	limited   atomic.Uint64
	saturated atomic.Uint64
	active    atomic.Int64
}

func newAdmission(options HandlerOptions) *admission {
	if options.RequestsPerSecond <= 0 || math.IsNaN(options.RequestsPerSecond) || math.IsInf(options.RequestsPerSecond, 0) {
		options.RequestsPerSecond = 100
	}
	if options.Burst <= 0 {
		options.Burst = 200
	}
	if options.MaxConcurrent <= 0 {
		options.MaxConcurrent = 64
	}
	now := time.Now()
	return &admission{
		tokens: float64(options.Burst), last: now, rate: options.RequestsPerSecond,
		burst: float64(options.Burst), inflight: make(chan struct{}, options.MaxConcurrent), started: now,
	}
}

// take uses one fixed-size bucket. Rejected requests do not reserve future
// tokens, so sustained input cannot keep delaying recovery after it stops.
func (a *admission) take(now time.Time) (bool, float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if now.After(a.last) {
		a.tokens = math.Min(a.burst, a.tokens+now.Sub(a.last).Seconds()*a.rate)
		a.last = now
	}
	if a.tokens < 1 {
		return false, (1 - a.tokens) / a.rate
	}
	a.tokens--
	return true, 0
}

func (a *admission) enter(w http.ResponseWriter) bool {
	a.requests.Add(1)
	if allowed, wait := a.take(time.Now()); !allowed {
		a.limited.Add(1)
		// Retry-After is an integer number of seconds, rounded up to avoid
		// encouraging retries before the next token is available.
		w.Header().Set("Retry-After", strconv.FormatFloat(math.Max(1, math.Ceil(wait)), 'f', 0, 64))
		writeJSON(w, http.StatusTooManyRequests, errorResponse{core.Error{Code: "RATE_LIMITED", Message: "request rate exceeded; retry later"}})
		return false
	}
	select {
	case a.inflight <- struct{}{}:
		a.active.Add(1)
		return true
	default:
		a.saturated.Add(1)
		w.Header().Set("Retry-After", "1")
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{core.Error{Code: "BUSY", Message: "concurrent request limit reached; retry later"}})
		return false
	}
}

func (a *admission) leave() {
	a.active.Add(-1)
	<-a.inflight
}

func nativePath(path string) bool {
	for _, root := range []string{"/api", "/mcp", "/metrics"} {
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

func hasOrigin(header http.Header) bool {
	// Get cannot distinguish an absent Origin from an empty one. Even an empty
	// or malformed present header is denied, including manually built requests.
	for key := range header {
		if strings.EqualFold(key, "Origin") {
			return true
		}
	}
	return false
}
