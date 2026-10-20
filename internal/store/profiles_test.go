package store

import (
	"agent-board/internal/core"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func settings() core.ProfileSettings {
	return core.ProfileSettings{Name: "Developer", Description: "Project member", Role: "developer", Instructions: "Review the contract first.", Provider: "unconfigured", Model: ""}
}
func createProfile(t *testing.T, f *fixture) core.AgentProfile {
	t.Helper()
	v, err := f.s.ProfileCreate(testContext, f.coord, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: settings()})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestProfileVersionsSnapshotReplayArchiveAndRestart(t *testing.T) {
	f := newFixture(t)
	profile := createProfile(t, f)
	same := createProfile(t, f)
	if same.ID == profile.ID {
		t.Fatal("names merged")
	}
	open := core.OpenSession{RequestID: f.id(), RunID: "explicit", Machine: f.p.Machine, Role: profile.Role, ProfileID: profile.ID, ProfileVersion: 1}
	bound, err := f.s.Describe(testContext, f.p, open)
	if err != nil || bound.Profile == nil || bound.Profile.Instructions != profile.Instructions {
		t.Fatal("binding", err)
	}
	v := settings()
	v.Instructions = "Changed"
	v.Role = "reviewer"
	updated, err := f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: f.id(), ProfileID: profile.ID, Version: 1, ProfileSettings: v})
	if err != nil || updated.Version != 2 {
		t.Fatal(err)
	}
	replay, err := f.s.Describe(testContext, f.p, open)
	if err != nil || replay.SessionID != bound.SessionID || replay.Role != "developer" || replay.Profile.Instructions != profile.Instructions {
		t.Fatal("snapshot changed", err)
	}
	archiveInput := core.ProfileArchiveInput{RequestID: f.id(), ProfileID: profile.ID, Version: 2, State: "archived"}
	archived, err := f.s.ProfileArchive(testContext, f.coord, archiveInput)
	if err != nil || archived.Version != 3 {
		t.Fatal(err)
	}
	exact, err := f.s.ProfileArchive(testContext, f.coord, archiveInput)
	if err != nil || exact.Version != 3 {
		t.Fatal("archive replay", err)
	}
	open.RequestID = f.id()
	_, err = f.s.Describe(testContext, f.p, open)
	requireCode(t, err, "CONFLICT")
	open.ProfileVersion = 3
	open.Role = "reviewer"
	_, err = f.s.Describe(testContext, f.p, open)
	requireCode(t, err, "CONFLICT")
	restore, err := f.s.ProfileArchive(testContext, f.coord, core.ProfileArchiveInput{RequestID: f.id(), ProfileID: profile.ID, Version: 3, State: "active"})
	if err != nil || restore.Version != 4 {
		t.Fatal(err)
	}
	read, err := f.s.Profile(testContext, f.coord, core.ProfileInput{ProfileID: profile.ID, Version: 1})
	if err != nil || read.Profile.Instructions != profile.Instructions || len(read.Revisions) != 4 {
		t.Fatal("immutable revision lost", err)
	}
	page, err := f.s.Profiles(testContext, f.coord, core.ProfilesInput{State: "all"})
	if err != nil || len(page.Profiles) != 2 || page.Profiles[0].Instructions != "" {
		t.Fatal("summary leak", err)
	}
	f.s.Close()
	f.s, err = Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := f.s.Heartbeat(testContext, core.Actor{Principal: f.p, SessionID: bound.SessionID}, core.HeartbeatInput{RequestID: f.id()})
	if err != nil || hb.Profile.Version != 1 || hb.Role != "developer" {
		t.Fatal("restart snapshot", err)
	}
	agents, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: bound.SessionID})
	if err != nil || len(agents.Sessions) != 1 || agents.Sessions[0].Profile.Version != 1 {
		t.Fatal("control snapshot", err)
	}
}
func TestProfileInvalidMutationIsAtomicAndBounded(t *testing.T) {
	f := newFixture(t)
	profile := createProfile(t, f)
	for _, change := range []func(*core.ProfileSettings){func(v *core.ProfileSettings) { v.Name = "" }, func(v *core.ProfileSettings) { v.Role = "human" }, func(v *core.ProfileSettings) { v.Instructions = strings.Repeat("x", 16385) }, func(v *core.ProfileSettings) { v.Model = "bad\nmodel" }, func(v *core.ProfileSettings) { v.Provider = "exec" }, func(v *core.ProfileSettings) { v.Description = strings.Repeat("x", 4097) }} {
		v := settings()
		change(&v)
		rid := f.id()
		_, err := f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: rid, ProfileID: profile.ID, Version: 1, ProfileSettings: v})
		requireCode(t, err, "VALIDATION")
		var count int
		if err = f.s.db.QueryRow(`SELECT count(*) FROM agent_profile_revisions WHERE profile_id=?`, profile.ID).Scan(&count); err != nil || count != 1 {
			t.Fatal("invalid mutation persisted revision", err)
		}
	}
	_, err := f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: f.id(), ProfileID: profile.ID, Version: 0, ProfileSettings: settings()})
	requireCode(t, err, "CONFLICT")
	for i := 1; i < 128; i++ {
		createProfile(t, f)
	}
	_, err = f.s.ProfileCreate(testContext, f.coord, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: settings()})
	requireCode(t, err, "CONFLICT")
	var count int
	f.s.db.QueryRow(`SELECT count(*) FROM agent_profiles`).Scan(&count)
	if count != 128 {
		t.Fatal("quota bypass")
	}
	_, err = f.s.db.Exec(`UPDATE agent_profiles SET data=json_set(data,'$.version',1024) WHERE key=?`, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.ProfileArchive(testContext, f.coord, core.ProfileArchiveInput{RequestID: f.id(), ProfileID: profile.ID, Version: 1024, State: "archived"})
	requireCode(t, err, "CONFLICT")
}
func TestProfileDescribeStrictPairRoleAndNoIdentityAuthority(t *testing.T) {
	f := newFixture(t)
	profile := createProfile(t, f)
	for _, in := range []core.OpenSession{{ProfileID: profile.ID}, {ProfileVersion: 1}, {ProfileID: profile.ID, ProfileVersion: -1}, {ProfileID: profile.ID, ProfileVersion: 1, Role: "human"}} {
		in.RequestID = f.id()
		in.RunID = "explicit"
		in.Machine = f.p.Machine
		if in.Role == "" {
			in.Role = "developer"
		}
		_, err := f.s.Describe(testContext, f.p, in)
		requireCode(t, err, "VALIDATION")
	}
	_, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), RunID: "foreign", Machine: f.p.Machine, Role: "developer", ProfileID: "ap_" + strings.Repeat("0", 32), ProfileVersion: 1})
	requireCode(t, err, "NOT_FOUND")
	body, _ := json.Marshal(profile)
	if strings.Contains(string(body), f.p.ID) || profile.CreatedBy != f.coord.SessionID {
		t.Fatal("credential ID exposed")
	}
}
func TestProfileMigrationSchema2PreservesRowsAndRejectsDamagedCurrent(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	var before string
	f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&before)
	f.s.db.Exec(`DROP TABLE coordination_metadata`)
	f.s.db.Exec(`DROP TABLE agent_profiles`)
	f.s.db.Exec(`DROP TABLE agent_profile_revisions`)
	f.s.db.Exec(`PRAGMA user_version=2`)
	f.s.Close()
	var err error
	f.s, err = Open(f.path, core.Config{}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	var after string
	f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&after)
	if after != before {
		t.Fatal("migration rewrote task")
	}
	snapshots, _ := filepath.Glob(f.path + ".migrations/schema2-*.db")
	if len(snapshots) != 1 {
		t.Fatal("missing durable schema2 snapshot")
	}
	if _, err = os.Stat(snapshots[0] + ".sha256"); err != nil {
		t.Fatal(err)
	}
	f.s.db.Exec(`DROP TABLE agent_profile_revisions`)
	f.s.Close()
	if s, e := Open(f.path, core.Config{}, nil); e == nil {
		s.Close()
		t.Fatal("damaged current schema repaired")
	}
	db, err := sql.Open("sqlite", f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='agent_profile_revisions'`).Scan(&count)
	if count != 0 {
		t.Fatal("missing table recreated")
	}
}

func TestProfileEditDoesNotChangeLeasesAndCannotReplayAfterRevoke(t *testing.T) {
	f := newFixture(t)
	profile := createProfile(t, f)
	open := core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "native", Machine: f.p.Machine, ProfileID: profile.ID, ProfileVersion: 1}
	session, err := f.s.Describe(testContext, f.p, open)
	if err != nil {
		t.Fatal(err)
	}
	actor := core.Actor{Principal: f.p, SessionID: session.SessionID}
	task := f.task(t, "developer")
	claim := f.claim(t, actor, task.Key)
	changed := settings()
	changed.Role = "reviewer"
	changed.Instructions = "New role"
	_, err = f.s.ProfileUpdate(testContext, f.coord, core.ProfileUpdateInput{RequestID: f.id(), ProfileID: profile.ID, Version: 1, ProfileSettings: changed})
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: task.Key})
	if err != nil || current.Generation != claim.Lease.Generation || current.Lease.SessionID != session.SessionID || !current.Lease.ExpiresAt.Equal(claim.Lease.ExpiresAt) {
		t.Fatal("profile edit changed active lease", err)
	}
	hb, err := f.s.Heartbeat(testContext, actor, core.HeartbeatInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation})
	if err != nil || hb.Role != "developer" || hb.Profile.Version != 1 {
		t.Fatal("profile edit replaced session", err)
	}
	create := core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: settings()}
	if _, err = f.s.ProfileCreate(testContext, f.coord, create); err != nil {
		t.Fatal(err)
	}
	if err = f.s.RevokeToken(testContext, f.p.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.ProfileCreate(testContext, f.coord, create)
	requireCode(t, err, "UNAUTHORIZED")
	_, err = f.s.Describe(testContext, f.p, open)
	requireCode(t, err, "UNAUTHORIZED")
}

func TestMigrateExistingNeverBootstrapsMissingOrEmptyBoard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if err := MigrateExisting(testContext, path); err == nil {
		t.Fatal("missing board bootstrapped")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("missing board created")
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := MigrateExisting(testContext, path); err == nil {
		t.Fatal("empty board initialized")
	}
	if info, err := os.Stat(path); err != nil || info.Size() != 0 {
		t.Fatal("empty board modified")
	}
}
