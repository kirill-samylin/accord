package main

import (
	"encoding/json"
	"io"
)

// Operators must reject older binaries before they can open schema6 authority.
func printCapabilities(out io.Writer) error {
	return json.NewEncoder(out).Encode(map[string]any{
		"project_registry": 1, "project_acl": 1, "registry_schema": []int{1},
		"agent_profiles": 1, "coordination": 1, "deduplication": 1,
		"managed_agents": 1, "automation": 1, "board_schema": []int{6},
	})
}
