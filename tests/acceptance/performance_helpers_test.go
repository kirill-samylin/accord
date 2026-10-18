package acceptance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Complete logical SQL state includes triggers/indexes and every business table.
// Sorting canonical row encodings avoids relying on physical query row order.
func sqlState(t *testing.T, db *sql.DB) string {
	t.Helper()
	tables, err := db.Query("SELECT name FROM sqlite_schema WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for tables.Next() {
		var name string
		if err := tables.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := tables.Err(); err != nil {
		t.Fatal(err)
	}
	tables.Close()
	names = append(names, "sqlite_schema")
	state := map[string]string{}
	for _, name := range names {
		state[name] = tableState(t, db, name)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func tableState(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	rows, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(table, `"`, `""`) + `"`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var result []string
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(values)
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, string(raw))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(result)
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func legacyD3Binary(t *testing.T) string {
	t.Helper()
	path := os.Getenv("ACCORD_LEGACY_D_BINARY")
	if path == "" {
		t.Skip("NOT-RUN: set ACCORD_LEGACY_D_BINARY to pinned native D schema3 executable")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatal("D3 executable unavailable")
	}
	return path
}

func nativeD3Fixture(t *testing.T) (*board, *client) {
	t.Helper()
	legacy := legacyD3Binary(t)
	b := newBoard(t, "30s")
	b.executable = legacy
	b.start()
	token := filepath.Join(filepath.Dir(b.db), "d3.token")
	result := commandPath(legacy, nil, "token", "create", "--db", b.db, "--owner", "d3-owner", "--machine", "d3", "--output", token)
	fields := strings.Fields(string(result.data))
	if result.err != nil || len(fields) < 2 {
		t.Fatal("native D3 mint failed")
	}
	c := &client{b: b, token: token, tokenID: fields[1]}
	c.open(t, "coordinator", "d3")
	epic := c.epic(t)
	task := c.task(t, epic.Key, "developer")
	c.ok(t, "note", payload{"action": "create", "scope_type": "epic", "scope_id": epic.Key, "source_task_key": task.Key, "text": "native D3 synthetic note"})
	b.stop()
	b.executable = ""
	db := inspectDatabase(t, b.db)
	defer db.Close()
	if sqlString(t, db, "PRAGMA user_version") != "3" {
		t.Fatal("historical D did not create native schema3")
	}
	t.Logf("native D3 binary sha256=%x", hashFile(t, legacy))
	return b, c
}

func assertSchema4Projection(t *testing.T, db *sql.DB) {
	t.Helper()
	if sqlString(t, db, "PRAGMA user_version") != fmt.Sprint(expectedCurrentSchema) || sqlString(t, db, "SELECT built_cursor FROM projection_control") != sqlString(t, db, "SELECT coalesce(max(cursor),0) FROM events") || sqlString(t, db, "SELECT dirty FROM projection_control") != "0" || sqlString(t, db, "SELECT compatibility FROM projection_control") != "0" {
		t.Fatal("schema4 projection not completely built/clean/compatible")
	}
	if sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE type='index' AND name IN ('events_task_cursor','events_type_cursor','events_task_type_cursor')") != "3" || sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE type='trigger' AND name LIKE 'projection_dirty_%'") != "3" || sqlString(t, db, "SELECT count(*)>0 FROM stats_prefix") != "1" {
		t.Fatal("schema4 indexes/triggers/stats backfill absent")
	}
}
