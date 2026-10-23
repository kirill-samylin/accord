package store

import (
	"agent-board/internal/core"
	"path/filepath"
	"testing"
)

func TestManagedSchema4MigrationPreservesTaskBytesAndRejectsDamaged5(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	var before string
	if err := f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&before); err != nil {
		t.Fatal(err)
	}
	dropAutomationFixture(t, f.s)
	for _, q := range []string{`DROP TABLE managed_hosts`, `DROP TABLE managed_runs`, `DROP TABLE managed_commands`, `DROP TABLE managed_metadata`, `PRAGMA user_version=4`} {
		if _, err := f.s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	f.s.Close()
	var err error
	f.s, err = Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	var after string
	if err = f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&after); err != nil || before != after {
		t.Fatal("migration changed legacy task bytes", err)
	}
	snapshots, _ := filepath.Glob(f.path + ".migrations/schema4-*.db")
	if len(snapshots) != 1 {
		t.Fatal("missing durable schema4 snapshot")
	}
	if _, err = f.s.db.Exec(`DROP TABLE managed_commands`); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	if s, err := Open(f.path, core.Config{}, f.clock.read); err == nil {
		s.Close()
		t.Fatal("damaged declared schema5 repaired")
	}
}
func TestManagedSchema5RejectsExtraMetadataRows(t *testing.T) {
	for _, rows := range []string{"(1,1),(2,1)", "(2,1)"} {
		t.Run(rows, func(t *testing.T) {
			f := newFixture(t)
			for _, q := range []string{`DROP TABLE managed_metadata`, `CREATE TABLE managed_metadata(id INTEGER,version INTEGER)`, `INSERT INTO managed_metadata VALUES` + rows} {
				if _, err := f.s.db.Exec(q); err != nil {
					t.Fatal(err)
				}
			}
			f.s.Close()
			if s, err := Open(f.path, core.Config{}, f.clock.read); err == nil {
				s.Close()
				t.Fatal("extra managed metadata rows accepted")
			}
		})
	}
}
