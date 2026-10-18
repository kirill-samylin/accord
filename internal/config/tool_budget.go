package config

import "slices"

// ToolBudget reports a conservative byte bound, never a model-token measurement.
// Schema bytes are supplied separately by the runtime catalog and never charged
// to (or hidden in) DescriptionBytes.
type ToolBudget struct {
	Role             string `json:"role"`
	DescriptionBytes int    `json:"description_bytes"`
	SchemaBytes      int    `json:"schema_bytes"`
	Limit            int    `json:"configured_budget"`
	Unit             string `json:"unit"`
}

func (s *Snapshot) ToolBudgets(descriptions map[string]string, schemaBytes map[string]int) []ToolBudget {
	if descriptions == nil {
		descriptions = defaultDescriptions
	}
	c := s.Decode()
	roles := make([]string, 0, len(c.Roles))
	for role := range c.Roles {
		roles = append(roles, role)
	}
	slices.Sort(roles)
	out := make([]ToolBudget, 0, len(roles))
	for _, role := range roles {
		b := ToolBudget{Role: role, Limit: c.MCP.ToolBudgetTokens, Unit: "conservative_utf8_bytes"}
		for _, tool := range c.MCP.ToolsByRole[role] {
			b.DescriptionBytes += len(descriptions[tool])
			b.SchemaBytes += schemaBytes[tool]
		}
		out = append(out, b)
	}
	return out
}
