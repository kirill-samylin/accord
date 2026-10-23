package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"agent-board/internal/core"
)

type permissionProof struct {
	Digest           string `json:"digest"`
	ExecutableSHA256 string `json:"executable_sha256"`
	GitIdentity      string `json:"git_identity"`
}
type configInspection struct {
	Directory        string   `json:"directory"`
	Profile          string   `json:"profile"`
	ExecutableSHA256 string   `json:"executable_sha256"`
	GitIdentity      string   `json:"git_identity"`
	Connectors       []string `json:"connectors"`
}

var errPermissionProof = errors.New("native Git permission inspection blocked; effective local policy could not be proven")

func permissionError() error { return errPermissionProof }
func gitMetadataMode(c Config, run core.ManagedRun) bool {
	return run.Automation != nil && run.Profile != nil && run.Profile.Role == "developer" && run.Provider == "codex" && c.Automation != nil && c.Automation.Workspaces[run.Workspace].CodexGitMetadataWrites
}
func gitDirectoryIdentity(directory string) (string, error) {
	git := filepath.Join(directory, ".git")
	if validateLocalPath(git) != nil {
		return "", permissionError()
	}
	info, err := os.Lstat(git)
	if err != nil || !info.IsDir() {
		return "", permissionError()
	}
	// A linked worktree and alternate common directory need a separate consent.
	for _, name := range []string{"commondir", "gitdir"} {
		if _, err = os.Lstat(filepath.Join(git, name)); !os.IsNotExist(err) {
			return "", permissionError()
		}
	}
	for _, name := range []string{"config", "hooks"} {
		path := filepath.Join(git, name)
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 || validateLocalPath(path) != nil {
				return "", permissionError()
			}
		} else if !os.IsNotExist(err) {
			return "", permissionError()
		}
	}
	return directoryIdentity(git)
}
func executableDigest(path string) (string, error) {
	if validateLocalPath(path) != nil {
		return "", permissionError()
	}
	f, err := os.Open(path)
	if err != nil {
		return "", permissionError()
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil || !i.Mode().IsRegular() || i.Size() > 1<<30 {
		return "", permissionError()
	}
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, 1<<30+1)); err != nil {
		return "", permissionError()
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func permissionProfile(i configInspection) map[string]any {
	return map[string]any{"extends": ":workspace", "filesystem": map[string]any{filepath.Join(i.Directory, ".git"): map[string]any{".": "write", "config": "read", "hooks": "read"}}, "network": map[string]any{"enabled": false}}
}
func gitPermissionOverrides(i configInspection) []string {
	git := strconv.Quote(filepath.Join(i.Directory, ".git"))
	profile := fmt.Sprintf("permissions={%s={extends=\":workspace\",filesystem={%s={\".\"=\"write\",config=\"read\",hooks=\"read\"}},network={enabled=false}}}", strconv.Quote(i.Profile), git)
	return []string{"-c", "default_permissions=" + strconv.Quote(i.Profile), "-c", profile, "-c", "approval_policy=\"never\""}
}
func newConfigInspection(directory, executable string, connectors []mcpConnector) (configInspection, []string, error) {
	i := configInspection{Directory: directory}
	var err error
	i.GitIdentity, err = gitDirectoryIdentity(directory)
	if err != nil {
		return i, nil, err
	}
	i.ExecutableSHA256, err = executableDigest(executable)
	if err != nil {
		return i, nil, err
	}
	h := sha256.Sum256([]byte(directory + "\x00" + i.GitIdentity))
	i.Profile = "accord_git_" + hex.EncodeToString(h[:16])
	var tables []string
	seen := map[string]bool{}
	for _, c := range connectors {
		if c.Name == "" || seen[c.Name] || len(c.Name) > 128 {
			return i, nil, permissionError()
		}
		seen[c.Name] = true
		i.Connectors = append(i.Connectors, c.Name)
		switch c.Transport {
		case "stdio":
			tables = append(tables, strconv.Quote(c.Name)+"={enabled=false,command="+strconv.Quote(executable)+",args=[]}")
		case "streamable_http":
			tables = append(tables, strconv.Quote(c.Name)+"={enabled=false,url=\"http://127.0.0.1:1\"}")
		default:
			return i, nil, permissionError()
		}
	}
	args := gitPermissionOverrides(i)
	slices.Sort(i.Connectors)
	slices.Sort(tables)
	// Empty maps do not replace lower-layer maps. The parser separately proves
	// every inherited connector is known and disabled after Codex merges them.
	args = append(args, "-c", "mcp_servers={"+joinTables(tables)+"}")
	return i, args, nil
}
func joinTables(s []string) string {
	var out string
	for n, v := range s {
		if n > 0 {
			out += ","
		}
		out += v
	}
	return out
}
func permissionDigest(value any) string {
	b, _ := json.Marshal(value)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
