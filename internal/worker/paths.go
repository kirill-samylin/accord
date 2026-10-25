package worker

import (
	"errors"
	"os"
	"path/filepath"
)

type pathPin struct {
	Path     string `json:"path"`
	Identity string `json:"identity"`
}

func validateLocalPath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("local consent requires an absolute path")
	}
	current := filepath.Clean(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("locally allowed path is unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink paths are not allowed; configure the canonical local path")
		}
		if err = platformLocalPath(current); err != nil {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return nil
}

func pinPaths(config Config) (map[string]pathPin, error) {
	pins := map[string]pathPin{}
	for provider, path := range config.Executables {
		if err := validateLocalPath(path); err != nil {
			return nil, err
		}
		id, err := directoryIdentity(path)
		if err != nil {
			return nil, err
		}
		pins["provider:"+provider] = pathPin{Path: path, Identity: id}
	}
	for alias, workspace := range config.Workspaces {
		if err := validateLocalPath(workspace.Path); err != nil {
			return nil, err
		}
		id, err := directoryIdentity(workspace.Path)
		if err != nil {
			return nil, err
		}
		pins["workspace:"+alias] = pathPin{Path: workspace.Path, Identity: id}
	}
	return pins, nil
}

// pinError names which local pin changed: the provider executable (for example
// an auto-updated CLI with a new inode) or the workspace directory. The text
// keeps "local workspace or provider identity changed" for owner guidance.
type pinError struct{ kind, name string }

func (e pinError) Error() string {
	if e.kind == "provider" {
		return "local workspace or provider identity changed: provider executable " + e.name + " differs from the consented pin; run `agent-board worker consent`"
	}
	return "local workspace or provider identity changed: workspace " + e.name + " differs from the consented pin; run `agent-board worker consent`"
}

// Code is the stable diagnostic for the changed pin.
func (e pinError) Code() string { return e.kind + "_pin_changed" }

func (e *Engine) checkPins(provider, alias string) error {
	// Device+inode pins are unchanged; the order only makes the cause stable.
	for _, pin := range []struct{ kind, name, key, path string }{{"provider", provider, "provider:" + provider, e.config.Executables[provider]}, {"workspace", alias, "workspace:" + alias, e.config.Workspaces[alias].Path}} {
		saved, ok := e.journal.PathPins[pin.key]
		if !ok || saved.Path != pin.path || validateLocalPath(pin.path) != nil {
			return pinError{pin.kind, pin.name}
		}
		identity, err := directoryIdentity(pin.path)
		if err != nil || identity != saved.Identity {
			return pinError{pin.kind, pin.name}
		}
	}
	return nil
}
