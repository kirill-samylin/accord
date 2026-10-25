//go:build linux

package worker

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// recoveryNeedsGroups: without recorded process groups a Unix census proves
// nothing, so verified recovery refuses.
const recoveryNeedsGroups = true

// scanLocalProcesses reads /proc for live processes of this user: argument
// vector, process group and current directory. Processes of other users are
// unreadable here and cannot be this worker's agents. Nothing is signalled.
func scanLocalProcesses(map[string]bool) ([]localProcess, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	self := os.Getpid()
	var out []localProcess
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || pid <= 1 || pid == self {
			continue
		}
		base := filepath.Join("/proc", entry.Name())
		stat, statErr := os.ReadFile(filepath.Join(base, "stat"))
		if statErr != nil {
			continue
		}
		end := bytes.LastIndexByte(stat, ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(stat[end+1:]))
		if len(fields) < 3 || fields[0] == "Z" || fields[0] == "X" {
			continue
		}
		pgid, _ := strconv.Atoi(fields[2])
		cmdline, cmdErr := os.ReadFile(filepath.Join(base, "cmdline"))
		if cmdErr != nil || len(cmdline) == 0 {
			continue // Gone, or a kernel thread.
		}
		var args []string
		for _, arg := range bytes.Split(bytes.TrimRight(cmdline, "\x00"), []byte{0}) {
			args = append(args, string(arg))
		}
		p := localProcess{PID: pid, PGID: pgid, Args: args}
		cwd, cwdErr := os.Readlink(filepath.Join(base, "cwd"))
		if errors.Is(cwdErr, fs.ErrPermission) {
			continue // Another user's process.
		}
		if cwdErr == nil {
			p.Cwd, p.CwdKnown = cwd, true
		}
		out = append(out, p)
	}
	return out, nil
}
