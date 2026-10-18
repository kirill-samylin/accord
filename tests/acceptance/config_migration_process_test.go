package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func legacyBinary(t *testing.T) string {
	t.Helper()
	path := os.Getenv("ACCORD_LEGACY_BINARY")
	if path == "" {
		t.Skip("NOT-RUN: set ACCORD_LEGACY_BINARY to verified C schema2 executable")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatal("historical binary unavailable")
	}
	return path
}
func schema2Fixture(t *testing.T, b *board, legacy string) {
	t.Helper()
	r := commandPath(legacy, nil, "token", "create", "--db", b.db, "--owner", "schema2-owner", "--machine", "schema2", "--output", filepath.Join(filepath.Dir(b.db), "legacy.token"))
	if r.err != nil {
		t.Fatalf("native C schema2 fixture: %s", r.stderr)
	}
	db := inspectDatabase(t, b.db)
	defer db.Close()
	if sqlString(t, db, "PRAGMA user_version") != "2" {
		t.Fatal("historical binary did not create native schema2")
	}
}
func TestConfigNativeSchema2MigrationOldBinaryRefusalProcess(t *testing.T) {
	legacy := legacyBinary(t)
	b := newBoard(t, "30s")
	schema2Fixture(t, b, legacy)
	db := inspectDatabase(t, b.db)
	beforeTokens := sqlString(t, db, "SELECT count(*) FROM tokens")
	db.Close()
	b.start()
	snapshot := migrationSnapshotVersion(t, b, 2)
	current := configView(t, b)
	cfg := decode[config.Config](t, current.Config)
	if current.ConfigVersion != 1 || len(cfg.Roles) != 7 || cfg.Leases.TaskTTL != "30s" || cfg.Leases.SessionTTL != "1m0s" {
		t.Fatal("native migration did not preserve actual bootstrap inputs/seven roles")
	}
	r := command(nil, "token", "list", "--db", b.db)
	tokens := decode[[]core.TokenInfo](t, r.data)
	exact := []string{"coordinator", "analyst", "architect", "developer", "reviewer", "tester", "human"}
	if r.err != nil || len(tokens) != 1 || !reflect.DeepEqual(tokens[0].Roles, exact) || tokens[0].ScopesVersion != 1 {
		t.Fatal("legacy scopes were not exact seven historical roles")
	}
	db = inspectDatabase(t, b.db)
	if sqlString(t, db, "PRAGMA user_version") != fmt.Sprint(expectedCurrentSchema) || sqlString(t, db, "SELECT count(*) FROM tokens") != beforeTokens {
		t.Fatal("schema2→4 changed credentials")
	}
	snap := inspectDatabase(t, snapshot)
	if sqlString(t, snap, "SELECT count(*) FROM sqlite_schema WHERE name='config_history'") != "0" {
		t.Fatal("prebackup contains migrated config")
	}
	snap.Close()
	db.Close()
	b.stop()
	b.start()
	migrationSnapshotVersion(t, b, 2)
	if configView(t, b).SHA256 != current.SHA256 {
		t.Fatal("schema4 restart changed persisted config")
	}
	b.stop()
	before, err := os.ReadFile(b.db)
	if err != nil {
		t.Fatal(err)
	}
	refused := commandPath(legacy, nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0")
	if refused.err == nil || string(refused.stderr) != "agent-board: cannot open board database\n" || bytes.Contains(refused.data, []byte("listening")) {
		t.Fatal("historical binary did not reject schema4 before serving")
	}
	after, err := os.ReadFile(b.db)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("historical binary changed schema4 database bytes")
	}
	t.Logf("D056-11 native C2→D4 with verified schema2 snapshot/hash/modes; exact legacy scopes; restart persistence; old binary refuses and leaves DB unchanged (legacy sha256=%x)", hashFile(t, legacy))
}
func hashFile(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

func TestConfigSchema2MigrationDDLFailureAtomicProcess(t *testing.T) {
	b := newBoard(t, "30s")
	schema2Fixture(t, b, legacyBinary(t))
	db := inspectDatabase(t, b.db)
	if _, err := db.Exec("CREATE TABLE config_active(bad INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tokens := sqlString(t, db, "SELECT count(*) FROM tokens")
	events := sqlString(t, db, "SELECT count(*) FROM events")
	db.Close()
	failed := command(nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0", "--bootstrap-owner", "acceptance", "--task-ttl", "30s", "--session-ttl", "1m")
	if failed.err == nil || bytes.Contains(failed.data, []byte("listening")) {
		t.Fatal("DDL conflict allowed startup")
	}
	db = inspectDatabase(t, b.db)
	if sqlString(t, db, "PRAGMA user_version") != "2" || sqlString(t, db, "SELECT count(*) FROM sqlite_schema WHERE name='config_history'") != "0" || sqlString(t, db, "SELECT count(*) FROM pragma_table_info('tokens') WHERE name='roles'") != "0" || sqlString(t, db, "SELECT count(*) FROM tokens") != tokens || sqlString(t, db, "SELECT count(*) FROM events") != events || !strings.Contains(sqlString(t, db, "SELECT sql FROM sqlite_schema WHERE name='config_active'"), "bad INTEGER") {
		t.Fatal("failed migration partially wrote DDL/scopes/config/audit")
	}
	migrationSnapshotVersion(t, b, 2)
	t.Log("D056-12 native schema2 isolated config DDL fault rolls back schema/history/scopes/audit; verified prebackup remains")
}

func TestConfigInitialExplicitApplyYAMLAndHistoryProcess(t *testing.T) {
	b := newBoard(t, "30s")
	template := command(nil, "config", "init", "--profile", "davinci")
	if template.err != nil {
		t.Fatal("config init failed")
	}
	raw := bytes.ReplaceAll(template.data, []byte("OWNER"), []byte("initial-owner"))
	path := filepath.Join(filepath.Dir(b.db), "initial.yaml")
	if err := os.WriteFile(path, append([]byte("# private synthetic YAML\n"), raw...), 0600); err != nil {
		t.Fatal(err)
	}
	valid := command(nil, "config", "validate", path)
	if valid.err != nil || !decode[config.Result](t, valid.data).Valid {
		t.Fatal("explicit YAML not valid")
	}
	r := command(nil, "config", "apply", "--db", b.db, "--file", path, "--expected-version", "0", "--actor", "qa056", "--bootstrap-owner", "initial-owner", "--project", "accord", "--task-ttl", "15m", "--session-ttl", "15m")
	if r.err != nil {
		t.Fatalf("initial explicit apply: %s", r.stderr)
	}
	view := decode[core.ConfigView](t, r.data)
	if view.ConfigVersion != 1 || len(decode[config.Config](t, view.Config).Roles) != 12 {
		t.Fatal("initial apply silently used compatibility profile")
	}
	db := inspectDatabase(t, b.db)
	if !strings.HasPrefix(sqlString(t, db, "SELECT raw_yaml FROM config_history WHERE version=1"), "# private synthetic YAML") {
		t.Fatal("local private history lost raw YAML")
	}
	info, err := os.Stat(b.db)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("raw history DB not private")
	}
	agent := newClient(t, b, "initial-agent")
	b.startOptions()
	status, body := configHTTP(t, agent, "GET", "/api/v1/config/history?after_version=0&limit=100", nil, true)
	if status != 200 || bytes.Contains(body, []byte("private synthetic YAML")) || bytes.Contains(body, []byte("raw_yaml")) {
		t.Fatal("remote history leaked raw YAML")
	}
	var output map[string]json.RawMessage
	if json.Unmarshal(view.Config, &output) != nil {
		t.Fatal("canonical result invalid")
	}
	t.Log(fmt.Sprintf("D056-13 CLI init→YAML validate→initial apply v%d; davinci explicit only; raw comments private and absent from HTTP history", view.ConfigVersion))
}
