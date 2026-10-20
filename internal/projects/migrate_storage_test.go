package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func rowsSnapshot(t *testing.T, db *sql.DB, tables []string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	for _, table := range tables {
		rows, err := db.Query(`SELECT * FROM ` + table + ` ORDER BY 1`)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		all := [][]any{}
		for rows.Next() {
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if err = rows.Scan(targets...); err != nil {
				t.Fatal(err)
			}
			all = append(all, values)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		raw, _ := json.Marshal(all)
		snapshot[table] = string(raw)
	}
	return snapshot
}
func closedSchema2(t *testing.T, path string) map[string]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{`DROP TABLE coordination_metadata`, `DROP TABLE agent_profiles`, `DROP TABLE agent_profile_revisions`, `PRAGMA user_version=2`} {
		if _, err = db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return rowsSnapshot(t, db, []string{"metadata", "tokens", "tasks", "sessions", "role_generations", "events", "idempotency", "notes", "note_metadata"})
}
func TestOfflineStorageMigrationAllBoardsPreservesEveryLegacyRow(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	active := f.create(t, f.who, "active")
	archived := f.create(t, f.who, "archived")
	f.describe(t, active.ID, f.token.Secret, "active-session", "developer")
	f.describe(t, archived.ID, f.token.Secret, "archived-session", "human")
	if _, err := f.r.updateProject(ctx, f.who, archived.ID, archived.Name, "archived", archived.Version); err != nil {
		t.Fatal(err)
	}
	root := f.r.options.DataDir
	f.r.Close()
	f.browser.Close()
	f.store.Close()
	paths := []string{f.db, filepath.Join(root, "projects", active.ID, "board.db"), filepath.Join(root, "projects", archived.ID, "board.db")}
	before := map[string]map[string]string{}
	for _, path := range paths {
		before[path] = closedSchema2(t, path)
	}
	registry, err := sql.Open("sqlite", filepath.Join(root, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	authority := rowsSnapshot(t, registry, []string{"registry_meta", "identities", "credentials", "memberships", "session_epochs", "projects", "repositories", "sessions", "actions", "audit"})
	registry.Close()
	for i := 0; i < 2; i++ {
		count, e := MigrateStorage(ctx, root, f.db, f.webDir)
		if e != nil || count != 3 {
			t.Fatal("migration", count, e)
		}
	}
	for _, path := range paths {
		db, e := sql.Open("sqlite", path)
		if e != nil {
			t.Fatal(e)
		}
		after := rowsSnapshot(t, db, []string{"metadata", "tokens", "tasks", "sessions", "role_generations", "events", "idempotency", "notes", "note_metadata"})
		var version int
		db.QueryRow(`PRAGMA user_version`).Scan(&version)
		db.Close()
		if version != 4 || !reflect.DeepEqual(after, before[path]) {
			t.Fatal("legacy rows changed")
		}
		backups, _ := filepath.Glob(path + ".migrations/schema2-*.db")
		if len(backups) != 1 {
			t.Fatal("missing or repeated migration snapshot")
		}
	}
	registry, err = sql.Open("sqlite", filepath.Join(root, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	if !reflect.DeepEqual(authority, rowsSnapshot(t, registry, []string{"registry_meta", "identities", "credentials", "memberships", "session_epochs", "projects", "repositories", "sessions", "actions", "audit"})) {
		t.Fatal("migration changed registry authority")
	}
}
func TestOfflineStorageMigrationPreflightsAllBoardsBeforeDDL(t *testing.T) {
	for _, damage := range []string{"missing", "future", "incomplete3"} {
		t.Run(damage, func(t *testing.T) {
			f := setup(t)
			p := f.create(t, f.who, "damaged")
			root := f.r.options.DataDir
			f.r.Close()
			f.browser.Close()
			f.store.Close()
			closedSchema2(t, f.db)
			path := filepath.Join(root, "projects", p.ID, "board.db")
			switch damage {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			default:
				db, err := sql.Open("sqlite", path)
				if err != nil {
					t.Fatal(err)
				}
				q := `PRAGMA user_version=99`
				if damage == "incomplete3" {
					q = `DROP TABLE agent_profile_revisions`
				}
				if _, err = db.Exec(q); err != nil {
					t.Fatal(err)
				}
				db.Close()
			}
			if _, err := MigrateStorage(context.Background(), root, f.db, f.webDir); err == nil {
				t.Fatal("damaged published storage accepted")
			}
			db, err := sql.Open("sqlite", f.db)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var version int
			db.QueryRow(`PRAGMA user_version`).Scan(&version)
			if version != 2 {
				t.Fatal("preflight allowed partial DDL")
			}
			if damage == "missing" {
				if _, err = os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("lost board recreated")
				}
			}
		})
	}
}

func TestOfflineResetRetainsProfileRevisionsAndMigratesSchema2(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "profiles3", true: "legacy2"}[legacy], func(t *testing.T) {
			f := setup(t)
			session := f.describe(t, "", f.token.Secret, "profile-session", "human")
			body := `{"request_id":"profile-create","name":"Retained","description":"","role":"developer","instructions":"Version one","provider":"unconfigured","model":""}`
			if out := f.native(t, "POST", "/api/v1/profile_create", f.token.Secret, session.SessionID, body); out.Code != 200 {
				t.Fatal(out.Code, out.Body.String())
			}
			root := f.r.options.DataDir
			f.r.Close()
			f.browser.Close()
			f.store.Close()
			var before map[string]string
			if legacy {
				closedSchema2(t, f.db)
			} else {
				db, err := sql.Open("sqlite", f.db)
				if err != nil {
					t.Fatal(err)
				}
				before = rowsSnapshot(t, db, []string{"agent_profiles", "agent_profile_revisions"})
				if _, err = db.Exec(`DROP TABLE coordination_metadata`); err != nil {
					t.Fatal(err)
				}
				if _, err = db.Exec(`PRAGMA user_version=3`); err != nil {
					t.Fatal(err)
				}
				db.Close()
			}
			if err := ResetOffline(context.Background(), root, f.db, f.webDir); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", f.db)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var version, revoked int
			db.QueryRow(`PRAGMA user_version`).Scan(&version)
			db.QueryRow(`SELECT revoked FROM tokens WHERE id=?`, f.token.ID).Scan(&revoked)
			if version != 4 || revoked != 1 {
				t.Fatal("reset/migration did not complete")
			}
			after := rowsSnapshot(t, db, []string{"agent_profiles", "agent_profile_revisions"})
			if !legacy && !reflect.DeepEqual(before, after) {
				t.Fatal("reset lost profile history")
			}
		})
	}
}
