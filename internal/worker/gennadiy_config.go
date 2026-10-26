package worker

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"time"
)

// gennadiyAgent is the native OpenCode coding agent the managed run starts.
// Its permissions are pinned by the injected preset below; the corporate
// default agent (plan) denies edits and shell outright.
const gennadiyAgent = "build"

// Gennadiy has no `--mcp-config` flag. OpenCode merges OPENCODE_CONFIG_CONTENT
// over the owner's global config, so the worker injects only this attempt's
// bridge, disables every discovered foreign connector and pins the permission
// preset. Headless `run` auto-rejects every "ask", so a pinned "allow"/"deny"
// is the whole effective policy and nothing is left to a hidden prompt.
var gennadiyExtraEnv = []string{"OPENCODE_DISABLE_AUTOUPDATE=1", "OPENCODE_DISABLE_MODELS_FETCH=1"}

func gennadiyEnv(content string) []string {
	return append(append([]string(nil), gennadiyExtraEnv...), "OPENCODE_CONFIG_CONTENT="+content)
}

// gennadiyPermissions pins the exact native policy for one managed attempt.
// read-only denies edits and shell: OpenCode has no OS sandbox, so a read-only
// shell cannot be promised. Unselected bridge tools are denied explicitly
// because the native default for MCP tools is allow, not prompt.
func gennadiyPermissions(mode string, approved []string) map[string]string {
	permission := map[string]string{"edit": "deny", "bash": "deny", "external_directory": "deny"}
	if mode == "workspace-write" {
		permission["edit"], permission["bash"] = "allow", "allow"
	}
	selected := map[string]bool{}
	for _, tool := range approved {
		selected[tool] = true
	}
	for _, tool := range managedMCPTools {
		value := "deny"
		if selected[tool] {
			value = "allow"
		}
		permission[managedMCPName+"_"+tool] = value
	}
	return permission
}

// gennadiyMode is the locally consented write mode for this run. Automation
// roles other than developer stay read-only, like the Codex sandbox choice.
func gennadiyMode(config Config, run RunState) string {
	mode := config.GennadiyPermissionMode
	if mode == "" {
		mode = "read-only"
	}
	if run.Automation == nil || run.Automation.Run.Profile == nil {
		return mode
	}
	switch run.Automation.Run.Profile.Role {
	case "developer":
		return mode
	case "tester":
		if config.Automation != nil && config.Automation.Workspaces[run.Workspace].AllowTesterArtifactWrites && mode == "workspace-write" {
			return "workspace-write"
		}
	}
	return "read-only"
}

func gennadiyConfigContent(connectors []mcpConnector, self string, args []string, permission map[string]string) (string, error) {
	mcp := map[string]any{}
	for _, connector := range connectors {
		if connector.Name == managedMCPName {
			if connector.Transport != "stdio" {
				return "", preparationFailureAt("prepare_override_reserved", nil)
			}
			continue
		}
		switch connector.Transport {
		case "stdio":
			mcp[connector.Name] = map[string]any{"type": "local", "command": []string{self}, "enabled": false}
		case "streamable_http":
			// A disabled loopback URL replaces the original without copying any
			// access-bearing URL or headers into the injected content.
			mcp[connector.Name] = map[string]any{"type": "remote", "url": "http://127.0.0.1:1", "enabled": false}
		default:
			return "", preparationFailureAt("prepare_override_transport", nil)
		}
	}
	mcp[managedMCPName] = map[string]any{"type": "local", "command": append([]string{self}, args...), "enabled": true}
	content := map[string]any{"mcp": mcp, "permission": permission, "agent": map[string]any{gennadiyAgent: map[string]any{"permission": permission}}}
	data, err := json.Marshal(content)
	if err != nil {
		return "", preparationFailureAt("prepare_gennadiy_content", err)
	}
	return string(data), nil
}

// gennadiyResolved is the subset of `debug config` output the worker checks.
type gennadiyResolved struct {
	MCP map[string]struct {
		Type    string   `json:"type"`
		Enabled *bool    `json:"enabled"`
		Command []string `json:"command"`
	} `json:"mcp"`
	Permission map[string]any `json:"permission"`
	Agent      map[string]struct {
		Permission map[string]any `json:"permission"`
	} `json:"agent"`
}

