package worker

import (
	"errors"
	"strings"
)

func probeProviders(config Config) error {
	for provider, executable := range config.Executables {
		var directory string
		for _, workspace := range config.Workspaces {
			for _, allowed := range workspace.Providers {
				if allowed == provider {
					directory = workspace.Path
				}
			}
		}
		if directory == "" {
			return errors.New("configured provider is not assigned to a workspace")
		}
		version, err := readNative(executable, directory, []string{"--version"})
		if err != nil || len(version) == 0 {
			return errors.New("native provider version preflight failed")
		}
		args := []string{"--help"}
		flags := []string{"--print", "--resume", "--output-format", "--strict-mcp-config", "--permission-mode", "--verbose"}
		if provider == "codex" {
			args = []string{"exec", "--help"}
			flags = []string{"--json", "resume"}
		}
		help, err := readNative(executable, directory, args)
		if err != nil {
			return errors.New("native provider CLI preflight failed")
		}
		for _, flag := range flags {
			if !strings.Contains(string(help), flag) {
				return errors.New("installed native provider does not support the required managed flags")
			}
		}
	}
	return nil
}
