package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BridgeConfig struct {
	ManagedRunID      string `json:"managed_run_id"`
	ManagedGeneration int64  `json:"managed_generation"`
	Server            string `json:"server"`
	Project           string `json:"project"`
	TokenFile         string `json:"token_file"`
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
	if decoder.Decode(new(any)) != io.EOF || !identifier.MatchString(config.ManagedRunID) || config.ManagedGeneration < 1 {
		return config, "", errors.New("managed bridge requires an immutable execution binding")
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
	if !identifier.MatchString(run.ID) || run.Generation < 1 {
		return nil, preparationFailureAt("prepare_binding", nil)
	}
	runDir := filepath.Join(e.config.StateDir, "runs", run.ID)
	if err := privateDirectory(runDir); err != nil {
		return nil, preparationFailureAt("prepare_run_directory", err)
	}
	dir := filepath.Join(runDir, "attempt-"+strconv.FormatInt(run.Generation, 10))
	if err := privateDirectory(dir); err != nil {
		return nil, preparationFailureAt("prepare_attempt_directory", err)
	}
	path := filepath.Join(dir, "bridge.json")
	data, _ := json.Marshal(BridgeConfig{ManagedRunID: run.ID, ManagedGeneration: run.Generation, Server: e.config.Server, Project: e.config.Project, TokenFile: e.config.TokenFile})
	if err := immutableBridgeFile(path, data); err != nil {
		return nil, preparationFailureAt("prepare_bridge_file", err)
	}
	self, err := os.Executable()
	if err != nil {
		return nil, preparationFailureAt("prepare_self", err)
	}
	args := []string{"worker", "bridge", "--config", path}
	if run.Provider == "codex" {
		// Codex merges maps instead of replacing them. Disable each discovered
		// connector for this invocation so another board session cannot be used.
		connectors, err := e.discover(e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path)
		if err != nil {
			return nil, preparationContext("prepare_discovery", err)
		}
		overrides, err := codexMCPOverrides(connectors, self, args, e.config.NativeMCPApprovedTools["codex"])
		if err != nil {
			return nil, err
		}
		if err = e.verifyMCP(e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path, overrides, self, args); err != nil {
			return nil, preparationContext("prepare_verify", err)
		}
		return overrides, nil
	}
	mcp := map[string]any{"mcpServers": map[string]any{"accord_managed": map[string]any{"command": self, "args": args}}}
	data, _ = json.Marshal(mcp)
	path = filepath.Join(dir, "mcp.json")
	if err = immutableBridgeFile(path, data); err != nil {
		return nil, preparationFailureAt("prepare_claude_file", err)
	}
	return append([]string{"--strict-mcp-config", "--mcp-config", path}, claudeMCPConsent(e.config.NativeMCPApprovedTools["claude"])...), nil
}

// Attempt-specific immutable files keep a surviving old bridge from observing
// the next generation's binding after a resume.
func immutableBridgeFile(path string, data []byte) error {
	if file, err := privateRead(path); err == nil {
		defer file.Close()
		old, e := io.ReadAll(io.LimitReader(file, 65537))
		if e != nil || !bytes.Equal(old, data) {
			return errors.New("immutable bridge configuration differs")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closed := file.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return syncDirectory(filepath.Dir(path))
}
