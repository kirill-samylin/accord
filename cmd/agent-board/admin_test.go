package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminCommandsRefuseMissingDatabase(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "missing.db")
	commands := [][]string{
		{"token", "list", "--db", db},
		{"token", "revoke", "--db", db, "--id", "unknown"},
		{"token", "revoke-all", "--db", db, "--reason", "restore"},
		{"sessions", "list", "--db", db},
		{"maintenance", "reset", "--db", db, "--reason", "restore"},
		{"backup", "--db", db, "--output", filepath.Join(dir, "snapshot.db")},
	}
	for _, args := range commands {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err == nil {
			t.Fatalf("missing database accepted by %s", args[0])
		}
		if _, err := os.Stat(db); !os.IsNotExist(err) {
			t.Fatalf("command created missing database: %v", args)
		}
	}
}

func TestAdminCLIBackupAndRecovery(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "board.db")
	secretPath := filepath.Join(dir, "owner.token")
	var out bytes.Buffer
	if err := run([]string{"token", "create", "--db", db, "--owner", "owner", "--machine", "mac", "--output", secretPath}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err = run([]string{"token", "list", "--db", db}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), strings.TrimSpace(string(secret))) || strings.Contains(out.String(), "hash") || strings.Contains(out.String(), "secret") {
		t.Fatal("token list leaked credential")
	}
	var tokens []struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(out.Bytes(), &tokens); err != nil || len(tokens) != 1 || tokens[0].ID == "" {
		t.Fatalf("token listing not usable %v", err)
	}
	snapshot := filepath.Join(dir, "backup.db")
	out.Reset()
	if err = run([]string{"backup", "--db", db, "--output", snapshot}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		SHA256 string `json:"sha256"`
	}
	if err = json.Unmarshal(out.Bytes(), &metadata); err != nil || len(metadata.SHA256) != 64 {
		t.Fatalf("invalid backup metadata %v", err)
	}
	out.Reset()
	if err = run([]string{"maintenance", "reset", "--db", snapshot, "--reason", "restore"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err = run([]string{"token", "list", "--db", snapshot}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"revoked":true`) {
		t.Fatal("recovered token was not revoked")
	}
}
