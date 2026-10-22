package worker

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCodexDisabledOverridesContainCompleteMatchingTransport(t *testing.T) {
	connectors := []mcpConnector{{Name: "code-review", Transport: "stdio", Enabled: true}, {Name: "remote.tool", Transport: "streamable_http", Enabled: true}}
	managedArgs := []string{"worker", "bridge", "--config", "/synthetic/private/bridge.json"}
	overrides, err := codexMCPOverrides(connectors, "/synthetic/agent-board", managedArgs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(overrides, []string{
		"-c", `mcp_servers={"code-review"={enabled=false,command="/synthetic/agent-board",args=[]},"remote.tool"={enabled=false,url="http://127.0.0.1:1"},"accord_managed"={command="/synthetic/agent-board",args=["worker","bridge","--config","/synthetic/private/bridge.json"],enabled=true}}`,
	}) {
		t.Fatal("disabled override does not contain a complete compatible transport")
	}
	for _, override := range overrides {
		for _, forbidden := range []string{"approval_policy", "sandbox_mode", "ignore-rules", "skip-git", "bypass", "bearer_token", "oauth"} {
			if strings.Contains(override, forbidden) {
				t.Fatal("MCP isolation changed native authentication or permissions")
			}
		}
	}
}

func TestCodexConnectorNamesUseTOMLMapKeysRatherThanQuotedCLIPaths(t *testing.T) {
	args, err := codexMCPOverrides([]mcpConnector{{Name: "plugin.server-name", Transport: "stdio"}}, "/synthetic/agent-board", []string{"worker", "bridge"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "-c" || !strings.HasPrefix(args[1], `mcp_servers={"plugin.server-name"={`) || strings.Contains(args[1], `mcp_servers."`) {
		t.Fatal("connector name was emitted as a literal quoted CLI path segment")
	}
}

func TestCodexReservedHTTPAndUnknownTransportFailClosed(t *testing.T) {
	for _, connector := range []mcpConnector{{Name: managedMCPName, Transport: "streamable_http"}, {Name: "unknown", Transport: "injected_unknown"}} {
		if _, err := codexMCPOverrides([]mcpConnector{connector}, "/synthetic/agent-board", nil, nil); err == nil {
			t.Fatal("created an incompatible mixed transport configuration")
		}
	}
}

func TestCodexIsolationRequiresExactManagedAndEveryOtherDisabled(t *testing.T) {
	self := "/synthetic/agent-board"
	args := []string{"worker", "bridge", "--config", "/synthetic/attempt-2/bridge.json"}
	valid := []mcpConnector{{Name: managedMCPName, Transport: "stdio", Enabled: true, Command: self, Args: args}, {Name: "builtin", Transport: "stdio", Enabled: false}, {Name: "http", Transport: "streamable_http", Enabled: false}}
	if err := checkCodexMCP(valid, self, args); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func([]mcpConnector) []mcpConnector{
		func(c []mcpConnector) []mcpConnector { c[1].Enabled = true; return c },
		func(c []mcpConnector) []mcpConnector { c[0].Command = "/synthetic/other"; return c },
		func(c []mcpConnector) []mcpConnector {
			c[0].Args = []string{"worker", "bridge", "--config", "/synthetic/old-attempt/bridge.json"}
			return c
		},
		func(c []mcpConnector) []mcpConnector { c[0].Enabled = false; return c },
		func(c []mcpConnector) []mcpConnector { c[0].Transport = "streamable_http"; return c },
		func(c []mcpConnector) []mcpConnector { return c[1:] },
		func(c []mcpConnector) []mcpConnector { return append(c, c[0]) },
	} {
		if err := checkCodexMCP(change(append([]mcpConnector(nil), valid...)), self, args); err == nil {
			t.Fatal("configuration inspection accepted mismatched or non-isolated MCP")
		}
	}
}

func TestBridgeRequiresConfigOnlyVerificationBeforeReturningNativeArgs(t *testing.T) {
	e := testEngine(t)
	verified := false
	e.verifyMCP = func(executable, directory string, overrides []string, self string, args []string) error {
		verified = true
		if executable != e.config.Executables["codex"] || directory != e.config.Workspaces["test"].Path || len(overrides) == 0 || self == "" || len(args) != 4 {
			t.Error("inspection did not receive exact local invocation binding")
		}
		return errors.New("synthetic non-isolated configuration")
	}
	if _, err := e.bridgeArgs(RunState{ID: "run_verification", Provider: "codex", Workspace: "test", Generation: 1}); err == nil || !verified {
		t.Fatal("native invocation bypassed configuration isolation verification")
	}
}

func TestMCPDiscoveryRejectsUnknownOrAmbiguousMetadata(t *testing.T) {
	for _, data := range []string{
		`[{"name":"builtin","enabled":true,"transport":{"type":"future_unknown"}}]`,
		`[{"name":"builtin","transport":{"type":"stdio"}}]`,
		`[{"name":"same","enabled":true,"transport":{"type":"stdio"}},{"name":"same","enabled":false,"transport":{"type":"stdio"}}]`,
	} {
		if _, err := parseMCPConnectors([]byte(data)); err == nil {
			t.Fatal("ambiguous transport metadata accepted")
		}
	}
}
