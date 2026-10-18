package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchema2To3BacksUpAndGrandfathersExactScopes(t *testing.T) {
	f := newFixture(t)
	for _, query := range []string{`DROP TABLE config_active`, `DROP TABLE config_history`, `ALTER TABLE tokens DROP COLUMN roles`, `ALTER TABLE tokens DROP COLUMN scopes_version`, `PRAGMA user_version=2`} {
		if _, err := f.s.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	f.s.Close()
	s, err := Open(f.path, core.Config{Project: "actual-project", SessionTTL: f.s.cfg.SessionTTL, LeaseTTL: f.s.cfg.LeaseTTL}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	backups, err := filepath.Glob(f.path + ".migrations/schema2-*.db")
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	var version int
	s.db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 3 {
		t.Fatal(version)
	}
	tokens, err := s.ListTokens(testContext)
	if err != nil || len(tokens) != 1 || !reflect.DeepEqual(tokens[0].Roles, legacyRoles) {
		t.Fatal(tokens, err)
	}
	v, err := s.Config(testContext)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Project struct{ Name string }
		Leases  map[string]string
		Roles   map[string]any
	}
	if err = json.Unmarshal(v.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Project.Name != "actual-project" || cfg.Leases["task_ttl"] != "1m0s" || len(cfg.Roles) != 7 {
		t.Fatal(cfg)
	}
}
