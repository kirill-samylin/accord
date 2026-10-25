package store

import (
	"fmt"
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

// maxAttemptTokens bounds one reported token amount (a delta, or the growth a
// final total adds). It is far above any real run; statistics saturate sums.
// Only the newly reported amount is bounded, so a legacy attempt whose stored
// total already exceeds it can still report other fields and zero deltas.
const maxAttemptTokens int64 = 1_000_000_000_000

// Areas are scope labels or relative paths used for overlap warnings; they are
// bounded like plan areas and never carry traversal or control characters.
const (
	maxAreas     = 32
	maxAreaRunes = 128
	maxOwner     = 128
)

// maxPlanAreaBytes keeps the analyst plan contract that predates task areas,
// so previously accepted plans still validate.
const maxPlanAreaBytes = 1024

// validPlanAreas applies the printable/UTF-8/no-repeat rules to analyst plan
// areas with the original per-area bound of 1..1024 bytes.
func validPlanAreas(areas []string) error {
	if len(areas) > maxAreas {
		return core.Fail("validation", fmt.Sprintf("areas accepts at most %d entries", maxAreas))
	}
	seen := make(map[string]bool, len(areas))
	for _, area := range areas {
		if !bounded(area, maxPlanAreaBytes) || !plainText(area, false) || seen[area] {
			return core.Fail("validation", fmt.Sprintf("Неверные области задачи плана: до %d значений по 1..%d байт, без управляющих символов и повторов", maxAreas, maxPlanAreaBytes))
		}
		seen[area] = true
	}
	return nil
}

func validAreas(areas []string) error {
	if len(areas) > maxAreas {
		return core.Fail("invalid_input", fmt.Sprintf("areas accepts at most %d entries", maxAreas))
	}
	seen := make(map[string]bool, len(areas))
	for _, area := range areas {
		if strings.TrimSpace(area) == "" || utf8.RuneCountInString(area) > maxAreaRunes || !plainText(area, false) {
			return core.Fail("invalid_input", fmt.Sprintf("each area must contain 1..%d characters without control characters", maxAreaRunes))
		}
		for _, segment := range strings.Split(area, "/") {
			if segment == "." || segment == ".." {
				return core.Fail("invalid_input", "area must not contain . or .. path segments")
			}
		}
		if seen[area] {
			return core.Fail("invalid_input", "areas must not repeat: "+area)
		}
		seen[area] = true
	}
	return nil
}

// Only epics have an owner; a task's participant comes from its lease, so an
// owner on task create is rejected rather than silently dropped.
func validOwner(kind, owner string) error {
	if owner == "" {
		return nil
	}
	if kind != "epic" {
		return core.Fail("invalid_input", "owner applies only to epics; a task's participant is the session holding its lease")
	}
	if strings.TrimSpace(owner) == "" || utf8.RuneCountInString(owner) > maxOwner || !plainText(owner, false) {
		return core.Fail("invalid_input", fmt.Sprintf("owner must contain 1..%d characters without control characters", maxOwner))
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
		if *delta < 0 || *delta > maxAttemptTokens || *delta > math.MaxInt64-a.TokensDelta {
			return core.Fail("invalid_input", fmt.Sprintf("tokens_spent_delta must be 0..%d without overflowing the attempt total", maxAttemptTokens))
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
		if *total-a.TokensDelta > maxAttemptTokens {
			return core.Fail("invalid_input", fmt.Sprintf("tokens_spent may exceed the reported deltas by at most %d", maxAttemptTokens))
		}
		v := *total
		a.TokensSpent = &v
	}
	return nil
}
