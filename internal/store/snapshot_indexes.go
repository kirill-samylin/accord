package store

import (
	"maps"
	"slices"

	"agent-board/internal/core"
)

// snapshotIndexes contains keys only. Entities remain the JSON source of truth;
// transaction reads merge the small dirty overlay with these committed buckets.
type snapshotIndexes struct {
	Children     map[string][]string
	Reviews      map[string][]string
	Dependents   map[string][]string
	Outgoing     map[string][]string
	RoleTasks    map[string][]string
	Ownership    map[string][]string
	Leased       []string
	Titles       map[string][]string
	Areas        map[string][]string
	AreaChildren map[string][]string
	Order        []string
}

func buildIndexes(b *board) *snapshotIndexes {
	x := &snapshotIndexes{Areas: map[string][]string{}, AreaChildren: map[string][]string{}, Children: map[string][]string{}, Reviews: map[string][]string{}, Dependents: map[string][]string{}, Outgoing: map[string][]string{}, RoleTasks: map[string][]string{}, Ownership: map[string][]string{}, Titles: map[string][]string{}}
	for key, t := range b.Tasks {
		x.add(key, t)
	}
	slices.SortFunc(x.Order, func(a, z string) int { return numericCompare(a, z) })
	return x
}
func numericCompare(a, z string) int {
	if numericKey(a) < numericKey(z) {
		return -1
	}
	if numericKey(a) > numericKey(z) {
		return 1
	}
	return 0
}
func (x *snapshotIndexes) add(key string, t *core.Task) {
	x.Order = append(x.Order, key)
	x.Children[t.EpicKey] = append(x.Children[t.EpicKey], key)
	x.RoleTasks[t.Role] = append(x.RoleTasks[t.Role], key)
	if t.Review != nil {
		x.Reviews[t.Review.TargetKey] = append(x.Reviews[t.Review.TargetKey], key)
	}
	if t.Lease != nil {
		x.Leased = append(x.Leased, key)
		for _, area := range t.Areas {
			x.Areas[area] = append(x.Areas[area], key)
			for _, parent := range areaParents(area) {
				x.AreaChildren[parent] = append(x.AreaChildren[parent], key)
			}
		}
		x.Ownership[t.Lease.SessionID] = append(x.Ownership[t.Lease.SessionID], key)
	}
	for _, link := range t.Links {
		if link.Type == "blocks" || link.Type == "contract_of" {
			x.Dependents[link.From] = append(x.Dependents[link.From], key)
			x.Outgoing[link.From] = append(x.Outgoing[link.From], key)
		}
	}
	if t.Type == "task" && !terminalState(t.State) {
		for gram := range titleGrams(t.Title) {
			x.Titles[gram] = append(x.Titles[gram], key)
		}
	}
}
func (b *board) dirtyTaskKeys() map[string]bool {
	keys := map[string]bool{}
	for part := b; part != nil; part = part.base {
		for key := range part.dirtyTasks {
			keys[key] = true
		}
	}
	return keys
}
func (b *board) indexedTasks(keys []string, predicate func(*core.Task) bool) []*core.Task {
	result := []*core.Task{}
	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
		if t := b.Tasks[key]; t != nil && predicate(t) {
			result = append(result, t)
		}
	}
	for key := range b.dirtyTaskKeys() {
		if seen[key] {
			continue
		}
		if t := b.Tasks[key]; t != nil && predicate(t) {
			result = append(result, t)
		}
	}
	return result
}
func (b *board) children(key string) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	return b.indexedTasks(b.index.Children[key], func(t *core.Task) bool { return t.EpicKey == key })
}
func (b *board) reviews(key string) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	return b.indexedTasks(b.index.Reviews[key], func(t *core.Task) bool { return t.Review != nil && t.Review.TargetKey == key })
}
func (b *board) leasedTasks() []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	return b.indexedTasks(b.index.Leased, func(t *core.Task) bool { return t.Lease != nil })
}
func (b *board) roleTasks(role string) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	return b.indexedTasks(b.index.RoleTasks[role], func(t *core.Task) bool { return t.Role == role })
}

