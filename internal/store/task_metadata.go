package store

import (
	"math"
	"strings"
	"unicode/utf8"

	"agent-board/internal/core"
)

func validateMetadata(b *board, t *core.Task) error {
	if len(t.Category) > 128 || !utf8.ValidString(t.Category) || len(t.ExternalID) > 256 || !utf8.ValidString(t.ExternalID) {
		return core.Fail("invalid_input", "category or external_id too long or invalid UTF8")
	}
	if t.Complexity != "" && t.Complexity != "low" && t.Complexity != "normal" && t.Complexity != "high" {
		return core.Fail("invalid_input", "complexity must be low, normal or high")
	}
	if t.ExternalID != "" {
		if strings.TrimSpace(t.ExternalID) == "" {
			return core.Fail("invalid_input", "external_id cannot be whitespace")
		}
		for _, other := range b.Tasks {
			if other.Key != t.Key && other.Type == t.Type && other.ExternalID == t.ExternalID {
				return core.Fail("conflict", "external_id already belongs to "+other.Key)
			}
		}
	}
	return nil
}
func currentAttempt(t *core.Task) *core.Attempt {
	for i := len(t.Attempts) - 1; i >= 0; i-- {
		if t.Attempts[i].EndedAt == nil {
			return &t.Attempts[i]
		}
	}
	return nil
}
func addTokens(t *core.Task, delta *int64, branch string) error {
	if len(branch) > 512 || !utf8.ValidString(branch) {
		return core.Fail("invalid_input", "invalid branch")
	}
	a := currentAttempt(t)
	if a == nil {
		return core.Fail("lease_lost", "active attempt required")
	}
	if delta != nil {
		if *delta < 0 || *delta > math.MaxInt64-a.TokensDelta {
			return core.Fail("invalid_input", "tokens delta must be nonnegative without overflow")
		}
		a.TokensDelta += *delta
		v := a.TokensDelta
		a.TokensSpent = &v
	}
	if branch != "" {
		a.Branch = branch
	}
	return nil
}
func finalTokens(t *core.Task, total *int64) error {
	a := currentAttempt(t)
	if a == nil {
		return core.Fail("lease_lost", "active attempt required")
	}
	if total != nil {
		if *total < 0 || *total < a.TokensDelta {
			return core.Fail("invalid_input", "tokens final total must be >= reported deltas")
		}
		v := *total
		a.TokensSpent = &v
	}
	return nil
}
