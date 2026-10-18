package transport

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestMetricsAuthenticatedWithoutBoardSession(t *testing.T) {
	f := newFixture(t)
	for _, token := range []string{"", "invalid-private-test-token", f.token} {
		req, err := http.NewRequest("GET", f.server.URL+"/metrics", nil)
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		want := http.StatusUnauthorized
		if token == f.token {
			want = http.StatusOK
		}
		if response.StatusCode != want {
			t.Fatalf("metrics HTTP %d, want %d", response.StatusCode, want)
		}
		for _, secret := range []string{f.token, "invalid-private-test-token", "test-machine"} {
			if strings.Contains(string(data), secret) {
				t.Fatal("metrics response exposed a credential or identity")
			}
		}
		if want == 200 {
			if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/plain; version=0.0.4") {
				t.Fatal("metrics has incorrect content type")
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if len(lines) != 14 || strings.Contains(string(data), "{") {
				t.Fatal("metrics added unbounded series or labels")
			}
			for _, name := range []string{"uptime_seconds", "http_requests_total", "http_rate_limited_total", "http_concurrency_rejected_total", "http_active_requests", "runtime_heap_bytes", "runtime_sys_bytes"} {
				if !strings.Contains(string(data), "\naccord_"+name+" ") {
					t.Fatalf("missing metric %s", name)
				}
			}
		}
	}
	for _, path := range []string{"/metrics?token=do-not-echo", "/metrics?session=private"} {
		status, data := request(t, f, "GET", path, "", "")
		if status != 400 || strings.Contains(string(data), "do-not-echo") || strings.Contains(string(data), "private") {
			t.Fatal("metrics accepted/echoed unknown query")
		}
	}
	if status, _ := request(t, f, "POST", "/metrics", "", ""); status != 405 {
		t.Fatalf("metrics POST HTTP %d", status)
	}
}

func TestMetricsRejectsRevokedBearer(t *testing.T) {
	f := newFixture(t)
	token, err := f.store.CreateToken(context.Background(), core.TokenInput{Owner: "synthetic", Machine: "metrics-test", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	f.token = token.Secret
	if status, _ := request(t, f, "GET", "/metrics", "", ""); status != 200 {
		t.Fatalf("new metrics credential denied: %d", status)
	}
	if err := f.store.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	if status, _ := request(t, f, "GET", "/metrics", "", ""); status != 401 {
		t.Fatalf("revoked metrics credential accepted: %d", status)
	}
}

func TestNativeSecurityHeadersAndMinimalHealth(t *testing.T) {
	handler := NewHandler(authProbe{call: func(context.Context, string) (core.Principal, error) {
		return core.Principal{}, nil
	}}, nil)
	for _, path := range []string{"/api/v1/events", "/mcp", "/metrics"} {
		result := nativeRequest(handler, "GET", path, "", nil)
		if result.Code != 401 {
			t.Fatalf("%s without bearer returned %d", path, result.Code)
		}
		for name, expected := range map[string]string{
			"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
			"Referrer-Policy": "no-referrer", "Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		} {
			if result.Header().Get(name) != expected {
				t.Fatalf("%s missing security header %s", path, name)
			}
		}
		if result.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("native endpoint exposed CORS")
		}
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		result := nativeRequest(handler, "GET", path, "", nil)
		if result.Code != 200 || result.Body.String() != "{\"status\":\"ok\"}\n" {
			t.Fatalf("minimal unauthenticated %s changed", path)
		}
	}
}
