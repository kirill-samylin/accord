//go:build windows

package worker

import (
	"errors"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
)

func platformLocalPath(path string) error {
	if strings.HasPrefix(path, `\\`) {
		return errors.New("UNC/network paths are not supported")
	}
	encoded, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attrs, err := windows.GetFileAttributes(encoded)
	if err != nil {
		return err
	}
	if attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return errors.New("reparse paths are not supported")
	}
	root, err := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return err
	}
	if windows.GetDriveType(root) == windows.DRIVE_REMOTE {
		return errors.New("network drives are not supported")
	}
	return nil
}
