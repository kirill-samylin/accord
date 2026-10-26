package worker

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// localProcess is one entry of a read-only process scan used only by
// `worker recover --unverified`. Nothing is ever signalled from a scan.
type localProcess struct {
	PID  int
	PGID int
	// Args is the argument vector when it was readable; it is matched but never
	// printed.
	Args []string
	// Cwd is the current directory; CwdKnown is false when it could not be read.
	Cwd      string
	CwdKnown bool
}

// providerProcessNames are executable base names of native providers: the
// fixed provider names plus every configured provider executable.
func providerProcessNames(config Config) map[string]bool {
	names := map[string]bool{}
	for _, provider := range nativeProviders {
		names[provider] = true
	}
	for provider, path := range config.Executables {
		names[strings.ToLower(provider)] = true
		if base := strings.ToLower(filepath.Base(path)); base != "" && base != "." && base != string(filepath.Separator) {
			names[strings.TrimSuffix(base, ".exe")] = true
		}
	}
	return names
}

// providerLike recognises a native provider process directly ("codex ...")
// or through a script interpreter ("node /.../claude ...").
func providerLike(p localProcess, names map[string]bool) bool {
	for i, arg := range p.Args {
		if i >= 3 {
			break
		}
		base := strings.TrimSuffix(strings.ToLower(filepath.Base(arg)), ".exe")
		if names[base] {
			return true
		}
	}
	return false
}

func pathWithin(path, root string) bool {
	if root == "" || path == "" {
		return false
	}
	path, root = filepath.Clean(path), filepath.Clean(root)
	if path == root {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(root, string(filepath.Separator))+string(filepath.Separator))
}

// refuseRunProcesses fails when any scanned process may still belong to the
// run: any process naming its native session ID, or a native provider process
// whose arguments name the workspace or whose current directory is inside it.
// A provider process whose directory cannot be read also refuses, because it
// cannot be excluded.
func refuseRunProcesses(processes []localProcess, workspace, sessionID string, names map[string]bool) error {
	var matched, unknown []int
	for _, p := range processes {
		session := false
		if sessionID != "" {
			for _, arg := range p.Args {
				if strings.Contains(arg, sessionID) {
					session = true
					break
				}
			}
		}
		if session {
			matched = append(matched, p.PID)
			continue
		}
		if !providerLike(p, names) {
			continue
		}
		hit := false
		if workspace != "" {
			for _, arg := range p.Args {
				if strings.Contains(arg, workspace) {
					hit = true
					break
				}
			}
		}
		if !hit && p.CwdKnown && pathWithin(p.Cwd, workspace) {
			hit = true
		}
		switch {
		case hit:
			matched = append(matched, p.PID)
		case !p.CwdKnown:
			unknown = append(unknown, p.PID)
		}
	}
	sort.Ints(matched)
	sort.Ints(unknown)
	if len(matched) > 0 {
		return fmt.Errorf("native provider processes %v still reference this run's workspace or native session; stop them and retry (nothing was changed)", matched)
	}
	if len(unknown) > 0 {
		return fmt.Errorf("cannot read the working directory of native provider processes %v, so they cannot be excluded; stop them or retry (nothing was changed)", unknown)
	}
	return nil
}
