package acceptance

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestPerformanceNativeD3MigrationProjectionRestartAndOldRefusalProcess(t *testing.T) {
	b, c := nativeD3Fixture(t)
	before := inspectDatabase(t, b.db)
	// Preserve JSON strings and durable replay payload bytes, rather than DTO equality.
	rawBefore := map[string]string{}
	for _, table := range []string{"tasks", "events", "idempotency", "notes", "config_history", "config_active", "tokens", "sessions"} {
		rawBefore[table] = tableState(t, before, table)
	}
	before.Close()
	// Offline stale apply migrates, then fails CAS without no-op audit or runtime Sweep.
	configBefore := commandPath(legacyD3Binary(t), nil, "config", "get", "--db", b.db)
	candidate, _ := configFile(t, b, func(c *config.Config) {})
	migrated := applyFile(b, candidate, 0)
	localFails(t, migrated, "CONFIG_CONFLICT")
	current := command(nil, "config", "get", "--db", b.db)
	if configBefore.err != nil || current.err != nil || !bytes.Equal(configBefore.data, current.data) {
		t.Fatal("migration changed canonical config")
	}
	snapshot := migrationSnapshotVersion(t, b, 3)
	db := inspectDatabase(t, b.db)
	for table, want := range rawBefore {
		if tableState(t, db, table) != want {
			t.Fatalf("migration changed exact native %s rows", table)
		}
	}
	assertSchema4Projection(t, db)
	snap := inspectDatabase(t, snapshot)
	if sqlString(t, snap, "SELECT count(*) FROM sqlite_schema WHERE name='projection_control'") != "0" {
		t.Fatal("native prebackup already contains performance schema")
	}
	snap.Close()
	b.start()
	stats := decode[core.Stats](t, c.ok(t, "stats", payload{"from": "2000-01-01T00:00:00Z", "to": "2100-01-01T00:00:00Z", "group_by": "owner"}))
	if stats.IncompleteData || stats.Totals.CurrentQueue.Ready != 1 {
		t.Fatal("native backfill/queue incorrect")
	}
	b.stop()
	state := sqlState(t, db)
	b.start()
	b.stop()
	if sqlState(t, db) != state {
		t.Fatal("ordinary schema4 restart rewrote SQL state")
	}
	migrationSnapshotVersion(t, b, 3)
	db.Close()
	raw, err := os.ReadFile(b.db)
	if err != nil {
		t.Fatal(err)
	}
	refused := commandPath(legacyD3Binary(t), nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0")
	after, err := os.ReadFile(b.db)
	if refused.err == nil || string(refused.stderr) != "agent-board: database schema is newer than this server\n" || bytes.Contains(refused.data, []byte("listening")) || err != nil || !bytes.Equal(raw, after) {
		t.Fatalf("old D3 refusal mismatch: err=%v stderr=%q db_unchanged=%v", refused.err, refused.stderr, bytes.Equal(raw, after))
	}
	t.Log("P072-01 native D3→4 raw tables/scopes/config/replay unchanged; complete indexed projection, restart and old D3 refusal")
}

func TestPerformanceSchema3DDLFailureRollsBackAllTablesProcess(t *testing.T) {
	b, _ := nativeD3Fixture(t)
	db := inspectDatabase(t, b.db)
	if _, err := db.Exec("CREATE TABLE stats_prefix(bad INTEGER)"); err != nil {
		t.Fatal(err)
	}
	before := sqlState(t, db)
	db.Close()
	candidate, version := configFile(t, b, func(c *config.Config) {})
	failed := applyFile(b, candidate, version)
	if failed.err == nil {
		t.Fatal("schema4 conflict allowed startup")
	}
	db = inspectDatabase(t, b.db)
	if sqlString(t, db, "PRAGMA user_version") != "3" || sqlState(t, db) != before {
		t.Fatal("failed schema4 migration partially changed source SQL")
	}
	migrationSnapshotVersion(t, b, 3)
	if sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE name IN ('events_task_cursor','events_type_cursor','events_task_type_cursor','projection_control','stats_groups','blocked_prefix','blocked_open')") != "0" {
		t.Fatal("partial schema4 objects survived rollback")
	}
	t.Log(fmt.Sprintf("P072-02 native3→4 DDL conflict rollback: all SQL unchanged, schema%s, verified prebackup", sqlString(t, db, "PRAGMA user_version")))
}
