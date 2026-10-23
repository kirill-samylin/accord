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

// Inspect actual migrated/snapshot storage; a version number alone does not
// prove the automation authority floor or its execution uniqueness constraints.
func assertSchema6AutomationAuthority(t *testing.T, db *sql.DB) {
	t.Helper()
	if got := sqlString(t, db, "PRAGMA user_version"); got != "6" {
		t.Fatalf("published database schema=%s, want schema6", got)
	}
	for _, table := range []string{"automation_policies", "automation_policy_revisions", "workflows", "workflow_steps", "workflow_plans", "workflow_questions", "workflow_attestations", "execution_capabilities", "workflow_reservations"} {
		if sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE type='table' AND name='"+table+"'") != "1" {
			t.Fatalf("schema6 omitted automation table %s", table)
		}
		rows, err := db.Query("SELECT key,data FROM " + table + " LIMIT 0")
		if err != nil {
			t.Fatalf("schema6 automation columns missing in %s", table)
		}
		if err := rows.Close(); err != nil {
			t.Fatal("automation storage inspection close failed")
		}
	}
	if sqlString(t, db, "SELECT count(*) FROM automation_metadata") != "1" || sqlString(t, db, "SELECT version FROM automation_metadata WHERE id=1") != "1" {
		t.Fatal("schema6 automation1 authority floor missing or changed")
	}
	for name, expected := range map[string]string{
		"automation_step_execution": "CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.workflow_id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))",
		"automation_cap_attempt":    "CREATE UNIQUE INDEX automation_cap_attempt ON execution_capabilities(json_extract(data,'$.run_id'),json_extract(data,'$.generation'))",
	} {
		actual := sqlString(t, db, "SELECT sql FROM sqlite_schema WHERE type='index' AND name='"+name+"'")
		if strings.TrimSpace(actual) != expected {
			t.Fatalf("schema6 execution constraint %s missing or changed", name)
		}
	}
}

func migrationSnapshot(t *testing.T, b *board) string {
	t.Helper()
	paths, err := filepath.Glob(b.db + ".migrations/schema1-*.db")
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
	if sqlString(t, db, "PRAGMA user_version") != "1" || sqlString(t, db, "PRAGMA integrity_check") != "ok" {
		t.Fatal("backup does not retain valid pre-migration schema1")
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
	b.start() // This actual server process performs schema migration.
	snapshot := migrationSnapshot(t, b)
	db := inspectDatabase(t, b.db)
	assertSchema6AutomationAuthority(t, db)
	if sqlString(t, db, "SELECT data FROM tasks WHERE key='T-2'") != taskRow || sqlString(t, db, "SELECT data FROM events WHERE cursor=1") != eventRow || sqlString(t, db, "SELECT response FROM idempotency WHERE key='synthetic-legacy-replay'") != replayRow {
		t.Fatal("migration rewrote historical rows or exact replay payload")
	}
	if sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE name IN ('agent_profiles','agent_profile_revisions')") != "2" {
		t.Fatal("schema4 migration omitted profile authority tables")
	}
	backupDB := inspectDatabase(t, snapshot)
	if sqlString(t, backupDB, "SELECT count(*) FROM sqlite_schema WHERE name='notes'") != "0" {
		t.Fatal("successful migration backup already contains schema2 table")
	}
	if sqlString(t, backupDB, "SELECT data FROM tasks WHERE key='T-2'") != taskRow || sqlString(t, backupDB, "SELECT data FROM events WHERE cursor=1") != eventRow || sqlString(t, backupDB, "SELECT response FROM idempotency WHERE key='synthetic-legacy-replay'") != replayRow {
		t.Fatal("pre-migration backup omitted historical state")
	}
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
	note := decode[core.Note](t, c.ok(t, "note", payload{"action": "create", "scope_type": "epic", "scope_id": "E-1", "source_task_key": "T-2", "text": "Schema2 accepts notes after migration"}))
	stat = decode[core.Stats](t, c.ok(t, "stats", statInput))
	incompleteBeforeRestart := stat.IncompleteEvents
	b.stop()
	b.start()
	migrationSnapshot(t, b) // Restart must not migrate or create another backup.
	notes := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "epic", "scope_id": "E-1"}))
	if len(notes.Notes) != 1 || notes.Notes[0].Key != note.Key {
		t.Fatal("schema2 notes lost on post-migration restart")
	}
	stat = decode[core.Stats](t, c.ok(t, "stats", statInput))
	if !stat.IncompleteData || stat.IncompleteEvents != incompleteBeforeRestart || sqlString(t, db, "SELECT data FROM events WHERE cursor=1") != eventRow {
		t.Fatal("restart erased legacy incomplete proof")
	}
	t.Log("C-04 canonical synthetic schema1 migrated in actual server; backup/hash and historical bytes retained; schema2 restart passed")
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
