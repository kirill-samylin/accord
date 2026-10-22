//go:build !windows

package worker

import (
	"errors"
	"os"
	"syscall"
)

func privateRead(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("worker file must be a private regular file")
	}
	return os.Open(path)
}

func privateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return errors.New("worker state directory must be private and not a symlink")
	}
	return nil
}

func lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
func unlockFile(file *os.File) { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }
