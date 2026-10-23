package worker

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

const managedMCPName = "accord_managed"

func codexMCPOverrides(connectors []mcpConnector, self string, args []string, approved []string) ([]string, error) {
	var tables []string
	for _, connector := range connectors {
		if connector.Name == managedMCPName {
			if connector.Transport != "stdio" {
				return nil, preparationFailureAt("prepare_override_reserved", nil)
			}
			continue
		}
		var transport string
		switch connector.Transport {
		case "stdio":
			// Plugin-injected stdio servers have no transport in user config.
			// An enabled-only override creates an invalid new table for them.
			transport = "{enabled=false,command=" + strconv.Quote(self) + ",args=[]}"
		case "streamable_http":
			// Keep HTTP and stdio mutually exclusive. A disabled loopback URL
			// avoids copying access-bearing original URLs into process argv.
			transport = `{enabled=false,url="http://127.0.0.1:1"}`
		default:
			return nil, preparationFailureAt("prepare_override_transport", nil)
		}
		tables = append(tables, strconv.Quote(connector.Name)+"="+transport)
	}
	encoded, _ := json.Marshal(args)
	tables = append(tables, strconv.Quote(managedMCPName)+"={command="+strconv.Quote(self)+",args="+string(encoded)+",enabled=true"+codexMCPConsent(approved)+"}")
	// Codex's CLI key-path parser treats quoted path segments literally.
	// Put names inside one TOML map, where quotes have TOML key semantics.
	return []string{"-c", "mcp_servers={" + strings.Join(tables, ",") + "}"}, nil
}

func verifyCodexMCP(executable, directory string, overrides []string, self string, args []string) error {
	command := append(append([]string(nil), overrides...), "mcp", "list", "--json")
	data, err := readNativeFor("verify", executable, directory, command, startProcess)
	if err != nil {
		return err
	}
	connectors, err := parseMCPFor("verify", data)
	if err != nil {
		return err
	}
	return checkCodexMCP(connectors, self, args)
}

func checkCodexMCP(connectors []mcpConnector, self string, args []string) error {
	found := false
	for _, connector := range connectors {
		if connector.Name == managedMCPName {
			if found || !connector.Enabled || connector.Transport != "stdio" || connector.Command != self || !reflect.DeepEqual(connector.Args, args) {
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
	return nil
}
