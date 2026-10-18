package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func configBytes(t *testing.T, s *Store, edit func(*config.Config)) ([]byte, int64) {
	t.Helper()
	v, err := s.Config(testContext)
	if err != nil {
		t.Fatal(err)
	}
	var c config.Config
	if err = json.Unmarshal(v.Config, &c); err != nil {
		t.Fatal(err)
	}
	edit(&c)
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return raw, v.ConfigVersion
}
func resetConfig(t *testing.T, f *fixture, edit func(*config.Config)) {
	t.Helper()
	f.clock.advance(2 * time.Hour)
	f.configure(t, edit)
	v, err := f.s.Config(testContext)
	if err != nil {
		t.Fatal(err)
	}
	var c config.Config
	if err = json.Unmarshal(v.Config, &c); err != nil {
		t.Fatal(err)
	}
	roles := []string{}
	for role := range c.Roles {
		roles = append(roles, role)
	}
	tokens, err := f.s.ListTokens(testContext)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range tokens {
		if token.ID == f.p.ID {
			_, err = f.s.SetTokenScopes(testContext, core.ScopesInput{ID: f.p.ID, Roles: roles, ExpectedVersion: token.ScopesVersion, Actor: "unit-admin"})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	f.coord = f.actor(t, "coordinator")
}

func TestConfigCASAcrossConnectionsNoOpAndFailureAtomicity(t *testing.T) {
	f := newFixture(t)
	other, err := OpenExisting(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	raw, version := configBytes(t, f.s, func(c *config.Config) { c.Notes.MaxPerRole = 12 })
	stores := []*Store{f.s, other}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, s := range stores {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			_, err := s.ApplyConfig(testContext, raw, version, "unit-admin")
			results <- err
		}(s)
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else {
			requireCode(t, err, "CONFIG_CONFLICT")
		}
	}
	if winners != 1 {
		t.Fatalf("winners %d", winners)
	}
	active, err := f.s.Config(testContext)
	if err != nil || active.ConfigVersion != version+1 {
		t.Fatal(active, err)
	}
	noOp, err := other.ApplyConfig(testContext, raw, active.ConfigVersion, "noop-admin")
	if err != nil || noOp.ConfigVersion != active.ConfigVersion {
		t.Fatal("no-op rewrote version", err)
	}
	_, err = f.s.ApplyConfig(testContext, raw, version, "stale-admin")
	requireCode(t, err, "CONFIG_CONFLICT")
	_, err = f.s.ApplyConfig(testContext, []byte("version: 99"), active.ConfigVersion, "invalid-admin")
	if err == nil {
		t.Fatal("invalid config accepted")
	}
	history, err := f.s.ConfigHistory(testContext, 0, 1)
	if err != nil || len(history.Revisions) != 1 || !history.HasMore || history.NextVersion != 1 {
		t.Fatal(history, err)
	}
	page, err := f.s.ConfigHistory(testContext, 1, 200)
	if err != nil || len(page.Revisions) != 1 || page.HasMore {
		t.Fatal(page, err)
	}
	var count int
	f.s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE json_extract(data,'$.type')='config_applied'`).Scan(&count)
	if count != 2 {
		t.Fatalf("audited applies %d", count)
	}
}
func TestLiveConfigRejectsChangesAndPreservesLeases(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, d, task.Key)
	for name, edit := range map[string]func(*config.Config){"ttl": func(c *config.Config) { c.Leases.TaskTTL = "2m" }, "evidence": func(c *config.Config) {
		r := c.Roles["developer"]
		r.EvidenceOnDone = append(r.EvidenceOnDone, "pr")
		c.Roles["developer"] = r
	}, "singleton": func(c *config.Config) { r := c.Roles["developer"]; r.Singleton = true; c.Roles["developer"] = r }} {
		t.Run(name, func(t *testing.T) {
			raw, version := configBytes(t, f.s, edit)
			_, err := f.s.ApplyConfig(testContext, raw, version, "unit-admin")
			requireCode(t, err, "CONFIG_IN_USE")
		})
	}
	after, err := f.s.Get(testContext, d, core.GetInput{TaskKey: task.Key})
	if err != nil || after.Lease == nil || *after.Lease != *claim.Lease {
		t.Fatal("failed apply changed lease", after, err)
	}
	history, err := f.s.ConfigHistory(testContext, 0, 200)
	if err != nil || len(history.Revisions) != 1 {
		t.Fatal("failed apply left history", err)
	}
	// Mutable bootstrap configuration has no authority after migration.
	f.s.cfg.LeaseTTL = time.Hour
	f.s.cfg.Notes.MaxPerRole = 1
	got := f.configure(t, func(c *config.Config) { c.Notes.MaxPerRole = 7 })
	second, err := Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	view, err := second.Describe(testContext, f.p, core.OpenSession{RequestID: "reload", Role: "analyst", RunID: "reload", Machine: f.p.Machine})
	if err != nil || view.ConfigVersion != got.ConfigVersion || view.TaskTTLSeconds != 60 {
		t.Fatal(view, err)
	}
}
func TestScopesDefaultExplicitCASAndLiveRemoval(t *testing.T) {
	f := newFixture(t)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "alice", Machine: "other", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Describe(testContext, p, core.OpenSession{RequestID: "no-human", Role: "human", RunID: "r", Machine: p.Machine})
	requireCode(t, err, "FORBIDDEN_ROLE")
	info, err := f.s.SetTokenScopes(testContext, core.ScopesInput{ID: token.ID, Roles: []string{"human"}, ExpectedVersion: 1, Actor: "admin"})
	if err != nil || info.ScopesVersion != 2 {
		t.Fatal(info, err)
	}
	_, err = f.s.SetTokenScopes(testContext, core.ScopesInput{ID: token.ID, Roles: []string{"developer"}, ExpectedVersion: 1, Actor: "admin"})
	requireCode(t, err, "CONFIG_CONFLICT")
	sess, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: "human", Role: "human", RunID: "r", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.SetTokenScopes(testContext, core.ScopesInput{ID: token.ID, Roles: []string{"developer"}, ExpectedVersion: 2, Actor: "admin"})
	requireCode(t, err, "CONFIG_IN_USE")
	if sess.Role != "human" {
		t.Fatal(sess)
	}
	_, err = f.s.CreateToken(testContext, core.TokenInput{Owner: "not-allowed", Machine: "other", TTL: time.Hour, Roles: []string{"human"}})
	requireCode(t, err, "FORBIDDEN_ROLE")
}
func TestStartupExplicitFlagsAndInitialApply(t *testing.T) {
	f := newFixture(t)
	if s, err := Open(f.path, core.Config{Project: "wrong", ExplicitProject: true}, nil); err == nil {
		s.Close()
		t.Fatal("explicit project replaced config")
	}
	if s, err := Open(f.path, core.Config{LeaseTTL: 2 * time.Minute, ExplicitTaskTTL: true}, nil); err == nil {
		s.Close()
		t.Fatal("explicit TTL replaced config")
	}
}

func TestConfigAuditFailureRollsBackHistoryAndActiveVersion(t *testing.T) {
	f := newFixture(t)
	raw, version := configBytes(t, f.s, func(c *config.Config) { c.Notes.MaxPerRole = 8 })
	if _, err := f.s.db.Exec(`CREATE TRIGGER reject_config_audit BEFORE INSERT ON events WHEN json_extract(NEW.data,'$.type')='config_applied' BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.ApplyConfig(testContext, raw, version, "operator"); err == nil {
		t.Fatal("apply survived audit failure")
	}
	active, err := f.s.Config(testContext)
	if err != nil || active.ConfigVersion != version {
		t.Fatal("failed apply changed active", active, err)
	}
	history, err := f.s.ConfigHistory(testContext, 0, 200)
	if err != nil || len(history.Revisions) != 1 {
		t.Fatal("failed apply left history", history, err)
	}
}
