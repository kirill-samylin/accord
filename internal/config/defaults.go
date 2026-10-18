package config

import (
	"encoding/json"
	"slices"
)

var defaultDescriptions = map[string]string{
	"comment":    "Record a short report on your current leased attempt.",
	"create":     "Create an epic or a ready task.",
	"describe":   "Open a board session with a fixed role and run ID.",
	"events":     "Read one page; persist next_cursor, never high_water.",
	"get":        "Read an epic or task, including attempts and evidence.",
	"heartbeat":  "Renew only live leases using their exact generations.",
	"link":       "Add a typed dependency; cycles and unsuccessful gates are rejected.",
	"next":       "Claim one eligible task for this session's role.",
	"note":       "Create or soft-delete a bounded role or epic note with provenance.",
	"release":    "Release your live task lease with a reason.",
	"transition": "Change state with current generation and required evidence.",
	"update":     "Edit permitted metadata with ownership checks.",
}

// ToolDescriptions returns an independent copy of the stable catalog. Runtime
// transports should pass their actual embedded descriptions through Options.
func ToolDescriptions() map[string]string {
	out := map[string]string{}
	for k, v := range defaultDescriptions {
		out[k] = v
	}
	return out
}
func toolNames() []string {
	out := make([]string, 0, len(defaultDescriptions))
	for k := range defaultDescriptions {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
func materialize(c *Config) {
	if c.Leases.SessionTTL == "" {
		c.Leases.SessionTTL = c.Leases.RoleTTL
	}
	for name, r := range c.Roles {
		if r.CanCreate == nil {
			if r.Kind == "agent" {
				r.CanCreate = []string{"task"}
			} else {
				r.CanCreate = []string{}
			}
		}
		r.EvidenceOnDone = nonNil(r.EvidenceOnDone)
		r.RecommendedOnDone = nonNil(r.RecommendedOnDone)
		r.IndependentFrom = nonNil(r.IndependentFrom)
		r.Members = nonNil(r.Members)
		if r.Categories == nil {
			r.Categories = map[string]Category{}
		}
		for key, category := range r.Categories {
			if category.ComplexityMax == "" {
				category.ComplexityMax = "high"
			}
			if category.AnswerBy == nil {
				category.AnswerBy = json.RawMessage(`"any"`)
			}
			r.Categories[key] = category
		}
		if r.EvidenceSchema == nil {
			r.EvidenceSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{}}`)
		}
		c.Roles[name] = r
	}
	if len(c.MCP.ToolsByRole) == 0 {
		c.MCP.ToolsByRole = map[string][]string{}
		for name := range c.Roles {
			c.MCP.ToolsByRole[name] = toolNames()
		}
	}
	for i := range c.Areas {
		c.Areas[i].Aliases = nonNil(c.Areas[i].Aliases)
	}
}

var builtinFloors = map[string][]string{
	"developer": {"commit", "run_summary"}, "analyst": {"artifact", "summary"}, "architect": {"artifact", "summary"},
	"human": {"answer"}, "coordinator": {"summary"}, "devops": {"summary"}, "llm-quality": {"artifact", "summary"},
	"design-owner": {"commit", "pr"}, "reviewer": {"verdict", "verified_commit", "findings"},
	"security": {"verdict", "verified_commit", "findings"}, "design-reviewer": {"verdict", "verified_commit", "screens"}, "tester": {"result", "verified_commit", "run_summary"},
}

// RequiredFloor returns the immutable built-in minimum, independent of rule mode.
func RequiredFloor(role string) []string { return append([]string(nil), builtinFloors[role]...) }
