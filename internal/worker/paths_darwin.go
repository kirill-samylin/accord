//go:build darwin

package worker

import (
	"errors"
	"golang.org/x/sys/unix"
	"strings"
)

func platformLocalPath(path string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return err
	}
	var fs strings.Builder
	for _, b := range stat.Fstypename {
		if b == 0 {
			break
		}
		fs.WriteByte(byte(b))
	}
	switch fs.String() {
	case "nfs", "smbfs", "afpfs", "webdav":
		return errors.New("network filesystem paths are not supported")
	}
	return nil
}
