package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func configView(t *testing.T, b *board) core.ConfigView {
	t.Helper()
	r := command(nil, "config", "get", "--db", b.db)
	if r.err != nil {
		t.Fatalf("config get: %s", r.stderr)
	}
	return decode[core.ConfigView](t, r.data)
}
func configFile(t *testing.T, b *board, edit func(*config.Config)) (string, int64) {
	t.Helper()
	v := configView(t, b)
	c := decode[config.Config](t, v.Config)
	edit(&c)
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(filepath.Dir(b.db), "candidate-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Chmod(0600); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name(), v.ConfigVersion
}
func applyFile(b *board, path string, v int64) result {
	return command(nil, "config", "apply", "--db", b.db, "--file", path, "--expected-version", fmt.Sprint(v), "--actor", "qa056")
}
func applyConfig(t *testing.T, b *board, edit func(*config.Config)) core.ConfigView {
	t.Helper()
	path, v := configFile(t, b, edit)
	r := applyFile(b, path, v)
	if r.err != nil {
		t.Fatalf("config apply: %s", r.stderr)
	}
	return decode[core.ConfigView](t, r.data)
}
func localFails(t *testing.T, r result, code string) {
	t.Helper()
	if r.err == nil || !bytes.Contains(r.stderr, []byte(code)) {
		t.Fatalf("local command expected %s, stderr=%s", code, r.stderr)
	}
}
func configBoard(t *testing.T, edit func(*config.Config)) *board {
	t.Helper()
	b := newBoard(t, "30s")
	b.start()
	b.stop()
	applyConfig(t, b, edit)
	return b
}
func configHTTP(t *testing.T, c *client, method, path string, body []byte, auth bool) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, c.b.url+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if auth {
		secret, err := os.ReadFile(c.token)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+string(bytes.TrimSpace(secret)))
	}
	req.Header.Set("Content-Type", "application/json")
	if c.session != "" {
		req.Header.Set("X-Session", c.session)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}
func eventCountProcess(t *testing.T, b *board, kind string) int {
	t.Helper()
	db := inspectDatabase(t, b.db)
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM events WHERE json_extract(data,'$.type')=?", kind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func mintRoles(t *testing.T, b *board, name, roles string) *client {
	t.Helper()
	c := &client{b: b, token: filepath.Join(filepath.Dir(b.db), name+".token")}
	r := command(nil, "token", "create", "--db", b.db, "--owner", name, "--machine", name, "--roles", roles, "--output", c.token)
	if r.err != nil {
		t.Fatalf("explicit mint: %s", r.stderr)
	}
	fields := bytes.Fields(r.data)
	if len(fields) < 2 {
		t.Fatal("missing mint id")
	}
	c.tokenID = string(fields[1])
	return c
}
func startExistingImplicit(t *testing.T, b *board) int {
	t.Helper()
	b.startOptions()
	return b.cmd.Process.Pid
}
