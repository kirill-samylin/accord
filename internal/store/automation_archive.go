package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/base64"
	"encoding/json"
	"time"
)

type automationArchiveCursor struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Key       string `json:"key"`
}

// AutomationArchive pages rotated history (newest rows are never here: only
// terminal entries archived by automationRotateHistory). It has the same read
// boundary and projection as the live list readers: no prompts, output or
// profile instructions. Rows are ordered by archive key.
func (s *Store) AutomationArchive(ctx context.Context, a core.Actor, in core.AutomationArchiveInput) (core.AutomationArchivePage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.AutomationArchivePage, error) {
		out := core.AutomationArchivePage{Kind: in.Kind, Workflows: []core.Workflow{}, Runs: []core.ManagedRun{}}
		// Same boundaries as the live readers: WorkflowList for workflows,
		// ManagedList for managed runs.
		if in.Kind == "managed_runs" {
			if err := readActor(ctx, b, a, now); err != nil {
				return out, err
			}
		} else if _, _, err := automationActor(ctx, b, a, now, false); err != nil {
			return out, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return out, err
		}
		if _, ok := ExecutionIdentityFromContext(ctx); ok {
			return out, core.Fail("forbidden", "Дочерняя сессия не читает архив")
		}
		table := map[string]string{"workflows": "workflows", "managed_runs": "managed_runs"}[in.Kind]
		limit := in.Limit
		if limit == 0 {
			limit = 16
		}
		if table == "" || limit < 1 || limit > 32 {
			return out, core.Fail("validation", "Архив: kind workflows|managed_runs, limit 1..32")
		}
		cursor := automationArchiveCursor{Namespace: automationCursorNamespace(b), Kind: in.Kind, Key: automationArchivePrefix}
		if in.Cursor != "" {
			raw, err := base64.RawURLEncoding.DecodeString(in.Cursor)
			var c automationArchiveCursor
			if err != nil || len(raw) > 768 || json.Unmarshal(raw, &c) != nil || c.Namespace != cursor.Namespace || c.Kind != in.Kind {
				return out, core.Fail("validation", "Неверный archive cursor")
			}
			cursor = c
		}
		// table is one of two static names above, never client text.
		rows, err := b.tx.QueryContext(ctx, "SELECT key,data FROM "+table+" WHERE substr(key,1,8)=? AND key>? ORDER BY key LIMIT ?", automationArchivePrefix, cursor.Key, limit+1)
		if err != nil {
			return out, err
		}
		defer rows.Close()
		count, last := 0, ""
		for rows.Next() {
			var key string
			var data []byte
			if err = rows.Scan(&key, &data); err != nil {
				return out, err
			}
			if count == limit {
				cursor.Key = last
				out.NextCursor = encodeAutomationCursor(cursor)
				break
			}
			next := out
			if table == "workflows" {
				var w core.Workflow
				if err = json.Unmarshal(data, &w); err != nil {
					return out, err
				}
				next.Workflows = append(next.Workflows, wireWorkflow(&w, true))
			} else {
				var r core.ManagedRun
				if err = json.Unmarshal(data, &r); err != nil {
					return out, err
				}
				next.Runs = append(next.Runs, archiveRunView(r))
			}
			if automationWireSize(next)+automationCursorReserve > automationWireBudget {
				if count == 0 {
					return out, core.Fail("response_capacity", "Один элемент архива превышает размер страницы")
				}
				cursor.Key = last
				out.NextCursor = encodeAutomationCursor(cursor)
				break
			}
			out, count, last = next, count+1, key
		}
		return out, rows.Err()
	})
}
func archiveRunView(r core.ManagedRun) core.ManagedRun {
	if r.Automation != nil {
		binding := *r.Automation
		binding.Checks = nil
		r.Automation = &binding
	}
	r.Prompt = ""
	r.Output = []string{}
	if r.Profile != nil {
		p := *r.Profile
		p.Instructions = ""
		p.Description = ""
		r.Profile = &p
	}
	return r
}
