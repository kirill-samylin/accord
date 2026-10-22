package worker

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BridgeConfig struct {
	Server    string `json:"server"`
	Project   string `json:"project"`
	TokenFile string `json:"token_file"`
}

// ReadBridgeConfig is used only by a per-run stdio bridge child. The host proof
// is never put in this file or inherited by provider processes.
func ReadBridgeConfig(path string) (BridgeConfig, string, error) {
	var config BridgeConfig
	file, err := privateRead(path)
	if err != nil {
		return config, "", err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 65537))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&config); err != nil {
		return config, "", errors.New("invalid local bridge configuration")
	}
	token, err := ReadToken(config.TokenFile)
	return config, token, err
}

func ReadToken(path string) (string, error) {
	if path == "" {
		return "", errors.New("worker requires a private token file")
	}
	file, err := privateRead(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	token := strings.TrimSpace(string(data))
	if err != nil || token == "" || len(data) > 4096 || strings.ContainsAny(token, "\r\n \t") {
		return "", errors.New("invalid private worker credential")
	}
	return token, nil
}

func (e *Engine) bridgeArgs(run RunState) ([]string, error) {
	dir := filepath.Join(e.config.StateDir, "runs", run.ID)
	if err := privateDirectory(dir); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "bridge.json")
	data, _ := json.Marshal(BridgeConfig{Server: e.config.Server, Project: e.config.Project, TokenFile: e.config.TokenFile})
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := []string{"worker", "bridge", "--config", path}
	if run.Provider == "codex" {
		// Codex merges maps instead of replacing them. Disable each discovered
		// connector for this invocation so another board session cannot be used.
		names, err := e.discover(e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path)
		if err != nil {
			return nil, err
		}
		var overrides []string
		for _, name := range names {
			if name != "accord_managed" {
				overrides = append(overrides, "-c", "mcp_servers."+strconv.Quote(name)+".enabled=false")
			}
		}
		encoded, _ := json.Marshal(args)
		return append(overrides, "-c", "mcp_servers.accord_managed={command="+strconv.Quote(self)+",args="+string(encoded)+",enabled=true}"), nil
	}
	mcp := map[string]any{"mcpServers": map[string]any{"accord_managed": map[string]any{"command": self, "args": args}}}
	data, _ = json.Marshal(mcp)
	path = filepath.Join(dir, "mcp.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	return []string{"--strict-mcp-config", "--mcp-config", path}, nil
}
