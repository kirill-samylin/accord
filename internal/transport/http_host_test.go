package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"agent-board/internal/core"
)

func TestExternalHostValidation(t *testing.T) {
	for _, host := range []string{"", "board.example:8443", "board.example:443", "board-name.tail.example:65535"} {
		if err := ValidateExternalHost(host); err != nil {
			t.Fatalf("valid authority %q: %v", host, err)
		}
	}
	for _, host := range []string{
		"board.example", "Board.example:8443", "board.example.:8443", "*.example:8443", "https://board.example:8443",
		"user@board.example:8443", "board.example:8443/path", "board.example:8443?x", "board.example:8443#x",
		" board.example:8443", "board.example:8443\n", "board.example:08443", "board.example:+8443",
		"board.example:0", "board.example:65536", "board.example:", "board.example:8443,evil.example:8443",
		"127.0.0.1:8443", "localhost:8443", "127.1:8443", "0x7f000001:8443", "[::1]:8443", "[board.example]:8443",
		"bad_label.example:8443", "-bad.example:8443", "bad-.example:8443", "a..example:8443", "пример.рф:8443",
	} {
		if err := ValidateExternalHost(host); err == nil {
			t.Errorf("unsafe authority accepted: %q", host)
		}
	}
}

func TestExactHostPolicyBeforeAuthentication(t *testing.T) {
	var calls atomic.Int64
	service := authProbe{call: func(context.Context, string) (core.Principal, error) {
		calls.Add(1)
		return core.Principal{}, nil
	}}
	handler := NewHandlerWithOptions(service, nil, HandlerOptions{ExternalHost: "board.example:8443"})
	for _, host := range []string{"board.example:8443", "127.0.0.1:7892", "127.0.0.2", "localhost", "[::1]:7892", "[::1]"} {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.Host = host
		req.Header.Set("Authorization", "Bearer fixture")
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if result.Code != 200 {
			t.Fatalf("allowed authority %q: %d", host, result.Code)
		}
	}
	before := calls.Load()
	for _, host := range []string{"", "evil.example:8443", "board.example", "board.example:443", "Board.example:8443", "board.example.:8443", "127.1", "127.0.0.01", "[127.0.0.1]:7892", "[0:0:0:0:0:0:0:1]:7892", "localhost:00080", "localhost:0"} {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.Host = host
		req.Header.Set("Authorization", "Bearer fixture")
		req.Header.Set("X-Forwarded-Host", "board.example:8443")
		req.Header.Set("X-Forwarded-Proto", "https")
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if result.Code != 403 {
			t.Errorf("unsafe authority %q: %d", host, result.Code)
		}
	}
	for _, duplicate := range []string{"Host", "hOsT"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1/metrics", nil)
		req.Header[duplicate] = []string{"board.example:8443"}
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if result.Code != 403 {
			t.Fatal("duplicate authority accepted")
		}
	}
	req := httptest.NewRequest("GET", "http://evil.example/metrics", nil)
	req.Host = "board.example:8443"
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	if result.Code != 403 || calls.Load() != before {
		t.Fatal("unsafe/absolute-form authority reached authentication")
	}
}

func TestHostPolicyDefaultsAndInvalidConfigurationFailClosed(t *testing.T) {
	for _, option := range []HandlerOptions{{}, {ExternalHost: "*.example:8443"}} {
		handler := NewHandlerWithOptions(authProbe{}, nil, option)
		req := httptest.NewRequest("GET", "http://board.example:8443/healthz", nil)
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if result.Code < 400 {
			t.Fatal("external authority/configuration failed open")
		}
	}
}

// A real loopback HTTP listener supplies LocalAddrContextKey, reproducing the
// SDK guard that a ResponseRecorder alone cannot exercise behind a TLS proxy.
func TestMCPAdvertisedHostOnLoopbackListener(t *testing.T) {
	f := newFixture(t)
	server := httptest.NewServer(NewHandlerWithOptions(f.store, f.store.Ready, HandlerOptions{ExternalHost: "board.example:8443"}))
	defer server.Close()
	for _, test := range []struct {
		host, token, origin string
		status              int
	}{
		{"board.example:8443", f.token, "", 200},
		{"127.0.0.1:7892", f.token, "", 200},
		{"board.example:443", f.token, "", 403},
		{"evil.example:8443", f.token, "", 403},
		{"board.example:8443", "", "", 401},
		{"board.example:8443", f.token, "https://board.example:8443", 403},
	} {
		req, err := http.NewRequest("POST", server.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"host-regression","version":"1"}}}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Host = test.host
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if test.token != "" {
			req.Header.Set("Authorization", "Bearer "+test.token)
		}
		if test.origin != "" {
			req.Header.Set("Origin", test.origin)
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != test.status {
			t.Fatalf("authority %q: HTTP %d, expected %d: %s", test.host, response.StatusCode, test.status, body)
		}
		if test.status == 200 && !strings.Contains(string(body), `"protocolVersion":"2025-11-25"`) {
			t.Fatalf("initialize did not negotiate protocol: %s", body)
		}
	}
}
