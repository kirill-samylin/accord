package worker

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agent-board/internal/transport"
)

// readSetupConfig loads an existing private configuration without running
// Validate, so a file that no longer validates (e.g. a replaced binary) can
// still be edited. A missing file is a fresh setup; an unreadable or
// malformed one is refused rather than silently overwritten.
func readSetupConfig(path string) (Config, bool, error) {
	var config Config
	if !filepath.IsAbs(path) {
		return config, false, errors.New("worker setup requires an absolute --config path")
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return config, false, nil
	}
	file, err := privateRead(path)
	if err != nil {
		return config, true, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 || transport.ValidateJSONFields(data) != nil {
		return config, true, errors.New("existing worker configuration is invalid; fix or move it before setup")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || decoder.Decode(new(any)) != io.EOF {
		return config, true, errors.New("existing worker configuration is invalid; fix or move it before setup")
	}
	return config, true, nil
}

// writeSetupConfig writes the validated configuration atomically with mode
// 0600 (a private parent directory is created when missing). The previous
// file is replaced only after the new one is fully synced.
func writeSetupConfig(path string, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if _, err = os.Lstat(dir); os.IsNotExist(err) {
		if err = privateDirectory(dir); err != nil {
			return err
		}
	}
	next := path + ".next"
	file, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return errors.New("an incomplete setup write exists next to the configuration; remove the .next file first")
	}
	if err != nil {
		return err
	}
	if _, err = file.Write(append(data, '\n')); err == nil {
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
	if err = replaceFile(next, path); err != nil {
		os.Remove(next)
		return err
	}
	return syncDirectory(dir)
}

// replaceExecutable installs a copy of `source` at `target` (0700) through a
// same-directory temporary file, so a running service never sees a partial
// binary. Used only after the owner's explicit confirmation.
func replaceExecutable(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	next := target + ".next"
	out, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(next)
		return err
	}
	if err = replaceFile(next, target); err != nil {
		os.Remove(next)
		return err
	}
	return syncDirectory(filepath.Dir(target))
}
