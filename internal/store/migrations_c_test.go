package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"agent-board/internal/core"
)

func schema1Fixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	for _, q := range []string{`DROP TABLE managed_hosts`, `DROP TABLE managed_runs`, `DROP TABLE managed_commands`, `DROP TABLE managed_metadata`, `DROP TABLE coordination_metadata`, `DROP TABLE agent_profiles`, `DROP TABLE agent_profile_revisions`, `DROP TABLE notes`, `DROP TABLE note_metadata`, `PRAGMA user_version=1`} {
		if _, err := f.s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	return f
}
func TestMigrationSchema1BackupAndNoInventedHistoricalTimes(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	// Strip fields to mirror a historical JSON row. The migration does not
	// fabricate timestamps from ready_at or rewrite exact replay responses.
	_, err := f.s.db.Exec(`UPDATE tasks SET data=json_remove(data,'$.created_at') WHERE key=?`, task.Key)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`DROP TABLE managed_hosts`, `DROP TABLE managed_runs`, `DROP TABLE managed_commands`, `DROP TABLE managed_metadata`, `DROP TABLE coordination_metadata`, `DROP TABLE agent_profiles`, `DROP TABLE agent_profile_revisions`, `DROP TABLE notes`, `DROP TABLE note_metadata`, `PRAGMA user_version=1`} {
		if _, err = f.s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	f.s.Close()
	s, err := Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version int
	if err = s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != SchemaVersion {
		t.Fatalf("version %d %v", version, err)
	}
	var created sql.NullString
	if err = s.db.QueryRow(`SELECT json_extract(data,'$.created_at') FROM tasks WHERE key=?`, task.Key).Scan(&created); err != nil || created.Valid {
		t.Fatal("migration invented timestamp")
	}
	snapshots, err := filepath.Glob(f.path + ".migrations/schema1-*.db")
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("snapshots %v %v", snapshots, err)
	}
	backup, err := openReadOnly(testContext, snapshots[0])
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err = backup.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 1 {
		t.Fatal("pre-migration snapshot not schema1")
	}
	if err = checkSnapshot(testContext, snapshots[0]); err != nil {
		t.Fatal(err)
	}
}
func TestMigrationBackupFailureLeavesSchema1Untouched(t *testing.T) {
	f := schema1Fixture(t)
	if err := os.WriteFile(f.path+".migrations", []byte("block directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(f.path, core.Config{}, f.clock.read); err == nil {
		s.Close()
		t.Fatal("migration ignored backup failure")
	}
	db, err := openReadOnly(testContext, f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 1 {
		t.Fatal("failed migration changed version")
	}
	var count int
	db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='notes'`).Scan(&count)
	if count != 0 {
		t.Fatal("failed migration applied DDL")
	}
}
func TestMigrationDDLFailureAtomicAndNewerRejected(t *testing.T) {
	f := schema1Fixture(t)
	db, err := sql.Open("sqlite", f.path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE notes (bad INTEGER)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if s, err := Open(f.path, core.Config{}, f.clock.read); err == nil {
		s.Close()
		t.Fatal("migration accepted conflicting DDL")
	}
	db, err = sql.Open("sqlite", f.path)
	if err != nil {
		t.Fatal(err)
	}
	var version, count int
	db.QueryRow(`PRAGMA user_version`).Scan(&version)
	db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='note_metadata'`).Scan(&count)
	if version != 1 || count != 0 {
		t.Fatal("DDL failure partly committed")
	}
	if _, err = db.Exec(`PRAGMA user_version=99`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if s, err := Open(f.path, core.Config{}, f.clock.read); err == nil {
		s.Close()
		t.Fatal("future schema accepted")
	}
	db, err = sql.Open("sqlite", f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 99 {
		t.Fatal("newer schema rewritten")
	}
}
func TestCatalogInvalidFiltersOnEmptyResults(t *testing.T) {
	f := newFixture(t)
	_, err := f.s.ListTasks(testContext, f.coord, core.ListInput{State: "typo"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.ListEpics(testContext, f.coord, core.ListInput{State: "ready"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.Sessions(testContext, f.coord, core.ListInput{State: "typo", Role: "developer"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.ListNotes(testContext, f.coord, core.NotesInput{ScopeType: "typo"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.ListNotes(testContext, f.coord, core.NotesInput{ScopeType: "role", ScopeID: "unknown"})
	requireCode(t, err, "VALIDATION")
}

func TestLegacyReworkKeepsUnknownFirstTimestamps(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	task := f.task(t, "developer")
	f.finish(t, d, f.claim(t, d, task.Key), core.Evidence{Commit: commitA, RunSummary: "original"}, "")
	defect := f.task(t, "coordinator")
	f.finish(t, f.coord, f.claim(t, f.coord, defect.Key), core.Evidence{Summary: "defect"}, "")
	if _, err := f.s.db.Exec(`UPDATE tasks SET data=json_remove(data,'$.created_at','$.first_claim_at','$.first_done_at') WHERE key=?`, task.Key); err != nil {
		t.Fatal(err)
	}
	_, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Reason: "fix"}, State: "ready", EvidenceRef: defect.Key})
	if err != nil {
		t.Fatal(err)
	}
	claim := f.claim(t, d, task.Key)
	if claim.Task.FirstClaimAt != nil {
		t.Fatal("legacy claim invented first timestamp")
	}
	done := f.finish(t, d, claim, core.Evidence{Commit: commitB, RunSummary: "rework"}, "")
	if done.FirstDoneAt != nil || done.CreatedAt != nil {
		t.Fatal("legacy rework invented first timestamp")
	}
}
