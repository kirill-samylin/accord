//go:build darwin

package worker

import "golang.org/x/sys/unix"

func groupMembers(group int) (live int, leader bool, err error) {
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return 0, false, err
	}
	for _, entry := range processes {
		if int(entry.Eproc.Pgid) != group {
			continue
		}
		if int(entry.Proc.P_pid) == group {
			leader = true
		}
		// Darwin sys/proc.h SZOMB=5, SIDL=1; all other visible states
		// may still execute or own a pending syscall and therefore count live.
		if entry.Proc.P_stat != 5 {
			live++
		}
	}
	return live, leader, nil
}
