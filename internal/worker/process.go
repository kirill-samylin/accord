package worker

import (
	"errors"
	"io"
)

var errOwnership = errors.New("native process ownership is uncertain")
var errNativeExit = errors.New("native provider exited unsuccessfully")

// process keeps an operating-system ownership handle for its entire lifetime.
// The caller never adopts a PID from disk or from the remote server.
type process interface {
	Output() io.ReadCloser
	Wait() error
	Stop() error
	Forced() bool
}

type launchSpec struct {
	executable string
	directory  string
	args       []string
	env        []string
	prompt     string
}
