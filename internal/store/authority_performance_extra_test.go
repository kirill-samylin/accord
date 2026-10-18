package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"bytes"
	"testing"
	"time"
)

func performanceExtraOther(t *testing.T, f *fixture) *Store {
	t.Helper()
	s, err := OpenExisting(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPerformanceExtraExternalTaskSessionConfigAuthority(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer") // A retains the original task/session/config.
	other := performanceExtraOther(t, f)
	title := "external writer"
	changed, err := other.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key}, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	description := "local writer after external commit"
	fresh, err := f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key}, Description: &description})
	if err != nil || fresh.Title != title || fresh.Version != changed.Version+1 {
		t.Fatal("A overwrote B's committed task from stale snapshot", fresh, err)
	}
	opened, err := other.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "architect", RunID: "external", Machine: f.p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "architect", RunID: "local", Machine: f.p.Machine})
	requireCode(t, err, "ROLE_HELD")
	sessions, err := f.s.Sessions(testContext, f.coord, core.ListInput{Role: "architect"})
	if err != nil || len(sessions.Sessions) != 1 || sessions.Sessions[0].SessionID != opened.SessionID {
		t.Fatal("external session absent from fresh catalogue", sessions, err)
	}
	raw, version := configBytes(t, other, func(c *config.Config) { c.Notes.MaxPerRole = 1 })
	applied, err := other.ApplyConfig(testContext, raw, version, "external-unit-admin")
	if err != nil {
		t.Fatal(err)
	}
	dev := f.actor(t, "developer")
	noteInput := core.NoteInput{RequestID: f.id(), Action: "create", ScopeType: "role", ScopeID: "developer", SourceTaskKey: task.Key, Text: "one allowed"}
	if _, err = f.s.Note(testContext, dev, noteInput); err != nil {
		t.Fatal(err)
	}
	noteInput.RequestID = f.id()
	_, err = f.s.Note(testContext, dev, noteInput)
	requireCode(t, err, "LIMIT_EXCEEDED")
	current, err := f.s.ReadConfig(testContext, f.p)
	if err != nil || current.ConfigVersion != applied.ConfigVersion || !bytes.Equal(current.Config, applied.Config) {
		t.Fatal("external config authority remained stale", current, err)
	}
}

func TestPerformanceExtraExternalGenerationReplayCannotResurrectLease(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	input := core.NextInput{RequestID: f.id(), TaskKey: task.Key}
	old, err := f.s.Next(testContext, dev, input)
	if err != nil {
		t.Fatal(err)
	}
	other := performanceExtraOther(t, f)
	if _, err = other.Release(testContext, dev, core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: old.Lease.Generation, Reason: "handover"}); err != nil {
		t.Fatal(err)
	}
	newActor := f.actor(t, "developer")
	current, err := other.Next(testContext, newActor, core.NextInput{RequestID: f.id(), TaskKey: task.Key})
	if err != nil || current.Lease == nil || current.Lease.Generation != old.Lease.Generation+1 {
		t.Fatal(current, err)
	}
	// The durable old response remains replayable, but is never live authority.
	replayed, err := f.s.Next(testContext, dev, input)
	if err != nil || performanceExtraJSON(t, replayed) != performanceExtraJSON(t, old) {
		t.Fatal("replay bytes changed", err)
	}
	_, err = f.s.Comment(testContext, dev, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: old.Lease.Generation}, Text: "stale generation"})
	requireCode(t, err, "LEASE_LOST")
	got, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: task.Key})
	if err != nil || got.Lease == nil || *got.Lease != *current.Lease || len(got.Attempts) != 2 || len(got.Comments) != 0 {
		t.Fatal("old replay revived released ownership", got, err)
	}
}

