//go:build windows

package worker

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func directoryIdentity(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var info windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
}
