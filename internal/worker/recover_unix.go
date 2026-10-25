//go:build darwin || linux

package worker

// processGroupsAlive is a read-only census. A reused group ID only makes
// recovery refuse; no signal is ever sent from this path.
func processGroupsAlive(groups []int) ([]int, error) {
	var alive []int
	for _, group := range groups {
		if group <= 1 {
			continue
		}
		live, _, err := groupMembers(group)
		if err != nil {
			return nil, err
		}
		if live > 0 {
			alive = append(alive, group)
		}
	}
	return alive, nil
}
