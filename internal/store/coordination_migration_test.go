package store

import (
	"agent-board/internal/core"
	"os"
	"path/filepath"
	"testing"
)

func TestCoordinationSchema3MigrationPreservesRowsAndRejectsDamaged4(t *testing.T) {
	f := newFixture(t)
	p := createProfile(t, f)
	task := assignedTask(t, f, p)
	var before string
	if err := f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.db.Exec(`DROP TABLE coordination_metadata`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.db.Exec(`PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	var err error
	f.s, err = Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	var after string
	f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&after)
	if before != after {
		t.Fatal("DDL rewrote exact old JSON")
	}
	backups, _ := filepath.Glob(f.path + ".migrations/schema3-*.db")
	if len(backups) != 1 {
		t.Fatal("missing source snapshot")
	}
	if _, err = os.Stat(backups[0] + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.db.Exec(`DELETE FROM coordination_metadata`); err != nil {
		t.Fatal(err)
	}
	f.s.Close()
	if s, e := Open(f.path, core.Config{}, f.clock.read); e == nil {
		s.Close()
		t.Fatal("damaged4 repaired")
	}
}

func TestCoordinationSchema4RejectsExtraMetadataRows(t *testing.T) {
	f := newFixture(t)
	for _, q := range []string{`DROP TABLE coordination_metadata`, `CREATE TABLE coordination_metadata(id INTEGER,version INTEGER)`, `INSERT INTO coordination_metadata VALUES(1,1),(2,1)`} {
		if _, err := f.s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	f.s.Close()
	if s, e := Open(f.path, core.Config{}, f.clock.read); e == nil {
		s.Close()
		t.Fatal("extra metadata row accepted")
	}
}
