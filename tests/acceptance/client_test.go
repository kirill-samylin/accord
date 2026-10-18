package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

type payload map[string]any

type client struct {
	b       *board
	token   string
	tokenID string
	session string
	seq     int
}

type result struct {
	data []byte
	err  error
}

func command(input []byte, args ...string) result {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = bytes.NewReader(input)
	// A token from the invoking shell must not affect these synthetic clients.
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "AGENT_BOARD_TOKEN=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	data, err := cmd.Output()
	return result{data: data, err: err}
}

func newClient(t *testing.T, b *board, name string) *client {
	t.Helper()
	c := &client{b: b, token: filepath.Join(filepath.Dir(b.db), name+".token")}
	r := command(nil, "token", "create", "--db", b.db, "--owner", name, "--machine", name, "--output", c.token)
	if r.err != nil {
		t.Fatal("local token create failed")
	}
	info, err := os.Stat(c.token)
	if err != nil || info.Mode().Perm() != 0600 || !info.Mode().IsRegular() {
		t.Fatal("token file is not a private regular 0600 file")
	}
	secret, err := os.ReadFile(c.token)
	if err != nil || len(bytes.TrimSpace(secret)) == 0 {
		t.Fatal("private token is empty or unreadable")
	}
	if bytes.Contains(r.data, bytes.TrimSpace(secret)) {
		t.Fatal("token create disclosed secret on stdout")
	}
	fields := strings.Fields(string(r.data))
	if len(fields) < 2 || fields[0] != "Token" {
		t.Fatal("token create omitted token identifier")
	}
	c.tokenID = fields[1]
	database, err := os.Stat(b.db)
	if err != nil || !database.Mode().IsRegular() || database.Mode().Perm() != 0600 {
		t.Fatal("CLI bootstrap did not create a private regular SQLite file")
	}
	return c
}

func (c *client) open(t *testing.T, role, machine string) core.SessionView {
	t.Helper()
	view := decode[core.SessionView](t, c.ok(t, "describe", payload{"role": role, "run_id": machine + "-" + role, "machine": machine}))
	if view.SessionID == "" || view.Role != role {
		t.Fatal("describe did not return the requested board session")
	}
	c.session = view.SessionID
	return view
}

func (c *client) prepare(t *testing.T, input payload) []byte {
	t.Helper()
	if c.session != "" {
		input["session_id"] = c.session
	}
	if _, exists := input["request_id"]; !exists {
		c.seq++
		input["request_id"] = fmt.Sprintf("request-%d", c.seq)
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal("cannot encode synthetic request")
	}
	return data
}

func (c *client) raw(operation string, input []byte) result {
	return command(input, "call", operation, "--server", c.b.url, "--token-file", c.token)
}

func (c *client) ok(t *testing.T, operation string, input payload) []byte {
	t.Helper()
	if operation == "get" || operation == "events" || operation == "tasks" || operation == "epics" || operation == "sessions" || operation == "notes" || operation == "stats" {
		input["session_id"] = c.session
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal("cannot encode read request")
		}
		r := c.raw(operation, data)
		if r.err != nil {
			t.Fatalf("CLI %s failed (response suppressed)", operation)
		}
		return r.data
	}
	r := c.raw(operation, c.prepare(t, input))
	if r.err != nil {
		t.Fatalf("CLI %s failed (response suppressed)", operation)
	}
	return r.data
}

func (c *client) fails(t *testing.T, operation, code string, input payload) {
	t.Helper()
	r := c.raw(operation, c.prepare(t, input))
	if r.err == nil {
		t.Fatalf("CLI %s unexpectedly succeeded; expected %s", operation, code)
	}
	if domainCode(t, r.data) != code {
		t.Fatalf("CLI %s did not return expected domain code %s", operation, code)
	}
}

func domainCode(t *testing.T, data []byte) string {
	t.Helper()
	response := decode[struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}](t, data)
	return response.Error.Code
}

func decode[T any](t *testing.T, data []byte) T {
	t.Helper()
	var value T
	if json.Unmarshal(data, &value) != nil {
		t.Fatal("invalid JSON response (body suppressed)")
	}
	return value
}

func (c *client) epic(t *testing.T) core.Task {
	t.Helper()
	return decode[core.Task](t, c.ok(t, "create", payload{"type": "epic", "title": "Synthetic pilot", "description": "Process acceptance"}))
}

func (c *client) task(t *testing.T, epic, role string) core.Task {
	t.Helper()
	return decode[core.Task](t, c.ok(t, "create", payload{"type": "task", "epic_key": epic, "role": role, "title": "Synthetic slice", "description": "Process acceptance", "priority": 1}))
}

func (c *client) claim(t *testing.T, key string) core.Claim {
	t.Helper()
	claim := decode[core.Claim](t, c.ok(t, "next", payload{"task_key": key}))
	if claim.Task == nil || claim.Lease == nil || claim.Task.Key != key || claim.Lease.Generation < 1 {
		t.Fatal("next did not claim the requested task")
	}
	return claim
}
