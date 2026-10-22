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

func (e *Engine) checkPins(provider, alias string) error {
	for key, path := range map[string]string{"provider:" + provider: e.config.Executables[provider], "workspace:" + alias: e.config.Workspaces[alias].Path} {
		pin, ok := e.journal.PathPins[key]
		if !ok || pin.Path != path {
			return errors.New("local path mapping changed; explicit worker consent is required")
		}
		if err := validateLocalPath(path); err != nil {
			return err
		}
		identity, err := directoryIdentity(path)
		if err != nil || identity != pin.Identity {
			return errors.New("local path identity changed; explicit worker consent is required")
		}
	}
	return nil
}
