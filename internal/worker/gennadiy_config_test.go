package worker

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

const syntheticGennadiySession = "ses_0123456789abcdefABCDEFghij"

// The owner's resolved config as `debug config` prints it: corporate remote
// connectors with headers, one disabled local one, agent-level "ask" policy.
const syntheticGennadiyConfig = `{"model":"gennadiy/synthetic","default_agent":"plan","permission":{"edit":"ask","corp_send":"ask"},"agent":{"build":{"permission":{"edit":"ask","bash":"ask"}},"plan":{"permission":{"edit":"deny"}}},"mcp":{"corp-jira":{"type":"remote","url":"https://example.invalid/mcp?token=synthetic-url-secret","enabled":true,"headers":{"Authorization":"Bearer synthetic-header-secret"}},"ci-local":{"type":"local","enabled":false,"command":["/usr/bin/node","/synthetic/mcp.mjs"]}}}`

func testGennadiyEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, _ = filepath.EvalSymlinks(executable)
	token := filepath.Join(dir, "token")
	if err = os.WriteFile(token, []byte("synthetic-private-token"), 0600); err != nil {
		t.Fatal(err)
	}
	config := Config{Server: "http://127.0.0.1:12345", Project: "p_00000000000000000000000000000001", TokenFile: token, StateDir: filepath.Join(dir, "state"), Name: "synthetic", Machine: "test-machine", Executables: map[string]string{"gennadiy": executable}, GennadiyPermissionMode: "workspace-write", Workspaces: map[string]Workspace{"test": {Path: dir, Providers: []string{"gennadiy"}}}}
	e, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	e.discoverGennadiy = func(string, string) ([]mcpConnector, error) {
		resolved, err := parseGennadiyConfig("discovery", []byte(syntheticGennadiyConfig))
		return gennadiyConnectors(resolved), err
	}
	e.verifyGennadiy = func(string, string, string, string, []string, map[string]string) error { return nil }
	e.workspaceLockDir = filepath.Join(dir, "workspace-locks")
	e.probe = func(Config) error { return nil }
	e.journal.OwnerPrincipalID = "principal_test"
	if err = e.SetLease(1, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func TestGennadiyHeadlessArgumentsAndExactSession(t *testing.T) {
	args, err := providerArgs("gennadiy", "")
	if err != nil || !reflect.DeepEqual(args, []string{"run", "--format", "json", "--agent", "build"}) {
		t.Fatalf("start arguments: %v %v", args, err)
	}
	args, err = providerArgs("gennadiy", syntheticGennadiySession)
	if err != nil || !reflect.DeepEqual(args, []string{"run", "--format", "json", "--agent", "build", "--session", syntheticGennadiySession}) {
		t.Fatalf("resume arguments: %v %v", args, err)
	}
	for _, bad := range []string{syntheticSession, "ses_short", "--continue", "ses_" + strings.Repeat("a", 65), "ses_id --dangerously-skip-permissions"} {
		if _, err = providerArgs("gennadiy", bad); err == nil {
			t.Fatalf("accepted a non-exact OpenCode session %q", bad)
		}
	}
	if _, err = providerArgs("codex", syntheticGennadiySession); err == nil {
		t.Fatal("OpenCode session accepted for Codex")
	}
	if !validSessionID("gennadiy", syntheticGennadiySession) || validSessionID("gennadiy", syntheticSession) || !validSessionID("claude", syntheticSession) {
		t.Fatal("session format is not provider-exact")
	}
}

func TestGennadiyEventsRequireStepAndNoError(t *testing.T) {
	for _, test := range []struct {
		line                               string
		session, started, complete, failed bool
		output                             string
	}{
		{`{"type":"step_start","timestamp":1,"sessionID":"` + syntheticGennadiySession + `","part":{"type":"step-start"}}`, true, true, false, false, ""},
		{`{"type":"tool_use","sessionID":"` + syntheticGennadiySession + `","part":{"type":"tool","tool":"read"}}`, true, false, false, false, ""},
		{`{"type":"text","sessionID":"` + syntheticGennadiySession + `","part":{"type":"text","text":"synthetic answer"}}`, true, false, false, false, "synthetic answer"},
		{`{"type":"step_finish","sessionID":"` + syntheticGennadiySession + `","part":{"type":"step-finish"}}`, true, false, true, false, ""},
		{`{"type":"error","sessionID":"` + syntheticGennadiySession + `","error":{"name":"ProviderAuthError"}}`, true, false, false, true, ""},
		{`{"type":"step_start","sessionID":"` + syntheticSession + `"}`, false, true, false, false, ""},
		{`not json`, false, false, false, false, ""},
	} {
		event := parseProviderEvent("gennadiy", []byte(test.line))
		if (event.session != "") != test.session || event.started != test.started || event.complete != test.complete || event.failed != test.failed || event.output != test.output {
			t.Fatalf("unexpected event for %s: %+v", test.line, event)
		}
	}
	if parseProviderEvent("claude", []byte(`{"type":"step_finish","sessionID":"`+syntheticGennadiySession+`"}`)).complete {
		t.Fatal("OpenCode events interpreted for another provider")
	}
}

func TestGennadiyConfigValidation(t *testing.T) {
	e := testGennadiyEngine(t)
	config := e.config
	for _, mode := range []string{"", "read-only", "workspace-write"} {
		config.GennadiyPermissionMode = mode
		if err := config.Validate(); err != nil {
			t.Fatal(mode, err)
		}
	}
	config.GennadiyPermissionMode = "danger-full-access"
	if config.Validate() == nil {
		t.Fatal("unknown permission mode accepted")
	}
	config = e.config
	config.Executables = map[string]string{"codex": config.Executables["gennadiy"], "claude": config.Executables["gennadiy"], "gennadiy": config.Executables["gennadiy"]}
	if err := config.Validate(); err != nil {
		t.Fatal("three native providers rejected:", err)
	}
	config.Executables["opencode"] = config.Executables["gennadiy"]
	if config.Validate() == nil {
		t.Fatal("unknown provider accepted")
	}
	if validateNativeMCPConsent(map[string][]string{"gennadiy": {"describe"}}, e.config.Executables) != nil || validateNativeMCPConsent(map[string][]string{"gennadiy": {"describe"}}, nil) == nil {
		t.Fatal("gennadiy consent validation differs from other providers")
	}
	if !providerProcessNames(e.config)["gennadiy"] {
		t.Fatal("recovery scan does not recognise the provider")
	}
	for _, key := range []string{"OPENCODE_CONFIG_CONTENT=synthetic", "OPENCODE_CONFIG=/synthetic", "AGENT_BOARD_TOKEN=x"} {
		name, _, _ := strings.Cut(key, "=")
		t.Setenv(name, "synthetic")
	}
	if joined := strings.Join(providerEnv(), "\n"); strings.Contains(joined, "OPENCODE_CONFIG") || strings.Contains(joined, "AGENT_BOARD_") {
		t.Fatal("ambient OpenCode config override inherited by the native child")
	}
}

func TestGennadiyPermissionsArePinnedPerModeAndRole(t *testing.T) {
	ro := gennadiyPermissions("read-only", nil)
	rw := gennadiyPermissions("workspace-write", []string{"describe", "next"})
	if ro["edit"] != "deny" || ro["bash"] != "deny" || ro["external_directory"] != "deny" || rw["edit"] != "allow" || rw["bash"] != "allow" || rw["external_directory"] != "deny" {
		t.Fatalf("mode policy: %v %v", ro, rw)
	}
	for _, tool := range managedMCPTools {
		want := "deny"
		if tool == "describe" || tool == "next" {
			want = "allow"
		}
		if ro[managedMCPName+"_"+tool] != "deny" || rw[managedMCPName+"_"+tool] != want {
			t.Fatalf("bridge tool %s is not pinned exactly", tool)
		}
	}
	if len(rw) != 3+len(managedMCPTools) {
		t.Fatal("policy contains keys outside the managed scope")
	}
	config := Config{GennadiyPermissionMode: "workspace-write", Automation: &AutomationConfig{Workspaces: map[string]AutomationWorkspace{"test": {AllowTesterArtifactWrites: true}}}}
	manual := RunState{Workspace: "test"}
	role := func(name string) RunState {
		return RunState{Workspace: "test", Automation: &automationAttempt{Run: core.ManagedRun{Profile: &core.ProfileSnapshot{ProfileSettings: core.ProfileSettings{Role: name}}}}}
	}
	if gennadiyMode(config, manual) != "workspace-write" || gennadiyMode(Config{}, manual) != "read-only" || gennadiyMode(config, role("developer")) != "workspace-write" || gennadiyMode(config, role("tester")) != "workspace-write" || gennadiyMode(config, role("reviewer")) != "read-only" || gennadiyMode(config, role("analyst")) != "read-only" {
		t.Fatal("role-specific write mode differs from the Codex sandbox rule")
	}
	config.Automation.Workspaces["test"] = AutomationWorkspace{}
	if gennadiyMode(config, role("tester")) != "read-only" {
		t.Fatal("tester writes without separate artifact consent")
	}
}

func TestGennadiyInjectedContentBindsOnlyThisAttempt(t *testing.T) {
	resolved, err := parseGennadiyConfig("discovery", []byte(syntheticGennadiyConfig))
	if err != nil {
		t.Fatal(err)
	}
	connectors := gennadiyConnectors(resolved)
	if len(connectors) != 2 || connectors[0].Name != "ci-local" || connectors[0].Transport != "stdio" || connectors[0].Enabled || connectors[1].Name != "corp-jira" || connectors[1].Transport != "streamable_http" || !connectors[1].Enabled {
		t.Fatalf("discovery: %+v", connectors)
	}
	if encoded, _ := json.Marshal(connectors); strings.Contains(string(encoded), "synthetic") {
		t.Fatal("discovery retained connector credentials")
	}
	permission := gennadiyPermissions("read-only", []string{"describe"})
	content, err := gennadiyConfigContent(connectors, "/synthetic/self", []string{"worker", "bridge", "--config", "/private/bridge.json"}, permission)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "synthetic-url-secret") || strings.Contains(content, "synthetic-header-secret") || strings.Contains(content, "example.invalid") {
		t.Fatal("injected content copies foreign connector secrets")
	}
	var injected map[string]any
	if json.Unmarshal([]byte(content), &injected) != nil {
		t.Fatal("content is not JSON")
	}
	mcp := injected["mcp"].(map[string]any)
	if len(mcp) != 3 || mcp["corp-jira"].(map[string]any)["enabled"] != false || mcp["ci-local"].(map[string]any)["enabled"] != false || mcp[managedMCPName].(map[string]any)["enabled"] != true {
		t.Fatalf("foreign connectors stay enabled or bridge missing: %v", mcp)
	}
	if injected["agent"].(map[string]any)["build"].(map[string]any)["permission"].(map[string]any)["edit"] != "deny" || injected["permission"].(map[string]any)["accord_managed_describe"] != "allow" {
		t.Fatal("agent-level policy is not pinned")
	}
	for _, forbidden := range []string{"dangerously", "OPENCODE_CONFIG_DIR", "plugin", "provider", "model"} {
		if _, present := injected[forbidden]; present || strings.Contains(content, "dangerously") {
			t.Fatalf("content changes %s", forbidden)
		}
	}
	if _, err = gennadiyConfigContent([]mcpConnector{{Name: managedMCPName, Transport: "streamable_http", Enabled: true}}, "/s", nil, permission); err == nil || preparationDiagnostic(err) != "prepare_override_reserved" {
		t.Fatal("reserved bridge name with another transport accepted")
	}
}

