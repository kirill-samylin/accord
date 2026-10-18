package acceptance

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	_ "modernc.org/sqlite"
)

// Acceptance expectation is deliberately independent of store.SchemaVersion.
const expectedCurrentSchema = 4

func legacyDatabase(t *testing.T, b *board) {
	t.Helper()
	ddl, err := os.ReadFile("testdata/schema1.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := inspectDatabase(t, b.db)
	if _, err = db.Exec(string(ddl)); err != nil {
		t.Fatal("canonical schema1 fixture failed")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(b.db, 0600); err != nil {
		t.Fatal(err)
	}
}

func inspectDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func sqlString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var value string
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatal("fixture inspection query failed")
	}
	return value
}

func migrationSnapshot(t *testing.T, b *board) string { return migrationSnapshotVersion(t, b, 1) }

func migrationSnapshotVersion(t *testing.T, b *board, version int) string {
	t.Helper()
	paths, err := filepath.Glob(fmt.Sprintf("%s.migrations/schema%d-*.db", b.db, version))
	if err != nil || len(paths) != 1 {
		t.Fatal("migration did not publish exactly one backup")
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	checksum, err := os.ReadFile(paths[0] + ".sha256")
	if err != nil || string(checksum) != fmt.Sprintf("%x  %s\n", sha256.Sum256(data), filepath.Base(paths[0])) {
		t.Fatal("pre-migration backup checksum does not match snapshot")
	}
	for _, entry := range []struct {
		path string
		mode os.FileMode
	}{{b.db + ".migrations", 0700}, {paths[0], 0600}, {paths[0] + ".sha256", 0600}} {
		info, err := os.Lstat(entry.path)
		if err != nil || info.Mode().Perm() != entry.mode || info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("migration artifact permissions unsafe")
		}
	}
	db := inspectDatabase(t, paths[0])
	if sqlString(t, db, "PRAGMA user_version") != fmt.Sprint(version) || sqlString(t, db, "PRAGMA integrity_check") != "ok" {
		t.Fatal("backup does not retain valid pre-migration schema")
	}
	return paths[0]
}

