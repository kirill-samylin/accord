package main

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	webserver "agent-board/internal/web"
	"agent-board/internal/webui"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProjectCLIExplicitBootstrapCredentialAndRestoreRecovery(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	db := filepath.Join(dir, "board.db")
	web := filepath.Join(dir, "web")
	root := filepath.Join(dir, "projects")
	s, e := store.Open(db, core.Config{Project: "existing"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	old, e := s.CreateToken(context.Background(), core.TokenInput{Owner: "operator", Machine: "machine", TTL: time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	renderer, e := webui.New()
	if e != nil {
		t.Fatal(e)
	}
	browser, e := webserver.New(s, webserver.Options{Origin: "http://127.0.0.1:8080", DataDir: web, Renderer: renderer})
	if e != nil {
		t.Fatal(e)
	}
	browser.Close()
	s.Close()
	var out bytes.Buffer
	base := []string{"--projects-data", root, "--db", db, "--web-data", web}
	for i := 0; i < 2; i++ {
		args := append([]string{"projects", "bootstrap-owner"}, base...)
		args = append(args, "--identity", old.ID)
		out.Reset()
		if e = run(args, strings.NewReader(""), &out); e != nil {
			t.Fatal("offline bootstrap", e)
		}
	}
	tokenPath := filepath.Join(dir, "catalog.token")
	args := append([]string{"projects", "token-create"}, base...)
	args = append(args, "--owner", "catalog", "--machine", "machine", "--output", tokenPath)
	out.Reset()
	if e = run(args, strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
	secret, e := os.ReadFile(tokenPath)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out.String(), strings.TrimSpace(string(secret))) {
		t.Fatal("CLI output exposed secret")
	}
	var created struct {
		Identity struct {
			ID string `json:"id"`
		}
	}
	if e = json.Unmarshal(out.Bytes(), &created); e != nil || created.Identity.ID == "" {
		t.Fatal("missing public identity", e)
	}
	var bootstrap struct {
		Default string `json:"default_project"`
	}
	bootstrapArgs := append([]string{"projects", "bootstrap-owner"}, base...)
	bootstrapArgs = append(bootstrapArgs, "--identity", old.ID)
	out.Reset()
	if e = run(bootstrapArgs, strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(out.Bytes(), &bootstrap)
	recover := append([]string{"projects", "recover-owner"}, base...)
	recover = append(recover, "--project", bootstrap.Default, "--identity", created.Identity.ID, "--reason", "owner-recovery")
	if e = run(recover, strings.NewReader(""), &out); e != nil {
		t.Fatal("offline owner recovery", e)
	}
	// No web server or renderer is constructed by trusted offline recovery.
	args = append([]string{"projects", "maintenance-reset"}, base...)
	args = append(args, "--reason", "restore")
	out.Reset()
	if e = run(args, strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), `"restored_state_reset":true`) {
		t.Fatal("reset success not explicit")
	}
}
func TestCapabilitiesMachineJSON(t *testing.T) {
	var out bytes.Buffer
	if e := run([]string{"capabilities", "--json"}, strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
	var result struct {
		Registry int   `json:"project_registry"`
		ACL      int   `json:"project_acl"`
		Schemas  []int `json:"registry_schema"`
	}
	if e := json.Unmarshal(out.Bytes(), &result); e != nil || result.Registry != 1 || result.ACL != 1 || len(result.Schemas) != 1 || result.Schemas[0] != 1 {
		t.Fatal("capability contract", e, out.String())
	}
}
