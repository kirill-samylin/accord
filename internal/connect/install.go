package connect

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"agent-board/internal/worker"
)

// privateDir creates the owner-only install directory.
func privateDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("private install directory is unavailable")
	}
	return os.Chmod(dir, 0700)
}

// installToken keeps the token in a private 0600 file the CLI configs point
// at. A token already at the target path, or identical content, is kept.
func installToken(source, token, target string) (string, error) {
	if canonical, err := filepath.EvalSymlinks(source); err == nil && canonical == target {
		return target, nil
	}
	if current, err := os.ReadFile(target); err == nil && bytes.Equal(bytes.TrimSpace(current), []byte(token)) {
		return target, os.Chmod(target, 0600)
	}
	next := target + ".next"
	file, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", errors.New("cannot write the private token copy")
	}
	if _, err = file.WriteString(token + "\n"); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(next)
		return "", err
	}
	if err = os.Rename(next, target); err != nil {
		os.Remove(next)
		return "", err
	}
	return target, nil
}

// installBinary copies this executable to a stable private path so the
// CLI configs survive moving or rebuilding the checkout.
func installBinary(source, target string) (string, error) {
	if a, err := os.Stat(source); err == nil {
		if b, err := os.Stat(target); err == nil && os.SameFile(a, b) {
			return target, nil
		}
	}
	if err := worker.ReplaceExecutable(source, target); err != nil {
		return "", err
	}
	return target, nil
}
