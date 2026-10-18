package store

import (
	"strings"

	"agent-board/internal/core"
)

func areaParents(path string) []string {
	out := []string{path}
	for strings.Contains(path, "/") {
		index := strings.LastIndexByte(path, '/')
		path = path[:index]
		out = append(out, path)
	}
	return out
}
func (b *board) ownedTasks(session string) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	return b.indexedTasks(b.index.Ownership[session], func(t *core.Task) bool { return t.Lease != nil && t.Lease.SessionID == session })
}
func (b *board) areaCandidates(t *core.Task) []*core.Task {
	if b.index == nil {
		b.index = buildIndexes(b)
	}
	keys := []string{}
	seen := map[string]bool{}
	add := func(values []string) {
		for _, key := range values {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	for _, area := range t.Areas {
		add(b.index.AreaChildren[area])
		for _, parent := range areaParents(area) {
			add(b.index.Areas[parent])
		}
	}
	return b.indexedTasks(keys, func(other *core.Task) bool { return other.Lease != nil })
}
