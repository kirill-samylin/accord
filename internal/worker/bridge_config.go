package worker

import (
	"agent-board/internal/automationipc"
	"agent-board/internal/core"
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
	Automation            *automationipc.Config `json:"automation,omitempty"`
	ExecutionCapabilityID string                `json:"execution_capability_id,omitempty"`
	ManagedRunID          string                `json:"managed_run_id"`
	ManagedGeneration     int64                 `json:"managed_generation"`
	Server                string                `json:"server"`
	Project               string                `json:"project"`
	TokenFile             string                `json:"token_file"`
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
	if err == nil && config.Automation != nil {
		b := config.Automation.Binding
		if b.RunID != config.ManagedRunID || b.Generation != config.ManagedGeneration || b.WorkflowID == "" || b.StepID == "" || b.ControlEpoch < 1 || len(config.Automation.Nonce) != 64 || config.Automation.Endpoint == "" || len(config.ExecutionCapabilityID) != 35 || !strings.HasPrefix(config.ExecutionCapabilityID, "ec_") || !strings.HasPrefix(token, core.ExecutionTokenPrefix) || len(token) != len(core.ExecutionTokenPrefix)+64 {
			return config, "", errors.New("invalid scoped automation bridge")
		}
	}
	if err == nil && config.Automation == nil && config.ExecutionCapabilityID != "" {
		return config, "", errors.New("unexpected execution capability")
	}
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

// bridgeLaunch is the provider-specific binding of one attempt: extra argv
// for Codex/Claude, or injected environment for Gennadiy (OpenCode has no
// MCP flag). Both are composed by the worker from local config only.
type bridgeLaunch struct {
	Args []string
	Env  []string
}

func (e *Engine) bridgeArgs(run RunState) (bridgeLaunch, error) {
	if !identifier.MatchString(run.ID) || run.Generation < 1 {
		return bridgeLaunch{}, preparationFailureAt("prepare_binding", nil)
	}
	runDir := filepath.Join(e.config.StateDir, "runs", run.ID)
	if err := privateDirectory(runDir); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_run_directory", err)
	}
	dir := filepath.Join(runDir, "attempt-"+strconv.FormatInt(run.Generation, 10))
	if err := privateDirectory(dir); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_attempt_directory", err)
	}
	path := filepath.Join(dir, "bridge.json")
	bridgeConfig := BridgeConfig{ManagedRunID: run.ID, ManagedGeneration: run.Generation, Server: e.config.Server, Project: e.config.Project, TokenFile: e.config.TokenFile}
	if run.Automation != nil {
		if !run.Automation.Registered || run.Automation.IPC == nil {
			return bridgeLaunch{}, preparationFailureAt("prepare_binding", nil)
		}
		bridgeConfig.TokenFile = run.Automation.TokenFile
		bridgeConfig.ExecutionCapabilityID = run.Automation.CapabilityID
		bridgeConfig.Automation = run.Automation.IPC
	}
	data, _ := json.Marshal(bridgeConfig)
	if err := immutableBridgeFile(path, data); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_bridge_file", err)
	}
	self, err := os.Executable()
	if err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_self", err)
	}
	args := []string{"worker", "bridge", "--config", path}
	switch run.Provider {
	case "codex":
		// Codex merges maps instead of replacing them. Disable each discovered
		// connector for this invocation so another board session cannot be used.
		connectors, err := e.discover(e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path)
		if err != nil {
			return bridgeLaunch{}, preparationContext("prepare_discovery", err)
		}
		e.recordConnectors(connectors)
		overrides, err := codexMCPOverrides(connectors, self, args, e.config.NativeMCPApprovedTools["codex"])
		if err != nil {
			return bridgeLaunch{}, err
		}
		if err = e.verifyMCP(e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path, overrides, self, args); err != nil {
			return bridgeLaunch{}, preparationContext("prepare_verify", err)
		}
		return bridgeLaunch{Args: overrides}, nil
	case "gennadiy":
		return e.gennadiyLaunch(run, self, args)
	}
	mcp := map[string]any{"mcpServers": map[string]any{"accord_managed": map[string]any{"command": self, "args": args}}}
	data, _ = json.Marshal(mcp)
	path = filepath.Join(dir, "mcp.json")
	if err = immutableBridgeFile(path, data); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_claude_file", err)
	}
	return bridgeLaunch{Args: append([]string{"--strict-mcp-config", "--mcp-config", path}, claudeMCPConsent(e.config.NativeMCPApprovedTools["claude"])...)}, nil
}

func (e *Engine) recordConnectors(connectors []mcpConnector) {
	for _, c := range connectors {
		e.preparedConnectors = append(e.preparedConnectors, mcpConnector{Name: c.Name, Transport: c.Transport})
	}
}

// gennadiyLaunch composes and verifies the injected OpenCode config for one
// attempt. Global config stays untouched; the preset lives in the environment
// of this child only and never in argv, a file in the workspace or the server.
func (e *Engine) gennadiyLaunch(run RunState, self string, args []string) (bridgeLaunch, error) {
	executable, directory := e.config.Executables[run.Provider], e.config.Workspaces[run.Workspace].Path
	connectors, err := e.discoverGennadiy(executable, directory)
	if err != nil {
		return bridgeLaunch{}, preparationContext("prepare_discovery", err)
	}
	e.recordConnectors(connectors)
	permission := gennadiyPermissions(gennadiyMode(e.config, run), e.config.NativeMCPApprovedTools["gennadiy"])
	content, err := gennadiyConfigContent(connectors, self, args, permission)
	if err != nil {
		return bridgeLaunch{}, err
	}
	if err = e.verifyGennadiy(executable, directory, content, self, args, permission); err != nil {
		return bridgeLaunch{}, preparationContext("prepare_verify", err)
	}
	return bridgeLaunch{Env: gennadiyEnv(content)}, nil
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
