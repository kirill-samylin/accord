package connect

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// writeGennadiyConnector adds or replaces mcp.accord in the owner's
// gennadiy.json (OpenCode format: type local, command array). Every other key,
// including corporate connectors with their headers, is written back as read.
// A file that is not plain JSON is left alone and reported for manual editing.
func writeGennadiyConnector(path string, command []string) error {
	config := map[string]any{}
	mode := os.FileMode(0600)
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		if len(bytes.TrimSpace(data)) > 0 && json.Unmarshal(data, &config) != nil {
			return errors.New("gennadiy.json is not plain JSON")
		}
	case os.IsNotExist(err):
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	default:
		return err
	}
	mcp, _ := config["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
	}
	mcp[ServerName] = map[string]any{"type": "local", "command": command, "enabled": true}
	config["mcp"] = mcp
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	next := path + ".next"
	file, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return errors.New("cannot write gennadiy.json")
	}
	if _, err = file.Write(append(out, '\n')); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(next)
		return err
	}
	if err = os.Rename(next, path); err != nil {
		os.Remove(next)
		return err
	}
	return nil
}

func gennadiySnippet(command []string) string {
	data, _ := json.Marshal(map[string]any{"mcp": map[string]any{ServerName: map[string]any{"type": "local", "command": command, "enabled": true}}})
	return string(data)
}
