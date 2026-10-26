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
	stdin      io.Reader // Only helper-owned finite config dialogue uses this pipe.
	// mergeStderr captures stderr into Output for bounded config-only reads of
	// a CLI that prints help there (Gennadiy). Windows always merges both.
	mergeStderr bool
	inspection  *configInspection
	// onGroup receives the native process group a Unix supervisor reports, so
	// `worker recover` can census it later. It is called without Engine.mu.
	onGroup func(int)
}

// Native verification needs the observed OS exit status, not a fabricated code
// derived from success/failure. Older test seams need not implement this method.
func processExitCode(child process) (int, bool) {
	if status, ok := child.(interface{ ExitCode() (int, bool) }); ok {
		return status.ExitCode()
	}
	return 0, false
}
