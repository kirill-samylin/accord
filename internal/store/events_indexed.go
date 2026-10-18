package store

import (
	"context"
	"encoding/json"
	"sort"

	"agent-board/internal/core"
)

type eventRow struct {
	Cursor int64
	Raw    []byte
}

// indexedEvents bounds every filter stream to limit+1 and keeps only the first
// limit+1 merged rows. A predecessor lookup preserves highest-scanned cursors
// across gaps and unmatched tails without decoding that tail.
func indexedEvents(ctx context.Context, b *board, in core.EventsInput, page core.EventsPage) (core.EventsPage, error) {
	types := []string{""}
	if len(in.Types) > 0 {
		types = nil
		seen := map[string]bool{}
		for _, kind := range in.Types {
			if !seen[kind] {
				seen[kind] = true
				types = append(types, kind)
			}
		}
	}
	matches := []eventRow{}
	for _, kind := range types {
		query := eventStreamQuery(in.TaskKey != "", len(in.Types) > 0)

		args := []any{in.SinceCursor, page.HighWater}
		if in.TaskKey != "" {
			query += ` AND json_extract(data,'$.task_key')=?`
			args = append(args, in.TaskKey)
		}
		if len(in.Types) > 0 {
			query += ` AND json_extract(data,'$.type')=?`
			args = append(args, kind)
		}
		query += ` ORDER BY cursor LIMIT ?`
		args = append(args, in.Limit+1)
		rows, err := b.tx.QueryContext(ctx, query, args...)
		if err != nil {
			return page, err
		}
		for rows.Next() {
			var row eventRow
			if err = rows.Scan(&row.Cursor, &row.Raw); err != nil {
				break
			}
			matches = append(matches, row)
		}
		rowErr := rows.Err()
		rows.Close()
		if err != nil {
			return page, err
		}
		if rowErr != nil {
			return page, rowErr
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].Cursor < matches[j].Cursor })
		if len(matches) > in.Limit+1 {
			matches = matches[:in.Limit+1]
		}
	}
	upper := page.HighWater
	if len(matches) > in.Limit {
		page.HasMore = true
		upper = matches[in.Limit].Cursor - 1
		matches = matches[:in.Limit]
	}
	var last int64
	if err := b.tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(cursor),0) FROM events WHERE cursor>? AND cursor<=?`, in.SinceCursor, upper).Scan(&last); err != nil {
		return page, err
	}
	if last > page.NextCursor {
		page.NextCursor = last
	}
	for _, row := range matches {
		var event core.Event
		if err := json.Unmarshal(row.Raw, &event); err != nil {
			return page, err
		}
		page.Events = append(page.Events, event)
	}
	return page, nil
}

// INDEXED BY is an intentional schema4 plan requirement, not an optimizer
// suggestion. Without it SQLite can pick type-only and scan the whole annual
// type stream while testing task_key in JSON. Missing indexes fail closed.
func eventStreamQuery(task, kind bool) string {
	index := ""
	switch {
	case task && kind:
		index = "events_task_type_cursor"
	case task:
		index = "events_task_cursor"
	case kind:
		index = "events_type_cursor"
	}
	query := "SELECT cursor,data FROM events"
	if index != "" {
		query += " INDEXED BY " + index
	}
	return query + " WHERE cursor>? AND cursor<=?"
}
