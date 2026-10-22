//go:build !windows

package worker

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }

func syncDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
