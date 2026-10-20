package projects

import (
	"agent-board/internal/core"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOfflineRestoreInvalidatesEveryProjectAndOwnerRecoveryWorks(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "survives restore")
	credential, _, e := f.r.CreateCredential(ctx, "global", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	f.login(t, f.token.Secret)
	pending, _ := random("p_", 16)
	if _, e = f.r.db.Exec(`INSERT INTO projects VALUES(?,'pending','provisioning',1,'',0)`, pending); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(f.r.options.DataDir, "projects", pending)
	if _, e = privateRoot(dir); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "board.db-journal"), []byte("retained"), 0600); e != nil {
		t.Fatal(e)
	}
	key := append([]byte(nil), f.r.signingKey...)
	f.r.Close()
	f.browser.Close()
	if e = ResetOffline(ctx, f.r.options.DataDir, f.db, f.webDir); e != nil {
		t.Fatal(e)
	}
	if data, e := os.ReadFile(filepath.Join(f.r.options.DataDir, "quarantine", pending, "board.db-journal")); e != nil || string(data) != "retained" {
		t.Fatal("pending bytes lost", e)
	}
	r, e := Open(Options{DataDir: f.r.options.DataDir, DefaultDB: f.db, DefaultWeb: f.webDir, Default: f.store, Offline: true, Config: core.Config{Project: "existing-project"}})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if string(key) == string(r.signingKey) {
		t.Fatal("restore retained HMAC key")
	}
	for _, secret := range []string{f.token.Secret, credential.Secret} {
		if _, e = r.Authenticate(ctx, secret); e == nil {
			t.Fatal("restored credential still accepted")
		}
	}
	var count int
	if e = r.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE state='provisioning'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("pending project still routable", e)
	}
	token, who, e := r.CreateCredential(ctx, "recovery operator", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Authenticate(ctx, token.Secret); e != nil {
		t.Fatal(e)
	}
	for _, pid := range []string{r.defaultID, p.ID} {
		if e = r.RecoverOwner(ctx, pid, who.ID); e != nil {
			t.Fatalf("offline recover owner %s: %v", pid, e)
		}
	}
	list, e := r.List(ctx, who)
	if e != nil || len(list) != 2 {
		t.Fatalf("recovered memberships %#v %v", list, e)
	}
	for _, project := range list {
		if project.Role != "owner" {
			t.Fatal("owner not recovered")
		}
	}
}
func TestDefaultPathRebindingRequiresExplicitOfflineReset(t *testing.T) {
	f := setup(t)
	f.r.Close()
	f.browser.Close()
	_, e := Open(Options{DataDir: f.r.options.DataDir, DefaultDB: filepath.Join(f.root, "different.db"), DefaultWeb: f.webDir, Default: f.store, Offline: true, Config: core.Config{Project: "existing"}})
	if e == nil {
		t.Fatal("changed default DB silently accepted")
	}
}

func TestBootstrapSameChosenOwnerIsIdempotentButDifferentOwnerConflicts(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.r.BootstrapDefaultOwner(ctx, f.token.ID); e != nil {
		t.Fatal(e)
	}
	var version, epoch int64
	if e := f.r.db.QueryRow(`SELECT version,access_epoch FROM memberships WHERE project_id=? AND identity_id=?`, f.r.defaultID, f.who.ID).Scan(&version, &epoch); e != nil {
		t.Fatal(e)
	}
	if e := f.r.BootstrapDefaultOwner(ctx, f.token.ID); e != nil {
		t.Fatalf("bootstrap retry %v", e)
	}
	var nextVersion, nextEpoch int64
	if e := f.r.db.QueryRow(`SELECT version,access_epoch FROM memberships WHERE project_id=? AND identity_id=?`, f.r.defaultID, f.who.ID).Scan(&nextVersion, &nextEpoch); e != nil || version != nextVersion || epoch != nextEpoch {
		t.Fatal("bootstrap retry changed owner epoch/version", e)
	}
	other, e := f.store.CreateToken(ctx, core.TokenInput{Owner: "different", Machine: "machine", TTL: time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	if e = f.r.BootstrapDefaultOwner(ctx, other.ID); e == nil {
		t.Fatal("different operator silently promoted")
	}
}
