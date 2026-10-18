package main

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"agent-board/internal/store"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCLIInitialApplyHistoryValidationAndScopes(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "board.db")
	file := filepath.Join(dir, "config.yaml")
	var out bytes.Buffer
	if err := run([]string{"config", "init", "--profile", "davinci"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	raw := bytes.ReplaceAll(out.Bytes(), []byte("OWNER"), []byte("alice"))
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"config", "validate", file}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var validation config.Result
	if err := json.Unmarshal(out.Bytes(), &validation); err != nil || !validation.Valid {
		t.Fatal(validation, err)
	}
	args := []string{"config", "apply", "--db", db, "--file", file, "--expected-version", "0", "--actor", "operator", "--bootstrap-owner", "alice", "--project", "accord", "--task-ttl", "15m", "--session-ttl", "15m"}
	out.Reset()
	if err := run(args, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var view core.ConfigView
	if err := json.Unmarshal(out.Bytes(), &view); err != nil || view.ConfigVersion != 1 {
		t.Fatal(view, err)
	}
	if err := run(args, strings.NewReader(""), &out); err == nil {
		t.Fatal("stale initial apply succeeded")
	}
	out.Reset()
	if err := run([]string{"config", "history", "--db", db}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var history core.ConfigHistory
	if err := json.Unmarshal(out.Bytes(), &history); err != nil || len(history.Revisions) != 1 {
		t.Fatal(history, err)
	}
	path := filepath.Join(dir, "alice.token")
	out.Reset()
	if err := run([]string{"token", "create", "--db", db, "--owner", "alice", "--machine", "pc", "--output", path, "--roles", "human"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	s, err := store.OpenExisting(db, core.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tokens, err := s.ListTokens(context.Background())
	if err != nil || len(tokens) != 1 || len(tokens[0].Roles) != 1 || tokens[0].Roles[0] != "human" {
		t.Fatal(tokens, err)
	}
	if err = os.WriteFile(file, []byte("version: 99"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"config", "validate", file}, strings.NewReader(""), &out); err == nil {
		t.Fatal("invalid validate returned success")
	}
}
