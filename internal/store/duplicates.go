package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"agent-board/internal/config"
	"agent-board/internal/core"
	"agent-board/internal/similar"
)

// recentFinish keeps finished work comparable: re-creating something finished
// last week is as wasteful as duplicating open work.
const recentFinish = 14 * 24 * time.Hour

// maxCompared bounds pairwise detection; the newest items are kept.
const maxCompared = 3000

type dupCandidate struct {
	t   *core.Task
	doc similar.Doc
}

func duplicateMode(b *board) (string, float64) {
	mode, threshold := b.Config.DuplicateMode, b.Config.DuplicateThreshold
	if mode == "" {
		mode = core.DuplicateWarn
	}
	if threshold == 0 {
		threshold = 0.6
	}
	return mode, threshold
}

// dupKind separates questions, work and epics. Generated review/QA checks
// intentionally resemble each other and are never compared.
func dupKind(t *core.Task) string {
	switch {
	case t.Type == "epic":
		return "epic"
	case t.Review != nil || t.Role == "reviewer" || t.Role == "tester":
		return ""
	case t.Role == "human":
		return "question"
	}
	return "work"
}
func kindOf(typ, role string) string {
	return dupKind(&core.Task{Type: typ, Role: role})
}

func finishedRecently(t *core.Task, now time.Time) bool {
	if t.State != "done" {
		return false
	}
	end := t.FirstDoneAt
	if a := currentAttempt(t); a != nil && a.EndedAt != nil {
		end = a.EndedAt
	}
	return end != nil && now.Sub(*end) < recentFinish
}
func comparable(t *core.Task, now time.Time) bool {
	if dupKind(t) == "" {
		return false
	}
	switch t.State {
	case "created", "ready", "in_progress", "open":
		return true
	}
	return finishedRecently(t, now)
}

// linkedPair is true for any explicit link, including "related" (dismissed).
func linkedPair(a, b *core.Task) bool {
	for _, side := range [][2]*core.Task{{a, b}, {b, a}} {
		for _, l := range side[0].Links {
			if l.From == side[1].Key || l.To == side[1].Key {
				return true
			}
		}
	}
	return false
}

func sharedAreas(a, b []string) []string {
	var out []string
	for _, x := range a {
		for _, y := range b {
			if config.AreasIntersect(x, y) {
				out = append(out, x)
				break
			}
		}
	}
	return out
}

func summarize(b *board, t *core.Task, now time.Time, score float64, reasons, areas []string) core.SimilarTask {
	v := core.SimilarTask{Key: t.Key, Type: t.Type, Title: t.Title, State: t.State, Role: t.Role, EpicKey: t.EpicKey, Score: score, Reasons: reasons, Assignment: t.Assignment, CreatedAt: t.CreatedAt, SharedAreas: areas}
	people := core.Task{Requester: t.Requester, CreatedBy: t.CreatedBy, Lease: t.Lease}
	projectPeople(b, &people)
	v.CreatedBy, v.HolderOwner = people.CreatedBy, people.HolderOwner
	v.LiveLease = t.Lease != nil && t.Lease.ExpiresAt.After(now)
	v.FinishedRecent = finishedRecently(t, now)
	if v.Reasons == nil {
		v.Reasons = []string{}
	}
	return v
}

// similarTo compares one proposed item against existing comparable work.
func similarTo(b *board, now time.Time, in core.SimilarInput, min float64, limit int) []core.SimilarTask {
	return similarCandidates(b, now, in, min, limit, false)
}

func similarCandidates(b *board, now time.Time, in core.SimilarInput, min float64, limit int, liveOnly bool) []core.SimilarTask {
	kind := kindOf(in.Type, in.Role)
	out := []core.SimilarTask{}
	if kind == "" {
		return out
	}
	doc := similar.New(in.Title, in.Description, in.ExternalID)
	exclude := b.Tasks[in.ExcludeKey]
	for _, t := range b.Tasks {
		if t.Key == in.ExcludeKey || dupKind(t) != kind || !comparable(t, now) || exclude != nil && linkedPair(exclude, t) {
			continue
		}
		// Claim warnings select live work before ranking/truncation, otherwise
		// higher-ranked unclaimed items can hide another holder's actual work.
		if liveOnly && (t.Lease == nil || !t.Lease.ExpiresAt.After(now)) {
			continue
		}
		score, reasons := similar.Score(doc, similar.New(t.Title, t.Description, t.ExternalID))
		areas := sharedAreas(in.Areas, t.Areas)
		if len(areas) > 0 && score >= min-0.1 {
			score = minFloat(1, score+0.1)
			reasons = append(reasons, "areas")
		}
		if score >= min {
			out = append(out, summarize(b, t, now, score, reasons, areas))
		}
	}
	sortSimilar(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func sortSimilar(out []core.SimilarTask) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return keyOrder(out[i].Key) < keyOrder(out[j].Key)
	})
}
func keyOrder(key string) int64 {
	var n int64
	_, _ = fmt.Sscanf(key[strings.IndexByte(key, '-')+1:], "%d", &n)
	return n
}
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func similarKeys(items []core.SimilarTask) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

