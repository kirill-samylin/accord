//go:build darwin

package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// recoveryNeedsGroups: without recorded process groups a Unix census proves
// nothing, so verified recovery refuses.
const recoveryNeedsGroups = true

// scanLocalProcesses lists live processes of this user with their argument
// vectors (kern.procargs2). Working directories are read only for native
// provider candidates, through the fixed system lsof; nothing is signalled.
func scanLocalProcesses(names map[string]bool) ([]localProcess, error) {
	entries, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	uid, self := uint32(os.Getuid()), os.Getpid()
	var out []localProcess
	for _, entry := range entries {
		pid := int(entry.Proc.P_pid)
		if pid <= 1 || pid == self || entry.Proc.P_stat == 5 || entry.Eproc.Ucred.Uid != uid {
			continue
		}
		args, argErr := darwinProcessArgs(pid)
		if argErr != nil {
			// An exiting or protected process: its short command name still
			// identifies a provider, whose directory lsof then checks.
			args = []string{unix.ByteSliceToString(entry.Proc.P_comm[:])}
		}
		out = append(out, localProcess{PID: pid, PGID: int(entry.Eproc.Pgid), Args: args})
	}
	fillWorkingDirectories(out, func(p localProcess) bool { return providerLike(p, names) })
	return out, nil
}

func darwinProcessArgs(pid int) ([]string, error) {
	buf, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return nil, err
	}
	if len(buf) < 4 {
		return nil, unix.EINVAL
	}
	argc := int(binary.LittleEndian.Uint32(buf[:4]))
	rest := buf[4:]
	end := bytes.IndexByte(rest, 0)
	if end < 0 {
		return nil, unix.EINVAL
	}
	rest = rest[end:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	var args []string
	for len(args) < argc && len(rest) > 0 {
		end = bytes.IndexByte(rest, 0)
		if end < 0 {
			args = append(args, string(rest))
			break
		}
		args = append(args, string(rest[:end]))
		rest = rest[end+1:]
	}
	return args, nil
}

// fillWorkingDirectories reads cwd of the given processes with lsof. A process
// lsof does not report keeps CwdKnown=false.
func fillWorkingDirectories(processes []localProcess, pick func(localProcess) bool) {
	var pids []string
	index := map[int]int{}
	for i, p := range processes {
		if pick(p) {
			pids = append(pids, strconv.Itoa(p.PID))
			index[p.PID] = i
		}
	}
	if len(pids) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, _ := exec.CommandContext(ctx, "/usr/sbin/lsof", "-nP", "-w", "-a", "-d", "cwd", "-Fn", "-p", strings.Join(pids, ",")).Output()
	current := -1
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "p"):
			pid, err := strconv.Atoi(line[1:])
			current = -1
			if i, ok := index[pid]; ok && err == nil {
				current = i
			}
		case strings.HasPrefix(line, "n") && current >= 0:
			processes[current].Cwd, processes[current].CwdKnown = line[1:], true
		}
	}
}
