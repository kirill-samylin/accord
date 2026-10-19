package store

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"agent-board/internal/core"
)

// ControlTasks searches the full project before applying native list pagination.
func (s *Store) ControlTasks(ctx context.Context, a core.Actor, in core.ControlTasksInput) (core.TasksPage, error) {
	in.Query = strings.ToLower(strings.TrimSpace(in.Query))
	if !utf8.ValidString(in.Query) || utf8.RuneCountInString(in.Query) > 200 || len(in.Query) > 1024 {
		return core.TasksPage{}, core.Fail("validation", "Поиск ограничен 200 символами и 1024 байтами")
	}
	return transact(s, ctx, func(b *board, now time.Time) (core.TasksPage, error) {
		if err := validateList(in.ListInput, "task"); err != nil {
			return core.TasksPage{}, err
		}
		if in.Limit < 0 || in.Limit > 100 {
			return core.TasksPage{}, core.Fail("validation", "limit must be 1..100")
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return core.TasksPage{}, err
		}
		filter := in
		filter.Cursor = ""
		filter.Limit = 0
		return taskListPage(b, in.ListInput, "task", "control_tasks", filter, in.Query)
	})
}
