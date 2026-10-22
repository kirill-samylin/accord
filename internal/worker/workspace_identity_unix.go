//go:build darwin || linux

package worker

import (
	"fmt"
	"os"
	"syscall"
)

func directoryIdentity(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	stat := info.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%x-%x", stat.Dev, stat.Ino), nil
}
