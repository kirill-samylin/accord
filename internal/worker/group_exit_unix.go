//go:build darwin || linux

package worker

import "time"

// The WNOWAIT leader pins the group identity while this checks that no member
// can execute. Zombies have exited and cannot write the checkout.
func waitGroupExit(group int) error {
	deadline := time.Now().Add(3 * time.Second)
	for {
		live, leader, err := groupMembers(group)
		if err != nil {
			return ownershipSystemFailure("unix_exit_census", err)
		}
		if !leader {
			return ownershipFailureAt("unix_exit_census_leader_missing")
		}
		if live == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return ownershipFailureAt("unix_group_exit_timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
