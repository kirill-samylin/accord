package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"
)

type questionCursor struct {
	High     int64     `json:"h"`
	Filter   string    `json:"f"`
	Priority int       `json:"p"`
	ReadyAt  time.Time `json:"r"`
	Key      string    `json:"k"`
}

func questionLess(a, b core.Task) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if !a.ReadyAt.Equal(b.ReadyAt) {
		return a.ReadyAt.Before(b.ReadyAt)
	}
	return a.Key < b.Key
}

// ListQuestions preserves the claim queue's priority/ready-time/key ordering.
// Paging fixes the creation boundary; current state/leases can still change.
// Reading this catalogue never acquires or renews a question lease.
func (s *Store) ListQuestions(ctx context.Context, a core.Actor, in core.ListInput) (core.TasksPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.TasksPage, error) {
		page := core.TasksPage{Tasks: []core.Task{}}
		if in.Role != "" && in.Role != "human" {
			return page, core.Fail("VALIDATION", "questions require human role")
		}
		if err := validateList(in, "task"); err != nil {
			return page, err
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		limit := in.Limit
		if limit == 0 {
			limit = 50
		}
		if limit < 1 || limit > 100 {
			return page, core.Fail("VALIDATION", "invalid question page limit")
		}
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		filter.Role = "human"
		raw, err := json.Marshal(filter)
		if err != nil {
			return page, err
		}
		hash := sha256.Sum256(raw)
		c := questionCursor{High: b.Sequence, Filter: hex.EncodeToString(hash[:])}
		if in.Cursor != "" {
			if len(in.Cursor) > 1024 {
				return page, core.Fail("VALIDATION", "invalid question cursor")
			}
			data, err := base64.RawURLEncoding.DecodeString(in.Cursor)
			if err != nil {
				return page, core.Fail("VALIDATION", "invalid question cursor")
			}
			expected := c.Filter
			decoder := json.NewDecoder(strings.NewReader(string(data)))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&c) != nil || decoder.Decode(new(any)) != io.EOF || c.Filter != expected || c.High < 0 || c.High > b.Sequence || numericKey(c.Key) < 1 || numericKey(c.Key) > c.High || c.Priority < 0 || c.Priority > 100 {
				return page, core.Fail("VALIDATION", "question cursor does not match filters")
			}
		}
		tasks := make([]core.Task, 0)
		for _, t := range b.Tasks {
			if t.Type != "task" || t.Role != "human" || numericKey(t.Key) > c.High {
				continue
			}
			if in.State != "" && t.State != in.State || in.State == "" && !in.IncludeClosed && (t.State == "done" || t.State == "cancelled") || in.EpicKey != "" && t.EpicKey != in.EpicKey {
				continue
			}
			owner := ""
			if epic := b.Tasks[t.EpicKey]; epic != nil {
				owner = epic.Owner
			}
			if in.Owner != "" && owner != in.Owner {
				continue
			}
			value := viewTask(b, t)
			if c.Key != "" && !questionLess(core.Task{Key: c.Key, Priority: c.Priority, ReadyAt: c.ReadyAt}, value) {
				continue
			}
			tasks = append(tasks, value)
		}
		sort.Slice(tasks, func(i, j int) bool { return questionLess(tasks[i], tasks[j]) })
		page.HighWater = c.High
		page.HasMore = len(tasks) > limit
		if page.HasMore {
			tasks = tasks[:limit]
		}
		page.Tasks = tasks
		if page.HasMore {
			last := tasks[len(tasks)-1]
			c.Priority = last.Priority
			c.ReadyAt = last.ReadyAt
			c.Key = last.Key
			data, err := json.Marshal(c)
			if err != nil {
				return page, err
			}
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
		}
		return page, nil
	})
}
