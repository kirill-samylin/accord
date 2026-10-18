package store

import (
	"testing"
)

// dropPerformanceFixture models the actual prior schema. Leaving schema4 DDL
// behind and changing only user_version would be a deliberately corrupt input.
func dropPerformanceFixture(t *testing.T, s *Store) {
	t.Helper()
	for _, query := range []string{
		`DROP TRIGGER projection_dirty_INSERT`, `DROP TRIGGER projection_dirty_UPDATE`, `DROP TRIGGER projection_dirty_DELETE`,
		`DROP INDEX events_task_cursor`, `DROP INDEX events_type_cursor`, `DROP INDEX events_task_type_cursor`,
		`DROP TABLE projection_control`, `DROP TABLE stats_groups`, `DROP TABLE stats_prefix`, `DROP TABLE blocked_prefix`, `DROP TABLE blocked_open`,
	} {
		if _, err := s.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
}
