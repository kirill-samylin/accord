//go:build windows

package worker

// Native and check trees run in Job Objects with kill-on-close held only by the
// worker process. The caller holds the exclusive state lock, so that worker is
// gone and the kernel terminated every assigned process.
func processGroupsAlive([]int) ([]int, error) { return nil, nil }

// recoveryNeedsGroups is false: Job Object kill-on-close, not a recorded
// group census, proves that no tree of the previous worker survived.
const recoveryNeedsGroups = false

// scanLocalProcesses has nothing to add on Windows for the same reason.
func scanLocalProcesses(map[string]bool) ([]localProcess, error) { return nil, nil }
