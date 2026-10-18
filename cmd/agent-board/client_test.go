package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	s, err := store.Open(db, core.Config{}, nil)
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
	s, err := store.Open(filepath.Join(dir, "board.db"), core.Config{LeaseTTL: time.Minute, SessionTTL: time.Minute}, nil)
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
}

func TestTokenFilePrivacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "public.token")
	if err := os.WriteFile(path, []byte("synthetic-test"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(path); err == nil {
		t.Fatal("accepted public token file")
	}
}
