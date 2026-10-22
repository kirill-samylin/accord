//go:build linux

package worker

import (
	"errors"
	"golang.org/x/sys/unix"
)

func platformLocalPath(path string) error {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return err
	}
	switch uint64(stat.Type) {
	case 0x6969, 0xff534d42, 0xfe534d42, 0x9fa0, 0x564c, 0x73757245:
		return errors.New("network filesystem paths are not supported")
	}
	return nil
}
