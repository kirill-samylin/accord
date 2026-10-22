//go:build windows

package worker

import "golang.org/x/sys/windows"

func replaceFile(from, to string) error {
	source, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// Every file write is FlushFileBuffers'ed before atomic same-volume replacement.
// Directory FlushFileBuffers is unsupported on Windows.
func syncDirectory(string) error { return nil }
