package store

import (
	"agent-board/internal/core"
	"golang.org/x/text/cases"
	"sort"
	"strings"
)

func titleGrams(title string) map[string]bool {
	normalized := cases.Fold().String(strings.Join(strings.Fields(title), " "))
	chars := []rune(normalized)
	out := map[string]bool{}
	if len(chars) < 3 {
		out[normalized] = true
		return out
	}
	for i := 0; i+3 <= len(chars); i++ {
		out[string(chars[i:i+3])] = true
	}
	return out
}
func titleDice(a, b string) float64 {
	left, right := titleGrams(a), titleGrams(b)
	intersection := 0
	for gram := range left {
		if right[gram] {
			intersection++
		}
	}
	return 2 * float64(intersection) / float64(len(left)+len(right))
}
func checkDuplicate(b *board, t *core.Task, sess *core.Session, force bool, reason string) error {
	mode := b.Config.Workflow.Rules.DuplicateCheck.Mode
	if mode == "off" || t.Type != "task" || terminalState(t.State) {
		return nil
	}
	candidates := b.titleCandidates(t)
	if b.Config.Workflow.Rules.DuplicateCheck.Threshold == 0 {
		candidates = b.roleIndependentTasks()
	}
	for _, other := range candidates {
		if other.Key == t.Key || other.Type != "task" || other.EpicKey != t.EpicKey || terminalState(other.State) {
			continue
		}
		score := titleDice(t.Title, other.Title)
		if score >= b.Config.Workflow.Rules.DuplicateCheck.Threshold {
			t.Similar = append(t.Similar, core.SimilarTask{Key: other.Key, Score: score})
		}
	}
	sort.Slice(t.Similar, func(i, j int) bool { return numericKey(t.Similar[i].Key) < numericKey(t.Similar[j].Key) })
	if len(t.Similar) == 0 {
		return nil
	}
	if mode == "block" {
		if !force || sess.Role != "coordinator" || !bounded(reason, 4096) {
			return core.Fail("DUPLICATE", "similar task "+t.Similar[0].Key+"; coordinator force with reason required")
		}
		ruleWarning(b, t, "duplicate_override", reason, sess.SessionID)
		return nil
	}
	ruleWarning(b, t, "duplicate_check", "similar nonterminal tasks exist", sess.SessionID)
	return nil
}
