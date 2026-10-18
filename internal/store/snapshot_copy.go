package store

import (
	"encoding/json"
	"maps"

	"agent-board/internal/core"
)

// detach copies the complete JSON graph, including RawMessage, evidence and
// nested lease/attempt pointers. Errors are never converted into shallow copies.
func detach[T any](value T) (T, error) {
	var out T
	raw, err := json.Marshal(value)
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}

// fork shares immutable entities; every entity write must pass an edit barrier.
// Maps are private so inserts cannot affect a committed or expiry snapshot.
func (b *board) fork() *board {
	c := *b
	c.Tasks = maps.Clone(b.Tasks)
	c.Sessions = maps.Clone(b.Sessions)
	c.Notes = maps.Clone(b.Notes)
	c.Roles = maps.Clone(b.Roles)
	c.Events = append([]core.Event(nil), b.Events...)
	c.base = b
	c.dirtyTasks = map[string]bool{}
	c.dirtySessions = map[string]bool{}
	c.dirtyNotes = map[string]bool{}
	c.gateMemo = map[string]bool{}
	return &c
}
func editEntity[T any](b *board, entities map[string]*T, dirty map[string]bool, key string) *T {
	value := entities[key]
	if value == nil || dirty[key] {
		return value
	}
	copy, err := detach(value)
	if err != nil {
		b.copyErr = err
		// A failed clone must never give the caller the shared pointer.
		copy = new(T)
	}
	entities[key] = copy
	dirty[key] = true
	b.gateMemo = map[string]bool{}
	return copy
}
func (b *board) editTask(key string) *core.Task {
	return editEntity(b, b.Tasks, b.dirtyTasks, key)
}
func (b *board) editSession(key string) *core.Session {
	return editEntity(b, b.Sessions, b.dirtySessions, key)
}
func (b *board) editNote(key string) *core.Note {
	return editEntity(b, b.Notes, b.dirtyNotes, key)
}
func (b *board) putTask(t *core.Task) {
	b.Tasks[t.Key] = t
	b.dirtyTasks[t.Key] = true
	b.gateMemo = map[string]bool{}
}
