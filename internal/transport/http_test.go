package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/store"
)

type fixture struct {
	store  *store.Store
	server *httptest.Server
	token  string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "board.db"), core.Config{SessionTTL: 15 * time.Minute, LeaseTTL: 15 * time.Minute}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	token, err := s.CreateToken(context.Background(), core.TokenInput{Owner: "test", Machine: "test-machine", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewHandler(s, s.Ready))
	t.Cleanup(server.Close)
	return fixture{s, server, token.Secret}
}
func request(t *testing.T, f fixture, method, path, session, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("X-Session", session)
	if method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, data
}
func parseResult[T any](t *testing.T, status int, data []byte) T {
	t.Helper()
	if status != 200 {
		t.Fatalf("HTTP %d: %s", status, data)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func openSession(t *testing.T, f fixture, role, id string) core.SessionView {
	t.Helper()
	body, _ := json.Marshal(core.OpenSession{RequestID: id, Role: role, RunID: id, Machine: "test-machine"})
	status, data := request(t, f, "POST", "/api/v1/session", "", string(body))
	return parseResult[core.SessionView](t, status, data)
}

func TestRESTValidationAndAuthorization(t *testing.T) {
	f := newFixture(t)
	coordinator := openSession(t, f, "coordinator", "open-1")
	tests := []struct {
		name, method, path, body, session string
		status                            int
	}{
		{"unknown principal", "POST", "/api/v1/tasks", `{"request_id":"1","type":"epic","title":"E","principal":{"id":"forged"}}`, coordinator.SessionID, 400},
		{"trailing JSON", "POST", "/api/v1/tasks", `{} {}`, coordinator.SessionID, 400},
		{"null body", "POST", "/api/v1/tasks", `null`, coordinator.SessionID, 400},
		{"missing session", "GET", "/api/v1/tasks/E-1", "", "", 404},
		{"bad cursor", "GET", "/api/v1/events?since_cursor=x", "", coordinator.SessionID, 400},
		{"repeated cursor", "GET", "/api/v1/events?since_cursor=0&since_cursor=1", "", coordinator.SessionID, 400},
		{"mismatched key", "PATCH", "/api/v1/tasks/T-1", `{"task_key":"T-2"}`, coordinator.SessionID, 400},
		{"body limit", "POST", "/api/v1/tasks", strings.Repeat("x", int(MaxBodyBytes)+1), coordinator.SessionID, 400},
		{"wrong method", "DELETE", "/api/v1/tasks/E-1", "", coordinator.SessionID, 405},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := request(t, f, tt.method, tt.path, tt.session, tt.body)
			if status != tt.status {
				t.Fatalf("HTTP %d, want %d", status, tt.status)
			}
		})
	}
	status, body := request(t, f, "POST", "/api/v1/session", "", `{"request_id":"bad-machine","role":"developer","run_id":"run","machine":"other"}`)
	if status != 400 {
		t.Fatalf("machine binding: HTTP %d %s", status, body)
	}
	req, _ := http.NewRequest("POST", f.server.URL+"/api/v1/session", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer must-never-echo")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != 401 || bytes.Contains(data, []byte("must-never-echo")) {
		t.Fatal("invalid token was not safely rejected")
	}
}

func TestRESTPersistenceReplayAndOrigin(t *testing.T) {
	f := newFixture(t)
	session := openSession(t, f, "coordinator", "open")
	create := `{"request_id":"epic","type":"epic","title":"One","description":"Purpose"}`
	status, data := request(t, f, "POST", "/api/v1/tasks", session.SessionID, create)
	epic := parseResult[core.Task](t, status, data)
	status, replayed := request(t, f, "POST", "/api/v1/tasks", session.SessionID, create)
	if status != 200 || !bytes.Equal(data, replayed) {
		t.Fatalf("replay changed response: %d", status)
	}
	status, get := request(t, f, "GET", "/api/v1/tasks/"+epic.Key, session.SessionID, "")
	if parseResult[core.Task](t, status, get).Key != epic.Key {
		t.Fatal("read did not return created epic")
	}
	req, _ := http.NewRequest("POST", f.server.URL+"/api/v1/session", strings.NewReader(create))
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Origin", "https://untrusted.example")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatalf("cross-origin POST HTTP %d", response.StatusCode)
	}
}

func TestLoopbackOnly(t *testing.T) {
	for _, address := range []string{"0.0.0.0:8080", ":8080", "localhost:8080", "192.0.2.1:8080"} {
		if ValidateListen(address) == nil {
			t.Fatal("accepted", address)
		}
	}
	for _, address := range []string{"127.0.0.1:8080", "[::1]:8080"} {
		if err := ValidateListen(address); err != nil {
			t.Fatal(err)
		}
	}
}
