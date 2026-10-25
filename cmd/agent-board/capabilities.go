package main

import (
	"encoding/json"
	"io"
)

// Operators must reject older binaries before they can open schema6 authority.
// registry_schema 2 and the people_admin floor are permanent after the
// registry migration of docs/admin.md; the guard requires exact equality.
func printCapabilities(out io.Writer) error {
	return json.NewEncoder(out).Encode(map[string]any{
		"project_registry": 1, "project_acl": 1, "registry_schema": []int{2},
		"agent_profiles": 1, "coordination": 1, "deduplication": 1,
		"managed_agents": 1, "automation": 1, "people_admin": 1, "board_schema": []int{6},
	})
}
