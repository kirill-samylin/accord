package worker

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

type mcpConnector struct {
	Name      string
	Transport string
	Enabled   bool
	Command   string
	Args      []string
}

func discoverCodexMCP(executable, directory string) ([]mcpConnector, error) {
	data, err := readNative(executable, directory, []string{"mcp", "list", "--json"})
	if err != nil {
		return nil, err
	}
	return parseMCPConnectors(data)
}

func readNative(executable, directory string, args []string) ([]byte, error) {
	child, err := startProcess(launchSpec{executable: executable, directory: directory, args: args, env: providerEnv()})
	if err != nil {
		return nil, errors.New("cannot inspect local Codex MCP configuration")
	}
	timer := time.AfterFunc(5*time.Second, func() { _ = child.Stop() })
	defer timer.Stop()
	data, readErr := io.ReadAll(io.LimitReader(child.Output(), 1<<20+1))
	child.Output().Close()
	if readErr != nil || len(data) > 1<<20 {
		_ = child.Stop()
		_ = child.Wait()
		return nil, errors.New("local Codex MCP configuration exceeds limit")
	}
	if child.Wait() != nil {
		return nil, errors.New("cannot inspect local Codex MCP configuration")
	}
	return data, nil
}

func parseMCPConnectors(data []byte) ([]mcpConnector, error) {
	var entries []struct {
		Name      string `json:"name"`
		Enabled   *bool  `json:"enabled"`
		Transport struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"transport"`
	}
	if json.Unmarshal(data, &entries) != nil || entries == nil || len(entries) > 128 {
		return nil, errors.New("invalid local Codex MCP configuration")
	}
	var connectors []mcpConnector
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Name == "" || len(entry.Name) > 128 || seen[entry.Name] || entry.Enabled == nil {
			return nil, errors.New("invalid local Codex MCP connector name")
		}
		if entry.Transport.Type != "stdio" && entry.Transport.Type != "streamable_http" {
			return nil, errors.New("unsupported local Codex MCP transport")
		}
		seen[entry.Name] = true
		connectors = append(connectors, mcpConnector{Name: entry.Name, Transport: entry.Transport.Type, Enabled: *entry.Enabled, Command: entry.Transport.Command, Args: entry.Transport.Args})
	}
	return connectors, nil
}