func TestPerformanceExtraExternalScopesAndRevokePrecedeReplay(t *testing.T) {
	f := newFixture(t)
	other := performanceExtraOther(t, f)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "alice", Machine: "synthetic-machine", TTL: time.Hour, Roles: []string{"developer", "analyst"}})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.ReadConfig(testContext, principal); err != nil {
		t.Fatal(err)
	}
	if _, err = other.SetTokenScopes(testContext, core.ScopesInput{ID: token.ID, Roles: []string{"analyst"}, ExpectedVersion: 1, Actor: "external-unit-admin"}); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Describe(testContext, principal, core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "denied", Machine: principal.Machine})
	requireCode(t, err, "FORBIDDEN_ROLE")
	in := core.OpenSession{RequestID: f.id(), Role: "analyst", RunID: "allowed", Machine: principal.Machine}
	view, err := f.s.Describe(testContext, principal, in)
	if err != nil {
		t.Fatal("allowed scoped role failed", err)
	}
	if err = other.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal(err)
	}
	durable := performanceExtraDump(t, f)
	_, err = f.s.Describe(testContext, principal, in)
	requireCode(t, err, "UNAUTHORIZED")
	_, err = f.s.ReadConfig(testContext, principal)
	requireCode(t, err, "UNAUTHORIZED")
	_, err = f.s.Comment(testContext, core.Actor{Principal: principal, SessionID: view.SessionID}, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, Text: "revoked"})
	requireCode(t, err, "UNAUTHORIZED")
	if performanceExtraDump(t, f) != durable {
		t.Fatal("revoked requests persisted candidate or replay")
	}
	if _, err = f.s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key}); err != nil {
		t.Fatal("unrelated principal broken after revoke", err)
	}
}

func TestPerformanceExtraTamperedCanonicalConfigFailsClosed(t *testing.T) {
	f := newFixture(t)
	before, err := f.s.ReadConfig(testContext, f.p)
	if err != nil {
		t.Fatal(err)
	}
	// Valid canonical bytes for different rules, deliberately retaining old hash.
	raw, _ := configBytes(t, f.s, func(c *config.Config) { c.Notes.MaxPerRole = 1 })
	snapshot, validation := config.Parse(raw, config.Options{})
	if !validation.Valid {
		t.Fatal(validation)
	}
	db := performanceExtraDB(t, f)
	if _, err = db.Exec(`UPDATE config_history SET canonical_json=? WHERE version=?`, string(snapshot.CanonicalJSON()), before.ConfigVersion); err != nil {
		t.Fatal(err)
	}
	durable := performanceExtraDump(t, f)
	_, err = f.s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key})
	requireCode(t, err, "CONFIG_INTERNAL")
	_, err = f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "epic", Title: "must fail closed"})
	requireCode(t, err, "CONFIG_INTERNAL")
	if performanceExtraDump(t, f) != durable {
		t.Fatal("invalid config committed a candidate")
	}
	if _, err = db.Exec(`UPDATE config_history SET canonical_json=? WHERE version=?`, string(before.Config), before.ConfigVersion); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key}); err != nil {
		t.Fatal("cache did not recover after config repair", err)
	}
}

func TestPerformanceExtraExpiredReplayUsesCurrentScopeAuthority(t *testing.T) {
	f := newFixture(t)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "alice", Machine: "replay-machine", TTL: 24 * time.Hour, Roles: []string{"developer", "analyst"}})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	in := core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "expired-replay", Machine: principal.Machine}
	view, err := f.s.Describe(testContext, principal, in)
	if err != nil {
		t.Fatal(err)
	}
	actor := core.Actor{Principal: principal, SessionID: view.SessionID}
	create := core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "scope replay", Description: "implement", Role: "developer"}
	created, err := f.s.Create(testContext, actor, create)
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(time.Hour)
	replayed, err := f.s.Describe(testContext, principal, in)
	if err != nil || performanceExtraJSON(t, replayed) != performanceExtraJSON(t, view) {
		t.Fatal("expiry alone must preserve exact replay without renewal", err)
	}
	replayedTask, err := f.s.Create(testContext, actor, create)
	if err != nil || performanceExtraJSON(t, replayedTask) != performanceExtraJSON(t, created) {
		t.Fatal("expired task mutation replay must remain exact with current scope", err)
	}
	other := performanceExtraOther(t, f)
	if _, err = other.SetTokenScopes(testContext, core.ScopesInput{ID: token.ID, Roles: []string{"analyst"}, ExpectedVersion: 1, Actor: "external-unit-admin"}); err != nil {
		t.Fatal(err)
	}
	durable := performanceExtraDump(t, f)
	_, err = f.s.Describe(testContext, principal, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	_, err = f.s.Create(testContext, actor, create)
	requireCode(t, err, "FORBIDDEN_ROLE")
	if performanceExtraDump(t, f) != durable {
		t.Fatal("scope-denied replay renewed expired session or changed audit")
	}
	if _, err = f.s.Describe(testContext, principal, core.OpenSession{RequestID: f.id(), Role: "analyst", RunID: "allowed", Machine: principal.Machine}); err != nil {
		t.Fatal("remaining scope no longer works", err)
	}
}
