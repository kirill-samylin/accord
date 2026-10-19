//go:build !darwin && !linux

package web

import (
	"errors"
	"os"
)

// The native client is portable; the private file server requires Unix owner
// and no-follow guarantees. Unsupported server platforms fail closed at startup.
const designNoFollow = 0

func designOwned(os.FileInfo) bool { return false }
func designFreeDisk(string) (uint64, error) {
	return 0, errors.New("web file server requires Linux or macOS")
}
