package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
)

func TestTokenFileAndRevoke(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "board.db")
	path := filepath.Join(dir, "owner.token")
	var output bytes.Buffer
	args := []string{"token", "create", "--db", db, "--owner", "test-owner", "--machine", "test-machine", "--output", path}
	if err := run(args, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	secret, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if output.Len() == 0 || bytes.Contains(output.Bytes(), bytes.TrimSpace(secret)) {
		t.Fatal("status missing or leaked secret")
	}
	if err := run(args, strings.NewReader(""), &output); err == nil {
		t.Fatal("token output was overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, secret) {
		t.Fatal("existing token file changed")
	}
	s, err := store.Open(db, core.Config{BootstrapUsers: []string{"alice"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	principal, err := s.Authenticate(context.Background(), strings.TrimSpace(string(secret)))
	if err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"token", "revoke", "--db", db, "--id", principal.ID}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(context.Background(), strings.TrimSpace(string(secret))); err == nil {
		t.Fatal("revoked token still authenticates")
	}
}

func TestCallDescribeCreateGetAndEvents(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "board.db"), core.Config{BootstrapUsers: []string{"alice"}, LeaseTTL: time.Minute, SessionTTL: time.Minute}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	token, err := s.CreateToken(context.Background(), core.TokenInput{Owner: "test", Machine: "local-test", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.token")
	if err := os.WriteFile(path, []byte(token.Secret), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(transport.NewHandler(s, s.Ready))
	defer server.Close()
	invoke := func(operation string, input any, out any) {
		t.Helper()
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		var result bytes.Buffer
		if err := run([]string{"call", operation, "--server", server.URL, "--token-file", path}, bytes.NewReader(data), &result); err != nil {
			t.Fatalf("%s: %v %s", operation, err, result.String())
		}
		if err := json.Unmarshal(result.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	var session core.SessionView
	invoke("describe", core.OpenSession{RequestID: "cli-open", Role: "coordinator", RunID: "cli-test", Machine: "local-test"}, &session)
	var epic core.Task
	invoke("create", map[string]string{"session_id": session.SessionID, "request_id": "cli-epic", "type": "epic", "title": "CLI", "description": "Test"}, &epic)
	var got core.Task
	invoke("get", map[string]string{"session_id": session.SessionID, "task_key": epic.Key}, &got)
	if got.Key != epic.Key {
		t.Fatal("get returned another item")
	}
	var page core.EventsPage
	invoke("events", map[string]any{"session_id": session.SessionID, "since_cursor": 0, "limit": 1}, &page)
	if len(page.Events) != 1 || !page.HasMore {
		t.Fatal("events did not preserve pagination")
	}
	var epics core.TasksPage
	invoke("epics", map[string]any{"session_id": session.SessionID, "limit": 1}, &epics)
	if len(epics.Tasks) != 1 || epics.Tasks[0].Key != epic.Key {
		t.Fatal("CLI epic page differs from created state")
	}
	var task core.Task
	invoke("create", map[string]string{"session_id": session.SessionID, "request_id": "cli-task", "type": "task",
		"epic_key": epic.Key, "role": "coordinator", "title": "CLI task", "description": "Synthetic"}, &task)
	var note core.Note
	invoke("note", map[string]string{"session_id": session.SessionID, "request_id": "cli-note", "action": "create",
		"scope_type": "role", "scope_id": "coordinator", "source_task_key": task.Key, "text": "Synthetic CLI note"}, &note)
	var notes core.NotesPage
	invoke("notes", map[string]string{"session_id": session.SessionID, "scope_type": "role", "scope_id": "coordinator"}, &notes)
	if len(notes.Notes) != 1 || notes.Notes[0].Key != note.Key {
		t.Fatal("CLI note page differs from mutation")
	}
	var stats core.Stats
	invoke("stats", map[string]string{"session_id": session.SessionID, "from": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		"to": time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), "group_by": "role"}, &stats)
	if stats.Totals.CurrentQueue.Ready != 1 {
		t.Fatal("CLI stats did not read the current queue")
	}
}

func TestTokenFilePrivacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "public.token")
	if err := os.WriteFile(path, []byte("synthetic-test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(path); err == nil {
		t.Fatal("accepted public token file")
	}
}

func TestCallResponseLimit(t *testing.T) {
	t.Setenv("AGENT_BOARD_TOKEN", "synthetic-response-limit")
	for _, size := range []int{maxResponseBytes - 1, maxResponseBytes, maxResponseBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			// Valid JSON at the boundary catches silent successful truncation.
			payload := `"` + strings.Repeat("x", size-2) + `"`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, payload)
			}))
			defer server.Close()
			var output bytes.Buffer
			err := call([]string{"describe", "--server", server.URL}, strings.NewReader(`{}`), &output)
			if size > maxResponseBytes {
				if err == nil || !strings.Contains(err.Error(), "ownership may be uncertain") || output.Len() != 0 {
					t.Fatal("oversized response was not rejected before output with ownership guidance")
				}
			} else if err != nil || output.String() != payload || !json.Valid(output.Bytes()) {
				t.Fatalf("response within limit rejected or changed: %v", err)
			}
		})
	}
}
