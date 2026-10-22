package store

import (
	"context"
	"sort"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/similar"
)

// projectPeople fills response-only owner labels. Legacy tasks without
// CreatedBy resolve the requester principal through recorded sessions.
func projectPeople(b *board, v *core.Task) {
	if v.Lease != nil {
		if sess := b.Sessions[v.Lease.SessionID]; sess != nil {
			v.HolderOwner, v.HolderMachine = sess.Owner, sess.Machine
		}
	}
	if v.CreatedBy == "" && v.Requester != "" {
		v.CreatedBy = principalOwner(b, v.Requester)
	}
	for _, l := range v.Links {
		if l.Type == "duplicate_of" && l.From == v.Key {
			v.DuplicateOf = l.To
		}
	}
}

func principalOwner(b *board, principal string) string {
	if b.owners == nil || b.ownersFor != len(b.Sessions) {
		b.owners = make(map[string]string, len(b.Sessions))
		for _, sess := range b.Sessions {
			if sess.Owner != "" {
				b.owners[sess.PrincipalID] = sess.Owner
			}
		}
		b.ownersFor = len(b.Sessions)
	}
	return b.owners[principal]
}

// markDuplicate cancels unleased duplicate work and records the canonical
// item. A live lease is never taken over: its holder must release first.
func (s *Store) markDuplicate(b *board, sess *core.Session, a core.Actor, key, canonicalKey, reason string, version int64, now time.Time) (core.Task, error) {
	dup, err := taskFor(b, key)
	if err != nil {
		return core.Task{}, err
	}
	canonical, err := taskFor(b, canonicalKey)
	if err != nil {
		return core.Task{}, err
	}
	if dup.Key == canonical.Key || dup.Type != canonical.Type {
		return core.Task{}, core.Fail("invalid_input", "Дубль и основная задача должны быть разными элементами одного типа")
	}
	if len(reason) > 4096 {
		return core.Task{}, core.Fail("invalid_input", "Причина длиннее 4 КиБ")
	}
	if version != 0 && version != dup.Version {
		return core.Task{}, core.Fail("conflict", "Задача изменилась; обновите страницу")
	}
	if canonical.State == "cancelled" {
		return core.Task{}, core.Fail("invalid_input", "Основная задача отменена; выберите действующую")
	}
	for at, seen := canonical, map[string]bool{}; at != nil && !seen[at.Key]; at = b.Tasks[duplicateTarget(at)] {
		seen[at.Key] = true
		if at.Key == dup.Key {
			return core.Task{}, core.Fail("cycle", "Основная задача сама отмечена дублем этой")
		}
	}
	if dup.Lease != nil && dup.Lease.ExpiresAt.After(now) {
		holder := ""
		if h := b.Sessions[dup.Lease.SessionID]; h != nil {
			holder = " у " + h.Owner
		}
		return core.Task{}, core.Fail("lease_held", "Задача в работе"+holder+"; исполнитель должен освободить её")
	}
	if dup.Type == "epic" && dup.State != "open" || dup.Type == "task" && dup.State != "ready" && dup.State != "created" {
		return core.Task{}, core.Fail("invalid_transition", "Отметить дублем можно только незавершённую задачу без исполнителя")
	}
	if !administrative(sess, dup, a) {
		return core.Task{}, core.Fail("forbidden", "Отметить дублем может координатор, человек или автор задачи")
	}
	if dup.Type == "epic" {
		for _, child := range b.Tasks {
			if child.EpicKey == dup.Key && child.State != "done" && child.State != "cancelled" {
				return core.Task{}, core.Fail("rule_violation", "В эпике есть незавершённые задачи; перенесите или отмените их")
			}
		}
	}
	old := dup.State
	dup.State = "cancelled"
	dup.Links = append(dup.Links, core.Link{From: dup.Key, To: canonical.Key, Type: "duplicate_of"})
	dup.Version++
	if dup.Type == "task" {
		reopenEpic(b, dup.EpicKey, now, a.SessionID)
	}
	emit(b, now, "task_duplicate", dup.Key, a.SessionID)
	b.Events[len(b.Events)-1].Data = &core.EventData{DuplicateOf: canonical.Key, DuplicateReason: reason, OldState: old, NewState: "cancelled", Owner: sess.Owner}
	return viewTask(b, dup), nil
}

func duplicateTarget(t *core.Task) string {
	for _, l := range t.Links {
		if l.Type == "duplicate_of" && l.From == t.Key {
			return l.To
		}
	}
	return ""
}

// Duplicate is the explicit REST/CLI/browser operation; MCP agents use
// link{type:"duplicate_of"} to stay within the fixed tool budget.
func (s *Store) Duplicate(ctx context.Context, a core.Actor, in core.DuplicateInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "duplicate", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		return s.markDuplicate(b, sess, a, in.TaskKey, in.CanonicalKey, in.Reason, in.Version, now)
	})
}

// duplicatePairs finds candidate pairs among comparable work using a stem
// index, so unrelated items are never scored against each other.
func duplicatePairs(b *board, now time.Time, threshold float64) []core.DuplicatePair {
	items := make([]dupCandidate, 0)
	for _, t := range b.Tasks {
		if comparable(t, now) {
			items = append(items, dupCandidate{t: t})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return keyOrder(items[i].t.Key) > keyOrder(items[j].t.Key) || keyOrder(items[i].t.Key) == keyOrder(items[j].t.Key) && items[i].t.Key < items[j].t.Key
	})
	if len(items) > maxCompared {
		items = items[:maxCompared]
	}
	index := map[string][]int{}
	for i := range items {
		items[i].doc = similar.New(items[i].t.Title, items[i].t.Description, items[i].t.ExternalID)
		keys := items[i].doc.Stems
		if items[i].doc.External != "" {
			keys = append(append([]string{}, keys...), "\x00"+items[i].doc.External)
		}
		for _, stem := range keys {
			index[stem] = append(index[stem], i)
		}
	}
	seen := map[[2]int]bool{}
	out := []core.DuplicatePair{}
	for _, group := range index {
		// Very common stems carry no signal; real duplicates share rarer ones.
		if len(group) > 400 {
			continue
		}
		for x := 0; x < len(group); x++ {
			for y := x + 1; y < len(group); y++ {
				i, j := group[x], group[y]
				if seen[[2]int{i, j}] {
					continue
				}
				seen[[2]int{i, j}] = true
				a, c := items[i].t, items[j].t
				if dupKind(a) != dupKind(c) || a.State == "done" && c.State == "done" || linkedPair(a, c) {
					continue
				}
				score, reasons := similar.Score(items[i].doc, items[j].doc)
				areas := sharedAreas(a.Areas, c.Areas)
				if len(areas) > 0 && score >= threshold-0.1 {
					score, reasons = minFloat(1, score+0.1), append(reasons, "areas")
				}
				if score < threshold {
					continue
				}
				older, newer := a, c
				if keyOrder(c.Key) < keyOrder(a.Key) {
					older, newer = c, a
				}
				out = append(out, core.DuplicatePair{Score: score, Reasons: reasons, Older: summarize(b, older, now, score, nil, areas), Newer: summarize(b, newer, now, score, nil, areas)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return keyOrder(out[i].Newer.Key) > keyOrder(out[j].Newer.Key)
	})
	return out
}
