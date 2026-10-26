package worker

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// dialogueFixture adds a token file, connection hints and a workspace to the
// discovery fixture so a scripted stdin can complete the whole dialogue.
func dialogueFixture(t *testing.T) (root string, probe setupProbe, token, workspace string) {
	t.Helper()
	root, probe = setupFixture(t)
	private := filepath.Join(root, "private")
	if err := os.MkdirAll(private, 0700); err != nil {
		t.Fatal(err)
	}
	token = filepath.Join(private, "gui.token")
	if err := os.WriteFile(token, []byte("synthetic-private-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hints := `{"origin":"https://accord.example","project_id":"p_00000000000000000000000000000001","machine":"mac-test","owner":"x"}`
	if err := os.WriteFile(filepath.Join(private, "connection.json"), []byte(hints), 0600); err != nil {
		t.Fatal(err)
	}
	workspace = filepath.Join(root, "Projects", "GUI App")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	return root, probe, token, workspace
}

func runDialogue(t *testing.T, probe setupProbe, configPath string, current Config, existing bool, answers ...string) (Config, string, error) {
	t.Helper()
	var out bytes.Buffer
	p := newSetupPrompter(strings.NewReader(strings.Join(answers, "\n")+"\n"), &out)
	config, err := runSetupDialogue(p, probe, configPath, current, existing)
	return config, out.String(), err
}

func TestSetupDialogueCreatesConfigFromScriptedAnswers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	root, probe, token, workspace := dialogueFixture(t)
	configPath := filepath.Join(root, "private", "worker.json")
	config, out, err := runDialogue(t, probe, configPath, Config{}, false,
		token, "", "", "", "", "", // connection: token, server/project/machine from hints, name default, state default
		"2", "", "2", // providers: codex found, claude skipped, gennadiy found
		workspace, "", "gennadiy", "", // first workspace, default alias, providers, finish
		"2", "", // codex workspace-write, gennadiy read-only
		"", "2", // tools: codex read-only preset, gennadiy task preset
		"", "gennadiy/m1, gennadiy/m2", // models
		"", // confirm write
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if config.Server != "https://accord.example" || config.Project != "p_00000000000000000000000000000001" || config.Machine != "mac-test" || config.TokenFile != token || config.StateDir != filepath.Join(root, "private", "state") || config.Name != defaultHostName() {
		t.Fatalf("connection fields: %+v", config)
	}
	want := map[string]string{"codex": filepath.Join(root, "prefix", "bin", "codex"), "gennadiy": filepath.Join(root, "prefix", "lib", "node_modules", "gennadiy", "node_modules", probe.gennadiyPlatformPackage(), "bin", "gennadiy")}
	if !reflect.DeepEqual(config.Executables, want) {
		t.Fatalf("executables: %+v", config.Executables)
	}
	if !reflect.DeepEqual(config.Workspaces, map[string]Workspace{"gui-app": {Path: workspace, Providers: []string{"gennadiy"}}}) {
		t.Fatalf("workspaces: %+v", config.Workspaces)
	}
	if config.CodexSandbox != "workspace-write" || config.GennadiyPermissionMode != "read-only" || config.ClaudePermissionMode != "" {
		t.Fatalf("modes: %+v", config)
	}
	if !reflect.DeepEqual(config.NativeMCPApprovedTools, map[string][]string{"codex": setupReadOnlyTools, "gennadiy": setupTaskTools}) {
		t.Fatalf("tools: %+v", config.NativeMCPApprovedTools)
	}
	if !reflect.DeepEqual(config.Models, map[string][]string{"gennadiy": {"gennadiy/m1", "gennadiy/m2"}}) {
		t.Fatalf("models: %+v", config.Models)
	}
	if err = config.Validate(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "synthetic-private-token") {
		t.Fatal("dialogue printed the token")
	}
}

func TestSetupDialogueEditsExistingConfigAndKeepsConnection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	root, probe, token, workspace := dialogueFixture(t)
	current := Config{Server: "https://kept.example", Project: "p_kept", TokenFile: token, StateDir: filepath.Join(root, "state"), Name: "mac-main", Machine: "kept-machine", Controllers: []string{"principal_other"},
		Executables: map[string]string{"codex": filepath.Join(root, "prefix", "bin", "codex")}, CodexSandbox: "workspace-write",
		NativeMCPApprovedTools: map[string][]string{"codex": {"describe", "next"}}, Workspaces: map[string]Workspace{"gui": {Path: workspace, Providers: []string{"codex"}}}}
	config, out, err := runDialogue(t, probe, filepath.Join(root, "w.json"), current, true,
		"", "", "2", // keep codex (default = found), skip claude, add gennadiy
		"codex,gennadiy", "", // gui providers, no new folder
		"", "2", // codex keeps workspace-write, gennadiy workspace-write
		"", "1", // codex keeps its current tools (default), gennadiy read-only
		"", "", // models
		"", // confirm
	)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if config.Server != current.Server || config.Project != current.Project || config.TokenFile != token || config.StateDir != current.StateDir || config.Machine != current.Machine || config.Name != current.Name || !reflect.DeepEqual(config.Controllers, current.Controllers) {
		t.Fatalf("connection fields changed: %+v", config)
	}
	if len(config.Executables) != 2 || config.Executables["gennadiy"] == "" || config.CodexSandbox != "workspace-write" || config.GennadiyPermissionMode != "workspace-write" {
		t.Fatalf("providers/modes: %+v", config)
	}
	if !reflect.DeepEqual(config.Workspaces["gui"].Providers, []string{"codex", "gennadiy"}) {
		t.Fatalf("workspace providers: %+v", config.Workspaces)
	}
	if !reflect.DeepEqual(config.NativeMCPApprovedTools, map[string][]string{"codex": {"describe", "next"}, "gennadiy": setupReadOnlyTools}) {
		t.Fatalf("tools: %+v", config.NativeMCPApprovedTools)
	}
	if !strings.Contains(out, "Подключение сохранено") || strings.Contains(out, "Путь к приватному файлу токена") {
		t.Fatalf("existing connection must not be asked again:\n%s", out)
	}
}

func TestSetupDialogueRejectsBadAnswersAndAborts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses Unix symlinks")
	}
	root, probe, token, workspace := dialogueFixture(t)
	// A relative token path and an unreadable token are re-asked; a non-https
	// server is re-asked; a workspace that is not a directory is re-asked.
	_, out, err := runDialogue(t, probe, filepath.Join(root, "w.json"), Config{}, false,
		"relative/token", filepath.Join(root, "missing.token"), token, "ftp://x", "", "", "", "bad name!", "mac-2", "",
		"2", "", "", filepath.Join(root, "nope"), workspace, "", "", "", "", "", "", "н")
	if !errors.Is(err, errSetupAborted) {
		t.Fatalf("expected abort, got %v\n%s", err, out)
	}
	for _, text := range []string{"Нужен абсолютный путь", "Файл токена не принят", "https://", "Допустимы латинские буквы", "Папка не принята"} {
		if !strings.Contains(out, text) {
			t.Fatalf("missing re-ask %q:\n%s", text, out)
		}
	}
	_, _, err = runDialogue(t, probe, filepath.Join(root, "w.json"), Config{}, false, token)
	if err == nil || errors.Is(err, errSetupAborted) {
		t.Fatalf("EOF must fail the dialogue, got %v", err)
	}
	_, out, err = runDialogue(t, probe, filepath.Join(root, "w.json"), Config{}, false, token, "", "", "", "", "", "", "", "")
	if !errors.Is(err, errSetupAborted) || !strings.Contains(out, "Не выбран ни один провайдер") {
		t.Fatalf("no provider must abort: %v\n%s", err, out)
	}
}
