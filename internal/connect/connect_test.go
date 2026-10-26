package connect

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"agent-board/internal/worker"
)

type harness struct {
	home, token, self string
	commands          []string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{home: filepath.Join(root, "home")}
	downloads := filepath.Join(root, "downloads")
	for _, dir := range []string{h.home, downloads} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	h.token = filepath.Join(downloads, "accord.token")
	if err := os.WriteFile(h.token, []byte("synthetic-private-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hints := `{"origin":"https://accord.example/","project_id":"p_00000000000000000000000000000001","machine":"m"}`
	if err := os.WriteFile(filepath.Join(downloads, "connection.json"), []byte(hints), 0600); err != nil {
		t.Fatal(err)
	}
	h.self = filepath.Join(root, "build", "agent-board")
	if err := os.MkdirAll(filepath.Dir(h.self), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.self, []byte("binary-v2"), 0700); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *harness) options(clis []worker.NativeCLI) Options {
	return Options{TokenFile: h.token, Home: h.home, Executable: h.self, CLIs: clis, Out: &bytes.Buffer{}, Run: func(_ context.Context, executable string, args ...string) error {
		h.commands = append(h.commands, filepath.Base(executable)+" "+strings.Join(args, " "))
		if strings.Contains(strings.Join(args, " "), "remove") {
			return os.ErrNotExist // a missing connector is not an error for remove
		}
		return nil
	}}
}

func TestConnectInstallsPrivateCopiesAndRegistersEveryCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions")
	}
	h := newHarness(t)
	gennadiy := filepath.Join(h.home, ".config", "gennadiy", "gennadiy.json")
	if err := os.MkdirAll(filepath.Dir(gennadiy), 0700); err != nil {
		t.Fatal(err)
	}
	original := `{"$schema":"x","model":"gennadiy/m","mcp":{"jira":{"type":"remote","url":"https://jira.corp","headers":{"Authorization":"Bearer corp"},"enabled":true}},"permission":{"edit":"ask"}}`
	if err := os.WriteFile(gennadiy, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	clis := []worker.NativeCLI{{Provider: "codex", Path: "/opt/codex", Version: "codex 1"}, {Provider: "claude", Path: "/opt/claude", Version: "claude 2"}, {Provider: "gennadiy", Path: "/opt/gennadiy-real", Version: "1.18.10"}, {Provider: "codex", Path: "/other/codex"}}
	report, err := Connect(context.Background(), h.options(clis))
	if err != nil {
		t.Fatal(err)
	}
	wantToken := filepath.Join(h.home, ".config", "accord", "p_00000000000000000000000000000001.token")
	wantBinary := filepath.Join(h.home, ".config", "accord", "agent-board")
	if report.TokenPath != wantToken || report.Binary != wantBinary {
		t.Fatalf("report: %+v", report)
	}
	for path, mode := range map[string]os.FileMode{wantToken: 0600, wantBinary: 0700, filepath.Dir(wantToken): 0700} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("%s: %v %v", path, info, err)
		}
	}
	if data, _ := os.ReadFile(wantToken); string(data) != "synthetic-private-token\n" {
		t.Fatalf("token copy: %q", data)
	}
	if data, _ := os.ReadFile(wantBinary); string(data) != "binary-v2" {
		t.Fatalf("binary copy: %q", data)
	}
	mcp := wantBinary + " mcp --server https://accord.example --token-file " + wantToken + " --project p_00000000000000000000000000000001"
	want := []string{"codex mcp remove accord", "codex mcp add accord -- " + mcp, "claude mcp remove --scope user accord", "claude mcp add --scope user accord -- " + mcp}
	if !reflect.DeepEqual(h.commands, want) {
		t.Fatalf("commands:\n%s\nwant:\n%s", strings.Join(h.commands, "\n"), strings.Join(want, "\n"))
	}
	var config map[string]any
	data, _ := os.ReadFile(gennadiy)
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	entries := config["mcp"].(map[string]any)
	jira := entries["jira"].(map[string]any)
	if jira["headers"].(map[string]any)["Authorization"] != "Bearer corp" || config["permission"].(map[string]any)["edit"] != "ask" || config["model"] != "gennadiy/m" {
		t.Fatalf("foreign keys changed: %s", data)
	}
	accord := entries["accord"].(map[string]any)
	if accord["type"] != "local" || accord["enabled"] != true || !reflect.DeepEqual(accord["command"], []any{wantBinary, "mcp", "--server", "https://accord.example", "--token-file", wantToken, "--project", "p_00000000000000000000000000000001"}) {
		t.Fatalf("accord entry: %+v", accord)
	}
	if info, _ := os.Stat(gennadiy); info.Mode().Perm() != 0600 {
		t.Fatal("gennadiy.json mode changed")
	}
	for _, entry := range report.Entries {
		if entry.Status != "configured" {
			t.Fatalf("entry: %+v", entry)
		}
	}
	var out bytes.Buffer
	report.Print(&out)
	if strings.Contains(out.String(), "synthetic-private-token") || !strings.Contains(out.String(), "gennadiy: подключён") {
		t.Fatalf("report output:\n%s", out.String())
	}
	// Running again is idempotent: same paths, token not rewritten, binary kept.
	h.commands = nil
	again, err := Connect(context.Background(), h.options(clis))
	if err != nil || again.TokenPath != wantToken || len(h.commands) != 4 {
		t.Fatalf("second run: %v %+v %v", err, again, h.commands)
	}
}

