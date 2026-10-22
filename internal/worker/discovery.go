package worker

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

func discoverCodexMCP(executable, directory string) ([]string, error) {
	data, err := readNative(executable, directory, []string{"mcp", "list", "--json"})
	if err != nil {
		return nil, err
	}
	return parseMCPNames(data)
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

func parseMCPNames(data []byte) ([]string, error) {
	var entries []struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(data, &entries) != nil || entries == nil || len(entries) > 128 {
		return nil, errors.New("invalid local Codex MCP configuration")
	}
	var names []string
	for _, entry := range entries {
		if entry.Name == "" || len(entry.Name) > 128 {
			return nil, errors.New("invalid local Codex MCP connector name")
		}
		names = append(names, entry.Name)
	}
	return names, nil
}