// affected includes dependencies, review assignments/targets and epic closures
// from both old and candidate graphs. Gate invalidations never stop at done.
func (b *board) affected() map[string]bool {
	keys := b.dirtyTaskKeys()
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	if b.base == nil || b.Config.ConfigVersion != b.base.Config.ConfigVersion {
		for key := range b.Tasks {
			keys[key] = true
		}
	}
	queue := slices.Collect(maps.Keys(keys))
	add := func(key string) {
		if key != "" && !keys[key] {
			keys[key] = true
			queue = append(queue, key)
		}
	}
	for i := 0; i < len(queue); i++ {
		key := queue[i]
		if t := b.Tasks[key]; t != nil {
			add(t.EpicKey)
			if t.Review != nil {
				add(t.Review.TargetKey)
			}
		}
		for _, dependent := range b.index.Dependents[key] {
			add(dependent)
		}
		for _, review := range b.reviews(key) {
			add(review.Key)
			add(review.EpicKey)
		}
		for dirty := range b.dirtyTaskKeys() {
			t := b.Tasks[dirty]
			for _, link := range t.Links {
				if link.From == key && (link.Type == "blocks" || link.Type == "contract_of") {
					add(dirty)
				}
			}
		}
	}
	return keys
}

func removeKey(values []string, key string) []string {
	return slices.DeleteFunc(values, func(value string) bool { return value == key })
}
func (x *snapshotIndexes) remove(key string, t *core.Task) {
	x.Order = removeKey(x.Order, key)
	x.Children[t.EpicKey] = removeKey(x.Children[t.EpicKey], key)
	x.RoleTasks[t.Role] = removeKey(x.RoleTasks[t.Role], key)
	if t.Review != nil {
		x.Reviews[t.Review.TargetKey] = removeKey(x.Reviews[t.Review.TargetKey], key)
	}
	if t.Lease != nil {
		x.Leased = removeKey(x.Leased, key)
		for _, area := range t.Areas {
			x.Areas[area] = removeKey(x.Areas[area], key)
			for _, parent := range areaParents(area) {
				x.AreaChildren[parent] = removeKey(x.AreaChildren[parent], key)
			}
		}
		x.Ownership[t.Lease.SessionID] = removeKey(x.Ownership[t.Lease.SessionID], key)
	}
	for _, link := range t.Links {
		if link.Type == "blocks" || link.Type == "contract_of" {
			x.Dependents[link.From] = removeKey(x.Dependents[link.From], key)
			x.Outgoing[link.From] = removeKey(x.Outgoing[link.From], key)
		}
	}
	if t.Type == "task" && !terminalState(t.State) {
		for gram := range titleGrams(t.Title) {
			x.Titles[gram] = removeKey(x.Titles[gram], key)
			if len(x.Titles[gram]) == 0 {
				delete(x.Titles, gram)
			}
		}
	}
}

// commitIndexes runs only after SQLite commit while the process gate is held.
// No caller may retain an internal index beyond that gate. Failed transactions
// never mutate committed index buckets.
func (b *board) commitIndexes() {
	if b.index == nil {
		b.index = buildIndexes(b)
		return
	}
	original := b
	for original.base != nil {
		original = original.base
	}
	for key := range b.dirtyTaskKeys() {
		if old := original.Tasks[key]; old != nil {
			b.index.remove(key, old)
		}
		b.index.add(key, b.Tasks[key])
	}
	slices.SortFunc(b.index.Order, numericCompare)
}
func (b *board) titleCandidates(t *core.Task) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	keys := []string{}
	for gram := range titleGrams(t.Title) {
		keys = append(keys, b.index.Titles[gram]...)
	}
	seen := map[string]bool{}
	unique := keys[:0]
	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			unique = append(unique, key)
		}
	}
	return b.indexedTasks(unique, func(other *core.Task) bool {
		return other.Type == "task" && other.EpicKey == t.EpicKey && !terminalState(other.State)
	})
}

func (b *board) roleIndependentTasks() []*core.Task {
	out := make([]*core.Task, 0, len(b.Tasks))
	for _, t := range b.Tasks {
		out = append(out, t)
	}
	return out
}