// duplicateCheck runs inside the create transaction, before the new task is
// stored. Strict mode rejects; warn mode returns advisory warnings.
func duplicateCheck(b *board, in core.CreateInput, t *core.Task, now time.Time) ([]core.Warning, error) {
	if in.Force && (!bounded(in.ForceReason, 1024) || !utf8.ValidString(in.ForceReason)) {
		return nil, core.Fail("invalid_input", "force requires force_reason (≤1 KiB)")
	}
	if !in.Force && in.ForceReason != "" {
		return nil, core.Fail("invalid_input", "force_reason requires force")
	}
	mode, threshold := duplicateMode(b)
	if mode == core.DuplicateOff || t.Review != nil {
		return nil, nil
	}
	found := similarTo(b, now, core.SimilarInput{Type: t.Type, Role: t.Role, Title: t.Title, Description: t.Description, ExternalID: t.ExternalID, Areas: t.Areas}, threshold, 5)
	if len(found) == 0 {
		return nil, nil
	}
	keys := similarKeys(found)
	if in.Force {
		t.DuplicateOverride = &core.DuplicateOverride{Reason: in.ForceReason, Similar: keys}
	} else if mode == core.DuplicateStrict {
		return nil, core.Fail("duplicate_suspected", duplicateMessage(found)+". Если это другая работа, повторите с force=true и force_reason")
	}
	return []core.Warning{{Rule: "duplicate_check", Message: duplicateMessage(found) + ". Проверьте их до начала работы; лишнюю задачу отметьте связью duplicate_of", Similar: found}}, nil
}

func duplicateMessage(found []core.SimilarTask) string {
	parts := make([]string, 0, len(found))
	for _, f := range found {
		who := f.CreatedBy
		if f.HolderOwner != "" {
			who = "в работе у " + f.HolderOwner
		}
		part := fmt.Sprintf("%s «%s» (%.0f%%", f.Key, f.Title, f.Score*100)
		if who != "" {
			part += ", " + who
		}
		parts = append(parts, part+")")
	}
	return "Похожая работа уже есть: " + strings.Join(parts, "; ")
}

func (s *Store) Similar(ctx context.Context, a core.Actor, in core.SimilarInput) (core.SimilarPage, error) {
	if !validTitle(in.Title) || len(in.Description) > 16384 || in.Type != "" && in.Type != "task" && in.Type != "epic" || in.Role != "" && !core.ValidRole(in.Role) || in.Limit < 0 || in.Limit > 20 || len(in.Areas) > 32 {
		return core.SimilarPage{}, core.Fail("validation", "Неверный запрос похожих задач")
	}
	if in.Type == "" {
		in.Type = "task"
	}
	if in.Limit == 0 {
		in.Limit = 8
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.SimilarPage, error) {
		mode, threshold := duplicateMode(b)
		page := core.SimilarPage{ObservedAt: now, Mode: mode, Threshold: threshold}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		// Suggestions while typing use a slightly lower bar than enforcement.
		page.Similar = similarTo(b, now, in, minFloat(threshold, 0.45), in.Limit)
		return page, nil
	})
}

// claimWarnings tells a new holder about live work by others that overlaps.
func claimWarnings(b *board, t *core.Task, now time.Time) []core.Warning {
	var out []core.Warning
	mode, threshold := duplicateMode(b)
	if mode != core.DuplicateOff {
		same := similarCandidates(b, now, core.SimilarInput{Type: t.Type, Role: t.Role, Title: t.Title, Description: t.Description, ExternalID: t.ExternalID, ExcludeKey: t.Key}, threshold, 5, true)
		if len(same) > 0 {
			out = append(out, core.Warning{Rule: "duplicate_check", Message: "Похожую задачу уже выполняют: " + strings.Join(similarKeys(same), ", ") + ". Сверьтесь до правок", Similar: same})
		}
	}
	var overlap []core.SimilarTask
	for _, other := range b.Tasks {
		if other.Key == t.Key || other.Lease == nil || !other.Lease.ExpiresAt.After(now) {
			continue
		}
		if areas := sharedAreas(t.Areas, other.Areas); len(areas) > 0 {
			overlap = append(overlap, summarize(b, other, now, 0, []string{"areas"}, areas))
		}
	}
	if len(overlap) > 0 {
		sortSimilar(overlap)
		if len(overlap) > 5 {
			overlap = overlap[:5]
		}
		out = append(out, core.Warning{Rule: "area_overlap", Message: "В тех же областях сейчас работают: " + strings.Join(similarKeys(overlap), ", ") + ". Области — метки, не блокировка файлов", Similar: overlap})
	}
	return out
}