func TestConnectReportsAbsentAndManualProviders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions")
	}
	h := newHarness(t)
	gennadiy := filepath.Join(h.home, ".config", "gennadiy", "gennadiy.json")
	if err := os.MkdirAll(filepath.Dir(gennadiy), 0700); err != nil {
		t.Fatal(err)
	}
	jsonc := "{\n  // corporate config\n  \"mcp\": {}\n}\n"
	if err := os.WriteFile(gennadiy, []byte(jsonc), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Connect(context.Background(), h.options([]worker.NativeCLI{{Provider: "gennadiy", Path: "/opt/g", Version: "1"}}))
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]Entry{}
	for _, entry := range report.Entries {
		status[entry.Provider] = entry
	}
	if status["codex"].Status != "absent" || status["claude"].Status != "absent" {
		t.Fatalf("absent providers: %+v", report.Entries)
	}
	if status["gennadiy"].Status != "manual" || !strings.Contains(status["gennadiy"].Detail, `"accord"`) {
		t.Fatalf("JSONC must be reported for manual editing: %+v", status["gennadiy"])
	}
	if data, _ := os.ReadFile(gennadiy); string(data) != jsonc {
		t.Fatal("JSONC file was modified")
	}
	if len(h.commands) != 0 {
		t.Fatalf("no CLI must be invoked: %v", h.commands)
	}
	// A missing gennadiy.json is created with only the connector.
	os.Remove(gennadiy)
	if _, err = Connect(context.Background(), h.options([]worker.NativeCLI{{Provider: "gennadiy", Path: "/opt/g"}})); err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	data, _ := os.ReadFile(gennadiy)
	if json.Unmarshal(data, &config) != nil || config["mcp"].(map[string]any)["accord"] == nil {
		t.Fatalf("created config: %s", data)
	}
}

func TestConnectRejectsBadInputs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions")
	}
	h := newHarness(t)
	o := h.options(nil)
	o.TokenFile = "relative.token"
	if _, err := Connect(context.Background(), o); err == nil {
		t.Fatal("relative token accepted")
	}
	o = h.options(nil)
	if err := os.Chmod(h.token, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(context.Background(), o); err == nil {
		t.Fatal("world-readable token accepted")
	}
	if err := os.Chmod(h.token, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(filepath.Dir(h.token), "connection.json")); err != nil {
		t.Fatal(err)
	}
	o = h.options(nil)
	if _, err := Connect(context.Background(), o); err == nil || !strings.Contains(err.Error(), "--server") {
		t.Fatalf("missing hints must ask for --server: %v", err)
	}
	o.Server = "http://accord.example"
	o.Project = "p_x"
	if _, err := Connect(context.Background(), o); err == nil {
		t.Fatal("plain http origin accepted")
	}
	o.Server = "https://accord.example"
	o.Project = ""
	if _, err := Connect(context.Background(), o); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("missing project must ask for --project: %v", err)
	}
	o.Project = "p_x"
	report, err := Connect(context.Background(), o)
	if err != nil || len(report.Entries) != 3 {
		t.Fatalf("explicit flags: %v %+v", err, report)
	}
	if _, err := os.Lstat(filepath.Join(h.home, ".config", "accord", "p_x.token")); err != nil {
		t.Fatal(err)
	}
}
