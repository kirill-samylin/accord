//go:build darwin || linux

package worker

import (
	"syscall"
	"testing"
)

func TestRecoveryCensusSeesLiveGroupsOnlyAndNeverSignals(t *testing.T) {
	own := syscall.Getpgrp()
	alive, err := processGroupsAlive([]int{own, 1, 0, 2147483000})
	if err != nil || len(alive) != 1 || alive[0] != own {
		t.Fatal("census missed this live group or reported an absent one", alive, err)
	}
}
