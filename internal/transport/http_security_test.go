package transport

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-board/internal/core"
)

type authProbe struct {
	Service
	call func(context.Context, string) (core.Principal, error)
}

func (p authProbe) Authenticate(ctx context.Context, token string) (core.Principal, error) {
	return p.call(ctx, token)
}

func nativeRequest(handler http.Handler, method, path, token string, headers http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1"+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, values := range headers {
		req.Header[key] = values
	}
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	return result
}

func TestNativeOriginPresencePolicy(t *testing.T) {
	var authenticated atomic.Int64
	service := authProbe{call: func(context.Context, string) (core.Principal, error) {
		authenticated.Add(1)
		return core.Principal{}, nil
	}}
	handler := NewHandler(service, nil)
	for _, path := range []string{"/api/v1/events", "/api/v1/session", "/mcp", "/metrics"} {
		for _, method := range []string{"GET", "POST"} {
			for _, origin := range []string{"https://foreign.example", "null", "://broken", "", "http://example.com"} {
				result := nativeRequest(handler, method, path, "fixture", http.Header{
					"Origin": {origin}, "X-Forwarded-Host": {"foreign.example"}, "X-Forwarded-Proto": {"https"},
				})
				if result.Code != http.StatusForbidden {
					t.Fatalf("%s %s Origin %q: HTTP %d", method, path, origin, result.Code)
				}
			}
		}
	}
	for _, values := range [][]string{nil, {"https://one.example", "https://two.example"}} {
		if result := nativeRequest(handler, "GET", "/metrics", "fixture", http.Header{"oRiGiN": values}); result.Code != 403 {
			t.Fatalf("noncanonical/present Origin accepted: %d", result.Code)
		}
	}
	if authenticated.Load() != 0 {
		t.Fatal("Origin rejection reached authentication")
	}
	if result := nativeRequest(handler, "GET", "/metrics", "fixture", nil); result.Code != 200 {
		t.Fatalf("native client denied: %d", result.Code)
	}
	if result := nativeRequest(handler, "GET", "/future-web", "fixture", http.Header{"Origin": {"https://foreign.example"}}); result.Code == 403 {
		t.Fatal("native policy was incorrectly applied to a future web route")
	}
}

func TestGlobalRateLimitBeforeAuthentication(t *testing.T) {
	var calls atomic.Int64
	service := authProbe{call: func(context.Context, string) (core.Principal, error) {
		calls.Add(1)
		return core.Principal{}, errors.New("must not echo credential")
	}}
	handler := NewHandlerWithOptions(service, nil, HandlerOptions{RequestsPerSecond: 0.001, Burst: 2})
	for i := 0; i < 8; i++ {
		result := nativeRequest(handler, "GET", "/metrics", strings.Repeat("x", i+1), nil)
		want := http.StatusTooManyRequests
		if i < 2 {
			want = http.StatusUnauthorized
		}
		if result.Code != want {
			t.Fatalf("request %d: HTTP %d, want %d", i, result.Code, want)
		}
		if i >= 2 && result.Header().Get("Retry-After") == "" {
			t.Fatal("rate rejection omitted Retry-After")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("limited requests reached expensive authentication: %d", calls.Load())
	}
}

func TestBucketRefillAndBoundedRecovery(t *testing.T) {
	a := newAdmission(HandlerOptions{RequestsPerSecond: 2, Burst: 1})
	now := a.last
	if ok, _ := a.take(now); !ok {
		t.Fatal("initial token missing")
	}
	for i := 0; i < 1000; i++ {
		if ok, wait := a.take(now); ok || wait != 0.5 {
			t.Fatal("rejections reserved future tokens or changed delay")
		}
	}
	if ok, _ := a.take(now.Add(500 * time.Millisecond)); !ok {
		t.Fatal("limiter failed to recover after refill")
	}
	if ok, _ := a.take(now.Add(time.Hour)); !ok {
		t.Fatal("idle bucket did not refill")
	}
	if ok, _ := a.take(now.Add(time.Hour)); ok {
		t.Fatal("idle bucket accumulated beyond burst")
	}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		defaults := newAdmission(HandlerOptions{RequestsPerSecond: rate})
		if defaults.rate != 100 || defaults.burst != 200 || cap(defaults.inflight) != 64 {
			t.Fatal("invalid/zero options bypassed safe defaults")
		}
	}
}

func TestConcurrentAdmissionAndPanicCleanup(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	service := authProbe{call: func(context.Context, string) (core.Principal, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			panic("secret-from-panic")
		}
		return core.Principal{}, nil
	}}
	handler := NewHandlerWithOptions(service, nil, HandlerOptions{MaxConcurrent: 1})
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() { completed <- nativeRequest(handler, "GET", "/metrics", "fixture", nil) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not reach authentication")
	}
	result := nativeRequest(handler, "GET", "/metrics", "fixture", nil)
	if result.Code != 503 || result.Header().Get("Retry-After") != "1" || calls.Load() != 1 {
		t.Fatal("concurrent request was queued or reached authentication")
	}
	close(release)
	result = <-completed
	if result.Code != 500 || strings.Contains(result.Body.String(), "secret-from-panic") {
		t.Fatal("panic error was not safely redacted")
	}
	result = nativeRequest(handler, "GET", "/metrics", "fixture", nil)
	if result.Code != 200 || !strings.Contains(result.Body.String(), "accord_http_active_requests 1\n") {
		t.Fatal("panic leaked concurrency capacity")
	}
}