func TestGennadiyVerificationRejectsForeignBindingsAndPolicyDrift(t *testing.T) {
	self, args := "/synthetic/self", []string{"worker", "bridge", "--config", "/private/bridge.json"}
	permission := gennadiyPermissions("workspace-write", nil)
	good := `{"permission":PERM,"agent":{"build":{"permission":PERM}},"mcp":{"corp-jira":{"type":"remote","enabled":false},"accord_managed":{"type":"local","enabled":true,"command":["/synthetic/self","worker","bridge","--config","/private/bridge.json"]}}}`
	perm, _ := json.Marshal(permission)
	parse := func(text string) gennadiyResolved {
		resolved, err := parseGennadiyConfig("verify", []byte(strings.ReplaceAll(text, "PERM", string(perm))))
		if err != nil {
			t.Fatal(err)
		}
		return resolved
	}
	if err := checkGennadiyConfig(parse(good), self, args, permission); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"foreign enabled":  strings.Replace(good, `"corp-jira":{"type":"remote","enabled":false}`, `"corp-jira":{"type":"remote","enabled":true}`, 1),
		"other command":    strings.Replace(good, `"/synthetic/self","worker"`, `"/synthetic/other","worker"`, 1),
		"bridge disabled":  strings.Replace(good, `"enabled":true,"command"`, `"enabled":false,"command"`, 1),
		"bridge missing":   `{"permission":PERM,"agent":{"build":{"permission":PERM}},"mcp":{}}`,
		"policy drift":     strings.Replace(good, `"agent":{"build":{"permission":PERM}}`, `"agent":{"build":{"permission":{"edit":"ask"}}}`, 1),
		"global drift":     `{"permission":{"edit":"deny"},"agent":{"build":{"permission":PERM}},"mcp":{"accord_managed":{"type":"local","enabled":true,"command":["/synthetic/self","worker","bridge","--config","/private/bridge.json"]}}}`,
		"remote transport": strings.Replace(good, `"accord_managed":{"type":"local"`, `"accord_managed":{"type":"remote"`, 1),
	} {
		if checkGennadiyConfig(parse(bad), self, args, permission) == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	for _, text := range []string{"", "banner\n{}", "[]", "{"} {
		if _, err := parseGennadiyConfig("verify", []byte(text)); err == nil || preparationDiagnostic(err) != "prepare_verify_json" {
			t.Fatalf("non-JSON config accepted: %q", text)
		}
	}
}

