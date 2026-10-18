package store

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// prepareDatabase establishes privacy before SQLite's first I/O, so SQLite
// derives private WAL/SHM permissions from the already-private database.
// Existing files are never chmodded: unsafe paths fail closed. The directory
// owner and other processes under the same OS identity remain trusted.
func prepareDatabase(path string) (string, error) {
	if path == ":memory:" {
		return path, nil
	}
	if strings.HasPrefix(path, "file:") {
		return "", fmt.Errorf("database requires a filesystem path, not a SQLite URI")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(absolute)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return "", fmt.Errorf("database directory must not be group or other writable")
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		if err = privateRegularFile(absolute + suffix); err != nil {
			return "", err
		}
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		err = file.Close()
	} else if os.IsExist(err) {
		err = privateRegularFile(absolute)
	}
	if err != nil {
		return "", err
	}
	// Encode special filename characters so SQLite opens the exact checked path.
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String(), nil
}

func privateRegularFile(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return fmt.Errorf("database and sidecars must be regular files with mode 0600: %s", filepath.Base(path))
	}
	return nil
}
