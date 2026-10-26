package worker

// Exported seams for `agent-board connect`, which reuses the local CLI
// discovery and private-file helpers of the worker without its engine.

// NativeCLI is one locally installed native CLI acceptable to the worker.
type NativeCLI struct {
	Provider string
	Path     string
	Version  string
	Note     string
}

// DiscoverNativeCLIs lists codex/claude/gennadiy installations on this
// computer, PATH first, each with one bounded `--version` read.
func DiscoverNativeCLIs() []NativeCLI {
	var out []NativeCLI
	found := defaultSetupProbe().discover()
	for _, provider := range nativeProviders {
		for _, cli := range found[provider] {
			out = append(out, NativeCLI{Provider: cli.Provider, Path: cli.Path, Version: cli.Version, Note: cli.Note})
		}
	}
	return out
}

// Hints are the non-secret connection defaults the GUI client leaves in
// connection.json beside a token file.
type Hints struct{ Origin, ProjectID, Machine string }

func ConnectionHints(tokenFile string) Hints {
	h := readConnectionHints(tokenFile)
	return Hints{Origin: h.Origin, ProjectID: h.ProjectID, Machine: h.Machine}
}

// ReplaceExecutable installs a copy of source at target (0700) atomically.
func ReplaceExecutable(source, target string) error { return replaceExecutable(source, target) }

// ProviderEnv is the process environment without Accord credentials.
func ProviderEnv() []string { return providerEnv() }
