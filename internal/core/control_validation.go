package core

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// ValidControlSessionID accepts only the server's opaque 256-bit identifiers.
func ValidControlSessionID(id string) bool {
	if len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil && strings.ToLower(id) == id
}
func ValidControlTaskKey(key string) bool { return validControlKey(key, "T-") }

// ValidControlEpicKey accepts canonical positive epic identifiers, not task IDs.
func ValidControlEpicKey(key string) bool { return validControlKey(key, "E-") }

func validControlKey(key, prefix string) bool {
	value, ok := strings.CutPrefix(key, prefix)
	n, err := strconv.ParseInt(value, 10, 64)
	return ok && err == nil && n > 0 && strconv.FormatInt(n, 10) == value
}

// ValidControlEventType is the current recorded vocabulary, not UI telemetry.
func ValidControlEventType(value string) bool {
	switch value {
	case "token_created", "token_revoked", "tokens_revoked_all", "session_opened", "task_created", "task_updated", "claimed", "released", "commented", "linked", "unlinked", "task_ready", "task_in_progress", "task_done", "task_cancelled", "human_answered", "epic_open", "epic_done", "epic_cancelled", "epic_reopened", "lease_expired", "lease_recovered", "attempt_recovered", "database_recovered", "note_created", "note_deleted", "blocked_changed", "profile_created", "profile_updated", "profile_state_changed", "task_assigned", "check_created", "task_duplicate":
		return true
	}
	return false
}
