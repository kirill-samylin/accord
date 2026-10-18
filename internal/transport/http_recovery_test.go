package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHeartbeatAfterRateLimitRecovery(t *testing.T) {
	f := newFixture(t)
	session := openSession(t, f, "developer", "recover-session")
	handler := NewHandlerWithOptions(f.store, f.store.Ready, HandlerOptions{RequestsPerSecond: 5, Burst: 1})
	if first := nativeRequest(handler, "GET", "/metrics", "", nil); first.Code != 401 {
		t.Fatalf("first unauthenticated request HTTP %d", first.Code)
	}
	if limited := nativeRequest(handler, "GET", "/metrics", "", nil); limited.Code != 429 {
		t.Fatalf("isolated limit was not exercised: %d", limited.Code)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/heartbeat", strings.NewReader(`{"request_id":"after-rate-limit"}`))
		req.Header.Set("Authorization", "Bearer "+f.token)
		req.Header.Set("X-Session", session.SessionID)
		req.Header.Set("Content-Type", "application/json")
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if result.Code == http.StatusOK {
			if !strings.Contains(result.Body.String(), session.SessionID) {
				t.Fatal("recovered heartbeat returned a different session")
			}
			return
		}
		if result.Code != 429 || time.Now().After(deadline) {
			t.Fatalf("heartbeat failed after limiter recovery: %d %s", result.Code, result.Body.String())
		}
		time.Sleep(25 * time.Millisecond)
	}
}
