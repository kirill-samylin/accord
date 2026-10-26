package worker

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// discoveredCLI is one locally installed native CLI that Config.Validate would
// accept: an absolute canonical path to a regular file, never a shell shim.
type discoveredCLI struct {
	Provider string
	Path     string
	Version  string
	// Note explains a resolved shim (e.g. the npm launcher of gennadiy).
	Note string
}

// setupProbe finds native CLIs on this computer only. PATH, HOME and the
// version reader are injectable so tests use fake files; production uses
// the process environment and a bounded `--version` run.
type setupProbe struct {
	pathDirs     []string
	systemDirs   []string
	home         string
	applications string
	goos         string
	goarch       string
	version      func(provider, executable string) (string, error)
}

func defaultSetupProbe() setupProbe {
	home, _ := os.UserHomeDir()
	return setupProbe{pathDirs: filepath.SplitList(os.Getenv("PATH")), systemDirs: []string{"/opt/homebrew/bin", "/usr/local/bin"}, home: home, applications: "/Applications", goos: runtime.GOOS, goarch: runtime.GOARCH, version: readSetupVersion}
}

// readSetupVersion runs `--version` once with the same bounded reader the
// worker uses for config-only invocations; the owner's credentials never
// reach the child (providerEnv strips Accord variables).
func readSetupVersion(provider, executable string) (string, error) {
	directory, err := os.UserHomeDir()
	if err != nil || directory == "" {
		directory = os.TempDir()
	}
	env := providerEnv()
	if provider == "gennadiy" {
		env = append(env, gennadiyExtraEnv...)
	}
	data, err := readNativeWithEnv("setup", executable, directory, []string{"--version"}, env, 5*time.Second, startProcess)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// firstLine keeps only the first bounded line of a version banner.
func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if len(line) > 80 {
		line = line[:80]
	}
	return strings.TrimSpace(line)
}

// gennadiyPlatformPackage is the npm package holding the real Mach-O/ELF
// binary of the corporate OpenCode fork; the launcher in PATH is a script.
func (p setupProbe) gennadiyPlatformPackage() string {
	arch := p.goarch
	if arch == "amd64" {
		arch = "x64"
	}
	return "gennadiy-" + p.goos + "-" + arch
}

func (p setupProbe) executableName(provider string) string {
	if p.goos == "windows" {
		return provider + ".exe"
	}
	return provider
}

// candidates lists where each CLI is usually installed. PATH comes first so
// the owner's own choice wins; typical locations cover GUI bundles and
// per-user installers that never touch PATH of a LaunchAgent.
func (p setupProbe) candidates(provider string) []string {
	name := p.executableName(provider)
	var dirs []string
	dirs = append(dirs, p.pathDirs...)
	dirs = append(dirs, p.systemDirs...)
	dirs = append(dirs, filepath.Join(p.home, ".local", "bin"), filepath.Join(p.home, ".volta", "bin"))
	if nodes, err := filepath.Glob(filepath.Join(p.home, ".nvm", "versions", "node", "*", "bin")); err == nil {
		dirs = append(dirs, nodes...)
	}
	var out []string
	for _, dir := range dirs {
		if dir != "" {
			out = append(out, filepath.Join(dir, name))
		}
	}
	switch provider {
	case "codex":
		if p.applications != "" {
			out = append(out, filepath.Join(p.applications, "ChatGPT.app", "Contents", "Resources", "codex-cli", "CodexCLI.app", "Contents", "MacOS", "codex"))
		}
		out = append(out, filepath.Join(p.home, ".codex", "bin", name))
	case "claude":
		out = append(out, filepath.Join(p.home, ".claude", "local", name), filepath.Join(p.home, ".claude", "local", "node_modules", ".bin", name))
	case "gennadiy":
		out = append(out, filepath.Join(p.home, ".gennadiy", "bin", name), filepath.Join(p.home, ".gennadiy", "node_modules", p.gennadiyPlatformPackage(), "bin", name))
	}
	return out
}

// isScript reports a shebang launcher. Such a file is executable for the
// shell but is not the native binary the worker pins by inode.
func isScript(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	head := make([]byte, 2)
	n, _ := file.Read(head)
	return n == 2 && bytes.Equal(head[:n], []byte("#!"))
}

// resolveGennadiyShim maps the npm launcher `node_modules/gennadiy/bin/*.cjs`
// to the platform package binary beside it. The bash shim in ~/.gennadiy/bin
// only re-executes whatever PATH holds, so it resolves through PATH instead.
func (p setupProbe) resolveGennadiyShim(script string) string {
	marker := string(filepath.Separator) + filepath.Join("node_modules", "gennadiy") + string(filepath.Separator)
	index := strings.Index(script, marker)
	if index < 0 {
		return ""
	}
	root := script[:index+len(marker)-1]
	return filepath.Join(root, "node_modules", p.gennadiyPlatformPackage(), "bin", p.executableName("gennadiy"))
}

// resolve turns a candidate into a Validate-acceptable path or "" with a note.
func (p setupProbe) resolve(provider, candidate string) (path, note string) {
	canonical, err := filepath.EvalSymlinks(candidate)
	if err != nil || !filepath.IsAbs(canonical) {
		return "", ""
	}
	lower := strings.ToLower(canonical)
	launcher := isScript(canonical) || strings.HasSuffix(lower, ".cjs") || strings.HasSuffix(lower, ".js")
	if launcher {
		if provider != "gennadiy" {
			return "", ""
		}
		real := p.resolveGennadiyShim(canonical)
		if real == "" {
			return "", ""
		}
		if canonical, err = filepath.EvalSymlinks(real); err != nil || isScript(canonical) {
			return "", ""
		}
		note = "настоящий бинарь из npm-пакета " + p.gennadiyPlatformPackage() + "; шим " + candidate + " не используется"
	}
	lower = strings.ToLower(canonical)
	if strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cjs") || strings.HasSuffix(lower, ".js") {
		return "", ""
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.Mode().IsRegular() || validateLocalPath(canonical) != nil {
		return "", ""
	}
	return canonical, note
}

// discover returns the acceptable installations per provider, PATH order
// first, deduplicated by canonical path, with a version when readable.
func (p setupProbe) discover() map[string][]discoveredCLI {
	found := map[string][]discoveredCLI{}
	for _, provider := range nativeProviders {
		seen := map[string]bool{}
		for _, candidate := range p.candidates(provider) {
			path, note := p.resolve(provider, candidate)
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			entry := discoveredCLI{Provider: provider, Path: path, Note: note}
			if p.version != nil {
				if version, err := p.version(provider, path); err == nil && firstLine(version) != "" {
					entry.Version = firstLine(version)
				} else {
					entry.Version = "версия не прочитана"
				}
			}
			found[provider] = append(found[provider], entry)
		}
	}
	return found
}

// describe renders a discovered CLI for the dialogue list.
func (c discoveredCLI) describe() string {
	text := c.Path
	if c.Version != "" {
		text += " — " + c.Version
	}
	if c.Note != "" {
		text += " (" + c.Note + ")"
	}
	return text
}

func sortedProviders(executables map[string]string) []string {
	var providers []string
	for provider := range executables {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}
