package store

import (
	"context"
	"sort"
	"strings"
	"time"

	"agent-board/internal/core"
)

func readActor(ctx context.Context, b *board, a core.Actor, now time.Time) error {
	if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
		return err
	}
	_, err := sessionFor(b, a, now)
	return err
}
func (s *Store) ListTasks(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return s.listTasks(ctx, a, in, "task")
}
func (s *Store) ListEpics(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return s.listTasks(ctx, a, in, "epic")
}
func (s *Store) listTasks(ctx context.Context, a core.Actor, in core.ListInput, kind string) (core.TasksPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.TasksPage, error) {
		page := core.TasksPage{Tasks: []core.Task{}}
		if err := validateList(in, kind); err != nil {
			return page, err
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		return taskListPage(b, in, kind, kind, filter, "")
	})
}

// Sessions is the authenticated paginated API; ListSessions remains the local
// administrator command and deliberately has no bearer/session parameters.
func (s *Store) Sessions(ctx context.Context, a core.Actor, in core.ListInput) (core.SessionsPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.SessionsPage, error) {
		page := core.SessionsPage{Sessions: []core.Session{}}
		if err := validateList(in, "sessions"); err != nil {
			return page, err
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		sessions := make([]core.Session, 0, len(b.Sessions))
		// Schema1 had no ordering field. Opening events provide its real creation
		// order without inventing timestamps or changing historical responses.
		orders, err := sessionOrders(ctx, b)
		if err != nil {
			return page, err
		}
		var high int64
		for _, sess := range b.Sessions {
			v := *sess
			if v.Order == 0 {
				v.Order = orders[v.SessionID]
			}
			if v.Order > high {
				high = v.Order
			}
			if v.Owner == "" {
				if err := b.tx.QueryRowContext(ctx, `SELECT owner FROM tokens WHERE id=?`, v.PrincipalID).Scan(&v.Owner); err != nil {
					return page, err
				}
			}
			sessions = append(sessions, v)
		}
		sort.Slice(sessions, func(i, j int) bool { return sessions[i].Order < sessions[j].Order })
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		c, limit, err := pageCursor(cursorKindForBoard(b, "sessions"), filter, in.Cursor, in.Limit, high)
		if err != nil {
			return page, err
		}
		last := c.After
		more := false
		for _, sess := range sessions {
			if sess.Order <= c.After || sess.Order > c.High {
				continue
			}
			live := sess.ExpiresAt.After(now)
			if in.State == "active" && !live || in.State == "expired" && live || in.State == "" && !in.IncludeClosed && !live {
				continue
			}
			if in.Role != "" && sess.Role != in.Role || in.Owner != "" && sess.Owner != in.Owner {
				continue
			}
			if in.EpicKey != "" {
				found := false
				for _, t := range b.Tasks {
					if t.EpicKey == in.EpicKey && t.Lease != nil && t.Lease.SessionID == sess.SessionID {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			if len(page.Sessions) == limit {
				more = true
				break
			}
			page.Sessions = append(page.Sessions, sess)
			last = sess.Order
		}
		page.PageInfo = finishPage(c, last, more)
		return page, nil
	})
}
func (s *Store) ListNotes(ctx context.Context, a core.Actor, in core.NotesInput) (core.NotesPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.NotesPage, error) {
		page := core.NotesPage{Notes: []core.Note{}}
		if err := validateNotes(in); err != nil {
			return page, err
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		c, limit, err := pageCursor(cursorKindForBoard(b, "notes"), filter, in.Cursor, in.Limit, b.NoteSequence)
		if err != nil {
			return page, err
		}
		notes := make([]core.Note, 0, len(b.Notes))
		for _, n := range b.Notes {
			notes = append(notes, *n)
		}
		sort.Slice(notes, func(i, j int) bool { return numericKey(notes[i].Key) < numericKey(notes[j].Key) })
		last := c.After
		more := false
		for _, n := range notes {
			id := numericKey(n.Key)
			if id <= c.After || id > c.High || n.DeletedAt != nil && !in.IncludeDeleted || in.ScopeType != "" && n.ScopeType != in.ScopeType || in.ScopeID != "" && n.ScopeID != in.ScopeID {
				continue
			}
			if len(page.Notes) == limit {
				more = true
				break
			}
			page.Notes = append(page.Notes, n)
			last = id
		}
		page.PageInfo = finishPage(c, last, more)
		return page, nil
	})
}

// taskListPage shares native filters with whole-project browser search.
func taskListPage(b *board, in core.ListInput, kind, cursorKind string, filter any, query string) (core.TasksPage, error) {
	page := core.TasksPage{Tasks: []core.Task{}}
	c, limit, err := pageCursor(cursorKindForBoard(b, cursorKind), filter, in.Cursor, in.Limit, b.Sequence)
	if err != nil {
		return page, err
	}
	keys := make([]string, 0, len(b.Tasks))
	for key := range b.Tasks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return numericKey(keys[i]) < numericKey(keys[j]) })
	last := c.After
	more := false
	for _, key := range keys {
		id := numericKey(key)
		t := b.Tasks[key]
		if id <= c.After || id > c.High || t.Type != kind {
			continue
		}
		if in.State != "" && t.State != in.State || in.Role != "" && t.Role != in.Role || in.EpicKey != "" && t.EpicKey != in.EpicKey {
			continue
		}
		if in.State == "" && !in.IncludeClosed && (t.State == "done" || t.State == "cancelled") {
			continue
		}
		owner := t.Owner
		if t.Type == "task" {
			if e := b.Tasks[t.EpicKey]; e != nil {
				owner = e.Owner
			}
		}
		if in.Owner != "" && owner != in.Owner {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(t.Key), query) && !strings.Contains(strings.ToLower(t.Title), query) && !strings.Contains(strings.ToLower(t.Description), query) {
			continue
		}
		if len(page.Tasks) == limit {
			more = true
			break
		}
		page.Tasks = append(page.Tasks, viewTask(b, t))
		last = id
	}
	page.PageInfo = finishPage(c, last, more)
	return page, nil
}