func TestCatalogLegacySchemaProcessMigrationBackupRestart(t *testing.T) {
	b := newBoard(t, "30s")
	legacyDatabase(t, b)
	before := inspectDatabase(t, b.db)
	taskRow := sqlString(t, before, "SELECT data FROM tasks WHERE key='T-2'")
	eventRow := sqlString(t, before, "SELECT data FROM events WHERE cursor=1")
	replayRow := sqlString(t, before, "SELECT response FROM idempotency WHERE key='synthetic-legacy-replay'")
	before.Close()
	// Migrate offline before the server's immediate Sweep normalizes entity
	// JSON. The stale expected0 is checked after compatibility migration v1;
	// rejection must leave the historical task/event/replay bytes untouched.
	template := command(nil, "config", "init", "--profile", "davinci")
	if template.err != nil {
		t.Fatal("migration input init failed")
	}
	candidate := filepath.Join(filepath.Dir(b.db), "migration.yaml")
	if err := os.WriteFile(candidate, bytes.ReplaceAll(template.data, []byte("OWNER"), []byte("acceptance")), 0600); err != nil {
		t.Fatal(err)
	}
	migrated := command(nil, "config", "apply", "--db", b.db, "--file", candidate, "--expected-version", "0", "--actor", "qa056", "--bootstrap-owner", "acceptance", "--project", "agent-board", "--task-ttl", "30s", "--session-ttl", "1m")
	localFails(t, migrated, "CONFIG_CONFLICT")
	snapshot := migrationSnapshot(t, b)
	db := inspectDatabase(t, b.db)
	if sqlString(t, db, "PRAGMA user_version") != fmt.Sprint(expectedCurrentSchema) || sqlString(t, db, "SELECT data FROM tasks WHERE key='T-2'") != taskRow || sqlString(t, db, "SELECT data FROM events WHERE cursor=1") != eventRow || sqlString(t, db, "SELECT response FROM idempotency WHERE key='synthetic-legacy-replay'") != replayRow {

		t.Fatal("migration rewrote historical rows or exact replay payload")
	}
	backupDB := inspectDatabase(t, snapshot)
	if sqlString(t, backupDB, "SELECT count(*) FROM sqlite_schema WHERE name='notes'") != "0" {
		t.Fatal("successful migration backup already contains schema2 table")
	}
	if sqlString(t, backupDB, "SELECT data FROM tasks WHERE key='T-2'") != taskRow || sqlString(t, backupDB, "SELECT data FROM events WHERE cursor=1") != eventRow || sqlString(t, backupDB, "SELECT response FROM idempotency WHERE key='synthetic-legacy-replay'") != replayRow {
		t.Fatal("pre-migration backup omitted historical state")
	}
	b.start() // Runtime Sweep may serialize additive default fields after migration.
	c := newClient(t, b, "legacy-reader")
	c.open(t, "coordinator", "legacy-reader")
	legacy := decode[core.Task](t, c.ok(t, "get", payload{"task_key": "T-2"}))
	if legacy.CreatedAt != nil || legacy.FirstClaimAt != nil || legacy.FirstDoneAt != nil {
		t.Fatal("migration invented legacy timing")
	}
	statInput := payload{"from": "2026-10-17T00:00:00Z", "to": "2026-10-19T00:00:00Z", "group_by": "owner"}
	stat := decode[core.Stats](t, c.ok(t, "stats", statInput))
	if !stat.IncompleteData || stat.IncompleteEvents != 1 || stat.Totals.FirstCompletions != 0 {
		t.Fatal("legacy event provenance was inferred instead of marked incomplete")
	}
	note := decode[core.Note](t, c.ok(t, "note", payload{"action": "create", "scope_type": "epic", "scope_id": "E-1", "source_task_key": "T-2", "text": "Schema4 accepts notes after migration"}))
	stat = decode[core.Stats](t, c.ok(t, "stats", statInput))
	incompleteBeforeRestart := stat.IncompleteEvents
	b.stop()
	b.start()
	migrationSnapshot(t, b) // Restart must not migrate or create another backup.
	notes := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "epic", "scope_id": "E-1"}))
	if len(notes.Notes) != 1 || notes.Notes[0].Key != note.Key {
		t.Fatal("schema4 notes lost on post-migration restart")
	}
	stat = decode[core.Stats](t, c.ok(t, "stats", statInput))
	if !stat.IncompleteData || stat.IncompleteEvents != incompleteBeforeRestart || sqlString(t, db, "SELECT data FROM events WHERE cursor=1") != eventRow {
		t.Fatal("restart erased legacy incomplete proof")
	}
	t.Log("C-04 canonical synthetic schema1 migrated by real offline CLI; backup/hash and historical bytes retained; schema4 restart passed")
}

func TestCatalogMigrationProcessFailuresAtomic(t *testing.T) {
	for _, mode := range []string{"backup-failure", "ddl-conflict", "future-schema"} {
		t.Run(mode, func(t *testing.T) {
			b := newBoard(t, "30s")
			legacyDatabase(t, b)
			db := inspectDatabase(t, b.db)
			switch mode {
			case "backup-failure":
				if err := os.WriteFile(b.db+".migrations", []byte("Synthetic backup path obstruction"), 0600); err != nil {
					t.Fatal(err)
				}
			case "ddl-conflict":
				if _, err := db.Exec("CREATE TABLE notes(bad INTEGER)"); err != nil {
					t.Fatal(err)
				}
			case "future-schema":
				if _, err := db.Exec("PRAGMA user_version=99"); err != nil {
					t.Fatal(err)
				}
			}
			task := sqlString(t, db, "SELECT data FROM tasks WHERE key='T-2'")
			db.Close()
			started := time.Now()
			failed := command(nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0")
			if failed.err == nil || bytes.Contains(failed.data, []byte("listening")) || time.Since(started) > 5*time.Second {
				t.Fatal("invalid migration did not reject before listening")
			}
			db = inspectDatabase(t, b.db)
			wantVersion := "1"
			if mode == "future-schema" {
				wantVersion = "99"
			}
			if sqlString(t, db, "PRAGMA user_version") != wantVersion || sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE name='note_metadata'") != "0" || sqlString(t, db, "SELECT data FROM tasks WHERE key='T-2'") != task {
				t.Fatal("failed migration partially committed DDL or rewrote legacy task")
			}
			if mode == "ddl-conflict" {
				if !strings.Contains(sqlString(t, db, "SELECT sql FROM sqlite_schema WHERE name='notes'"), "bad INTEGER") {
					t.Fatal("conflicting original table changed")
				}
				migrationSnapshot(t, b)
			}
		})
	}
}
