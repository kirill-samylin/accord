package store

import "agent-board/internal/core"

func validateList(in core.ListInput, kind string) error {
	if in.Role != "" && !core.ValidRole(in.Role) {
		return core.Fail("invalid_input", "unknown role filter")
	}
	if in.State == "" {
		return nil
	}
	allowed := map[string][]string{"task": {"created", "ready", "in_progress", "done", "cancelled"}, "epic": {"open", "done", "cancelled"}, "sessions": {"active", "expired"}}
	for _, state := range allowed[kind] {
		if in.State == state {
			return nil
		}
	}
	return core.Fail("invalid_input", "invalid state filter for "+kind)
}
func validateNotes(in core.NotesInput) error {
	if in.ScopeType != "" && in.ScopeType != "role" && in.ScopeType != "epic" {
		return core.Fail("invalid_input", "scope_type must be role or epic")
	}
	if in.ScopeType == "role" && in.ScopeID != "" && !core.ValidRole(in.ScopeID) {
		return core.Fail("invalid_input", "unknown note role")
	}
	return nil
}
