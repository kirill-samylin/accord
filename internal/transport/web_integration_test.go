package transport

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestWebExtraStaysInsideHostAdmissionAndNativeOriginBoundary(t *testing.T) {
	var calls atomic.Int64
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) })
	h := NewHandlerWithOptions(authProbe{}, nil, HandlerOptions{Extra: extra})
	for _, tc := range []struct {
		path, host string
		origin     bool
		want       int
	}{
		{"/web/login", "127.0.0.1", false, 204},
		{"/", "127.0.0.1", false, 204},
		{"/api/v1/designs", "127.0.0.1", false, 204},
		{"/web/login", "evil.example", false, 403},
		{"/api/v1/designs", "127.0.0.1", true, 403},
		{"/api/v1/tasks", "127.0.0.1", false, 401},
		{"/metrics", "127.0.0.1", false, 401},
		{"/mcp", "127.0.0.1", true, 403},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Host = tc.host
		if tc.origin {
			r.Header["Origin"] = []string{""}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s: %d want%d", tc.path, w.Code, tc.want)
		}
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("common headers bypassed")
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("extra received protected path: %d", calls.Load())
	}
	limited := NewHandlerWithOptions(authProbe{}, nil, HandlerOptions{Extra: extra, Burst: 1, RequestsPerSecond: 0.000001})
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		limited.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1/web/", nil))
		if i == 1 && w.Code != 429 {
			t.Fatal("web bypassed admission")
		}
	}
}
