//go:build linux

package worker

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func groupMembers(group int) (live int, leader bool, err error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, false, err
	}
	for _, entry := range entries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return 0, false, readErr
		}
		end := strings.LastIndexByte(string(data), ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(data[end+1:]))
		if len(fields) < 3 {
			continue
		}
		pgrp, parseErr := strconv.Atoi(fields[2])
		if parseErr != nil || pgrp != group {
			continue
		}
		if pid == group {
			leader = true
		}
		if fields[0] != "Z" && fields[0] != "X" {
			live++
		}
	}
	return live, leader, nil
}
