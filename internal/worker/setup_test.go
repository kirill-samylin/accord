package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type setupHarness struct {
	root, token, workspace, configPath, serviceBinary, self string
	probe                                                   setupProbe
	commands                                                []string
	connected                                               []Config
}

func newSetupHarness(t *testing.T) *setupHarness {
	t.Helper()
	root, probe, token, workspace := dialogueFixture(t)
	h := &setupHarness{root: root, token: token, workspace: workspace, probe: probe, configPath: filepath.Join(root, "private", "managed", "worker.json")}
	h.serviceBinary = filepath.Join(root, "private", "agent-board")
	h.self = filepath.Join(root, "build", "agent-board")
	if err := os.MkdirAll(filepath.Dir(h.self), 0755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{h.serviceBinary: "old-binary", h.self: "new-binary"} {
		if err := os.WriteFile(path, []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

func (h *setupHarness) options(answers []string, connect func(context.Context, Config, io.Writer) error) (SetupOptions, *bytes.Buffer) {
	out := &bytes.Buffer{}
	service := &workerService{label: launchAgentLabel, plist: filepath.Join(h.root, "agent.plist"), binary: h.serviceBinary, domain: "gui/501", run: func(_ context.Context, name string, args ...string) error {
		h.commands = append(h.commands, name+" "+strings.Join(args, " "))
		return nil
	}}
	if connect == nil {
		connect = func(_ context.Context, config Config, out io.Writer) error {
			h.connected = append(h.connected, config)
			_, err := io.WriteString(out, "Local host consent updated: host_synthetic\n")
			return err
		}
	}
	probe := h.probe
	return SetupOptions{ConfigPath: h.configPath, In: strings.NewReader(strings.Join(answers, "\n") + "\n"), Out: out, probe: &probe, service: service, connect: connect, preflight: func(Config) error { return nil }, executable: h.self}, out
}

// newConfigAnswers completes a fresh dialogue with gennadiy only, then
// confirms the service stop and the binary refresh.
func (h *setupHarness) newConfigAnswers() []string {
	return []string{h.token, "", "", "", "", "", "", "", "2", h.workspace, "gui", "", "", "", "2", "", "", "", "да"}
}

func TestSetupWritesPrivateConfigAndCyclesTheService(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	h := newSetupHarness(t)
	options, out := h.options(h.newConfigAnswers(), nil)
	if err := Setup(context.Background(), options); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	info, err := os.Lstat(h.configPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("config mode: %v %v", info, err)
	}
	if parent, err := os.Lstat(filepath.Dir(h.configPath)); err != nil || parent.Mode().Perm() != 0700 {
		t.Fatalf("new config directory must be private: %v", err)
	}
	config, err := LoadConfig(h.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.Executables["gennadiy"] == "" || len(config.Executables) != 1 || config.GennadiyPermissionMode != "read-only" || !strings.Contains(strings.Join(config.NativeMCPApprovedTools["gennadiy"], ","), "transition") {
		t.Fatalf("written config: %+v", config)
	}
	if len(h.connected) != 1 || h.connected[0].Project != config.Project {
		t.Fatalf("connect calls: %+v", h.connected)
	}
	want := []string{"launchctl bootout gui/501/" + launchAgentLabel, "launchctl bootstrap gui/501 " + filepath.Join(h.root, "agent.plist")}
	if strings.Join(h.commands, "|") != strings.Join(want, "|") {
		t.Fatalf("service commands: %v", h.commands)
	}
	if data, _ := os.ReadFile(h.serviceBinary); string(data) != "new-binary" {
		t.Fatalf("service binary not refreshed: %q", data)
	}
	if _, err = os.Lstat(h.configPath + ".next"); !os.IsNotExist(err) {
		t.Fatal("temporary file left behind")
	}
	if strings.Contains(out.String(), "synthetic-private-token") {
		t.Fatal("token printed")
	}
	if !strings.Contains(out.String(), "Служба com.accord.gui.worker запущена снова") {
		t.Fatalf("missing restart notice:\n%s", out)
	}
}

func TestSetupRestartsServiceEvenWhenConnectFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	h := newSetupHarness(t)
	answers := h.newConfigAnswers()
	answers[len(answers)-1] = "нет" // keep the old service binary
	failure := errors.New("Accord rejected the worker request: HTTP 403")
	options, out := h.options(answers, func(context.Context, Config, io.Writer) error { return failure })
	if err := Setup(context.Background(), options); !errors.Is(err, failure) {
		t.Fatalf("expected connect failure, got %v\n%s", err, out)
	}
	if len(h.commands) != 2 || !strings.HasPrefix(h.commands[1], "launchctl bootstrap") {
		t.Fatalf("service must be restarted after a failed consent: %v", h.commands)
	}
	if data, _ := os.ReadFile(h.serviceBinary); string(data) != "old-binary" {
		t.Fatal("binary replaced without confirmation")
	}
	if _, err := LoadConfig(h.configPath); err != nil {
		t.Fatalf("config must be kept for a manual retry: %v", err)
	}
	if !strings.Contains(out.String(), "worker consent --config "+h.configPath) {
		t.Fatalf("missing retry hint:\n%s", out)
	}
}

func TestSetupLeavesExistingFileUntouchedOnAbortOrPreflightFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	h := newSetupHarness(t)
	if err := os.MkdirAll(filepath.Dir(h.configPath), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"server":"https://kept.example","project":"p_kept","token_file":"` + h.token + `","state_dir":"` + filepath.Join(h.root, "state") + `","name":"mac-main","machine":"m","controllers":["principal_other"],"executables":{"codex":"` + filepath.Join(h.root, "prefix", "bin", "codex") + `"},"workspaces":{"gui":{"path":"` + h.workspace + `","providers":["codex"]}}}`)
	if err := os.WriteFile(h.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	// Existing file: providers (keep codex, skip claude, add gennadiy), gui providers, no folder, modes, tools, models, then decline.
	options, out := h.options([]string{"", "", "2", "codex,gennadiy", "", "", "", "", "", "", "", "н"}, nil)
	if err := Setup(context.Background(), options); !errors.Is(err, errSetupAborted) {
		t.Fatalf("%v\n%s", err, out)
	}
	options, out = h.options([]string{"", "", "2", "codex,gennadiy", "", "", "", "", "", "", "", ""}, nil)
	options.preflight = func(Config) error { return errors.New("native provider CLI preflight failed") }
	if err := Setup(context.Background(), options); err == nil || !strings.Contains(err.Error(), "nothing written") {
		t.Fatalf("%v\n%s", err, out)
	}
	if data, _ := os.ReadFile(h.configPath); !bytes.Equal(data, original) {
		t.Fatalf("existing config modified:\n%s", data)
	}
	if len(h.commands) != 0 || len(h.connected) != 0 {
		t.Fatalf("service or connect touched before a written config: %v %v", h.commands, h.connected)
	}
	// A world-readable existing file is refused rather than edited.
	if err := os.Chmod(h.configPath, 0644); err != nil {
		t.Fatal(err)
	}
	options, _ = h.options(nil, nil)
	if err := Setup(context.Background(), options); err == nil {
		t.Fatal("non-private config accepted")
	}
}