type envProbeProcess struct {
	output io.ReadCloser
}

func (p envProbeProcess) Output() io.ReadCloser { return p.output }
func (p envProbeProcess) Wait() error           { return nil }
func (p envProbeProcess) Stop() error           { return p.output.Close() }
func (p envProbeProcess) Forced() bool          { return false }

func TestGennadiyConfigReadsPassExplicitEnvironmentOnly(t *testing.T) {
	var seen launchSpec
	data, err := readNativeWithEnv("verify", "/synthetic/gennadiy", "/synthetic/workspace", []string{"debug", "config"}, []string{"OPENCODE_CONFIG_CONTENT={}"}, time.Second, func(spec launchSpec) (process, error) {
		seen = spec
		return envProbeProcess{output: io.NopCloser(strings.NewReader(syntheticGennadiyConfig))}, nil
	})
	if err != nil || !strings.HasPrefix(string(data), "{") {
		t.Fatal("bounded config read failed", err)
	}
	if !reflect.DeepEqual(seen.args, []string{"debug", "config"}) || !reflect.DeepEqual(seen.env, []string{"OPENCODE_CONFIG_CONTENT={}"}) || seen.prompt != "" || seen.directory != "/synthetic/workspace" {
		t.Fatalf("config-only invocation changed: %+v", seen)
	}
}