func parseGennadiyConfig(context string, data []byte) (gennadiyResolved, error) {
	var resolved gennadiyResolved
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' || json.Unmarshal(data, &resolved) != nil {
		return resolved, preparationFailureAt("prepare_"+context+"_json", nil)
	}
	if len(resolved.MCP) > 128 {
		return resolved, preparationFailureAt("prepare_"+context+"_json", nil)
	}
	return resolved, nil
}

func gennadiyConnectors(resolved gennadiyResolved) []mcpConnector {
	var connectors []mcpConnector
	for name, entry := range resolved.MCP {
		transport := "stdio"
		if entry.Type == "remote" {
			transport = "streamable_http"
		} else if entry.Type != "local" {
			transport = entry.Type
		}
		enabled := entry.Enabled == nil || *entry.Enabled
		connectors = append(connectors, mcpConnector{Name: name, Transport: transport, Enabled: enabled})
	}
	sort.Slice(connectors, func(i, j int) bool { return connectors[i].Name < connectors[j].Name })
	return connectors
}

// readGennadiyHelp reads `run --help`, which the fork prints to stderr.
func readGennadiyHelp(executable, directory string) ([]byte, error) {
	return readNativeSpec("preflight", launchSpec{executable: executable, directory: directory, args: []string{"run", "--help"}, env: append(providerEnv(), gennadiyExtraEnv...), mergeStderr: true}, 5*time.Second, startProcess)
}

func readGennadiyConfig(context, executable, directory string, env []string) (gennadiyResolved, error) {
	data, err := readNativeWithEnv(context, executable, directory, []string{"debug", "config"}, env, 20*time.Second, startProcess)
	if err != nil {
		return gennadiyResolved{}, err
	}
	return parseGennadiyConfig(context, data)
}

// discoverGennadiyMCP reads the owner's resolved global config without any
// injected content; names and transports only, never headers or URLs.
func discoverGennadiyMCP(executable, directory string) ([]mcpConnector, error) {
	resolved, err := readGennadiyConfig("discovery", executable, directory, append(providerEnv(), gennadiyExtraEnv...))
	if err != nil {
		return nil, err
	}
	for _, connector := range gennadiyConnectors(resolved) {
		if connector.Transport != "stdio" && connector.Transport != "streamable_http" {
			return nil, preparationFailureAt("prepare_discovery_transport", nil)
		}
		if connector.Name == "" || len(connector.Name) > 128 {
			return nil, preparationFailureAt("prepare_discovery_metadata", nil)
		}
	}
	return gennadiyConnectors(resolved), nil
}

// verifyGennadiyConfig re-reads the resolved config with the injected content
// and refuses the launch unless the bridge binding and pinned policy are the
// only effective ones. This is the second config-only inspection, like Codex.
func verifyGennadiyConfig(executable, directory, content, self string, args []string, permission map[string]string) error {
	resolved, err := readGennadiyConfig("verify", executable, directory, append(providerEnv(), gennadiyEnv(content)...))
	if err != nil {
		return err
	}
	return checkGennadiyConfig(resolved, self, args, permission)
}

func checkGennadiyConfig(resolved gennadiyResolved, self string, args []string, permission map[string]string) error {
	found := false
	for _, connector := range gennadiyConnectors(resolved) {
		if connector.Name == managedMCPName {
			entry := resolved.MCP[connector.Name]
			if found || !connector.Enabled || connector.Transport != "stdio" || !reflect.DeepEqual(entry.Command, append([]string{self}, args...)) {
				return preparationFailureAt("prepare_verify_binding", nil)
			}
			found = true
		} else if connector.Enabled {
			return preparationFailureAt("prepare_verify_binding", nil)
		}
	}
	if !found {
		return preparationFailureAt("prepare_verify_binding", nil)
	}
	for _, effective := range []map[string]any{resolved.Permission, resolved.Agent[gennadiyAgent].Permission} {
		for key, want := range permission {
			if got, _ := effective[key].(string); got != want {
				return preparationFailureAt("prepare_verify_permission", nil)
			}
		}
	}
	return nil
}
