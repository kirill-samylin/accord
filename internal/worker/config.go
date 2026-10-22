// Package worker runs only locally consented native agents for one Accord project.
package worker

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"agent-board/internal/transport"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var sessionID = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

type Workspace struct {
	Path      string   `json:"path"`
	Providers []string `json:"providers"`
}

// Config is a local allowlist. Remote commands cannot supply executable paths,
// working directories, environment, provider flags, or permission overrides.
type Config struct {
	NativeMCPApprovedTools map[string][]string  `json:"native_mcp_approved_tools,omitempty"`
	Server                 string               `json:"server"`
	Project                string               `json:"project"`
	TokenFile              string               `json:"token_file"`
	StateDir               string               `json:"state_dir"`
	Name                   string               `json:"name"`
	Machine                string               `json:"machine"`
	Controllers            []string             `json:"controllers"`
	Executables            map[string]string    `json:"executables"`
	Models                 map[string][]string  `json:"models,omitempty"`
	CodexSandbox           string               `json:"codex_sandbox,omitempty"`
	ClaudePermissionMode   string               `json:"claude_permission_mode,omitempty"`
	Workspaces             map[string]Workspace `json:"workspaces"`
}

func LoadConfig(path string) (Config, error) {
	var config Config
	file, err := privateRead(path)
	if err != nil {
		return config, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 || transport.ValidateJSONFields(data) != nil {
		return config, errors.New("invalid or duplicate worker configuration fields")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&config); err != nil {
		return config, errors.New("invalid worker configuration")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return config, errors.New("trailing worker configuration")
	}
	return config, config.Validate()
}

func (c Config) Validate() error {
	if err := validateNativeMCPConsent(c.NativeMCPApprovedTools, c.Executables); err != nil {
		return err
	}
	if c.CodexSandbox != "" && c.CodexSandbox != "read-only" && c.CodexSandbox != "workspace-write" {
		return errors.New("codex_sandbox must be read-only or workspace-write")
	}
	if c.ClaudePermissionMode != "" && c.ClaudePermissionMode != "default" && c.ClaudePermissionMode != "acceptEdits" {
		return errors.New("claude_permission_mode must be default or acceptEdits")
	}
	if !identifier.MatchString(c.Name) || c.Machine == "" || !filepath.IsAbs(c.StateDir) || len(c.Workspaces) == 0 || len(c.Workspaces) > 32 {
		return errors.New("worker requires a name, absolute state_dir, and 1..32 workspaces")
	}
	if len(c.Controllers) > 32 {
		return errors.New("at most 32 explicit local controller principal IDs are allowed")
	}
	if len(c.Executables) == 0 || len(c.Executables) > 2 {
		return errors.New("configure codex and/or claude executables")
	}
	for provider, path := range c.Executables {
		if err := validateLocalPath(path); err != nil {
			return err
		}
		if provider != "codex" && provider != "claude" {
			return errors.New("unsupported native provider")
		}
		if !filepath.IsAbs(path) {
			return errors.New("provider executable must be an absolute path")
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("provider executable is unavailable")
		}
		if strings.HasSuffix(strings.ToLower(path), ".cmd") || strings.HasSuffix(strings.ToLower(path), ".bat") {
			return errors.New("use a native executable, shell shims are unsupported")
		}
	}
	for alias, workspace := range c.Workspaces {
		if err := validateLocalPath(workspace.Path); err != nil {
			return err
		}
		if !identifier.MatchString(alias) || !filepath.IsAbs(workspace.Path) || len(workspace.Providers) == 0 {
			return errors.New("invalid workspace allowlist")
		}
		info, err := os.Stat(workspace.Path)
		if err != nil || !info.IsDir() {
			return errors.New("workspace is unavailable")
		}
		for _, provider := range workspace.Providers {
			if c.Executables[provider] == "" {
				return errors.New("workspace refers to an unconfigured provider")
			}
		}
	}
	return nil
}

func (c Config) launch(provider, alias string) (string, string, error) {
	workspace, ok := c.Workspaces[alias]
	if !ok {
		return "", "", errors.New("workspace alias is not locally allowed")
	}
	for _, allowed := range workspace.Providers {
		if provider == allowed && c.Executables[provider] != "" {
			return c.Executables[provider], workspace.Path, nil
		}
	}
	return "", "", errors.New("provider is not locally allowed for this workspace")
}