func TestGennadiyBridgeInjectsEnvironmentWithoutArgvOrFiles(t *testing.T) {
	e := testGennadiyEngine(t)
	e.config.NativeMCPApprovedTools = map[string][]string{"gennadiy": {"describe", "next"}}
	var verifiedContent string
	var verifiedPermission map[string]string
	e.verifyGennadiy = func(_, _, content, self string, args []string, permission map[string]string) error {
		verifiedContent, verifiedPermission = content, permission
		if !strings.Contains(content, self) || len(args) != 4 || args[0] != "worker" || args[1] != "bridge" {
			t.Error("verification received another binding")
		}
		return nil
	}
	run := RunState{ID: "run_gennadiy", Provider: "gennadiy", Workspace: "test", Generation: 1}
	launch, err := e.bridgeArgs(run)
	if err != nil {
		t.Fatal(err)
	}
	if len(launch.Args) != 0 || len(launch.Env) != 3 || launch.Env[2] != "OPENCODE_CONFIG_CONTENT="+verifiedContent || launch.Env[0] != "OPENCODE_DISABLE_AUTOUPDATE=1" {
		t.Fatalf("bridge binding is not injected through the environment only: %+v", launch)
	}
	if verifiedPermission["accord_managed_next"] != "allow" || verifiedPermission["accord_managed_update"] != "deny" || verifiedPermission["edit"] != "allow" {
		t.Fatal("locally selected consent and write mode are not pinned")
	}
	bridge := filepath.Join(e.config.StateDir, "runs", run.ID, "attempt-1", "bridge.json")
	if _, err = os.Stat(bridge); err != nil {
		t.Fatal("attempt binding file missing", err)
	}
	if _, err = os.Stat(filepath.Join(e.config.StateDir, "runs", run.ID, "attempt-1", "mcp.json")); err == nil {
		t.Fatal("Claude MCP file written for OpenCode")
	}
	if !strings.Contains(verifiedContent, bridge) || strings.Contains(verifiedContent, e.journal.HostSecret) || strings.Contains(verifiedContent, "synthetic-private-token") {
		t.Fatal("content binds another attempt or leaks credentials")
	}
	if len(e.preparedConnectors) != 2 || e.preparedConnectors[1].Name != "corp-jira" {
		t.Fatal("discovered connectors not recorded for the attempt")
	}
	e.verifyGennadiy = func(string, string, string, string, []string, map[string]string) error {
		return preparationFailureAt("prepare_verify_binding", nil)
	}
	if _, err = e.bridgeArgs(run); err == nil || preparationDiagnostic(err) != "prepare_verify_binding" {
		t.Fatal("unverified binding launched", err)
	}
	e.discoverGennadiy = func(string, string) ([]mcpConnector, error) {
		return nil, preparationFailureAt("prepare_discovery_json", nil)
	}
	if _, err = e.bridgeArgs(run); err == nil || preparationDiagnostic(err) != "prepare_discovery_json" {
		t.Fatal("launch without connector discovery", err)
	}
}
