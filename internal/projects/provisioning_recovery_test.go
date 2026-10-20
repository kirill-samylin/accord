package projects

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflinePendingRejectionRecoversQuotaPreservesBytesAndPermanentlyRejectsID(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	active := f.create(t, f.who, "active")
	for _, pid := range []string{f.r.defaultID, active.ID} {
		if e := RejectProvisioning(ctx, f.r.options.DataDir, pid); e == nil {
			t.Fatal("published project rejected")
		}
	}
	ids := []string{}
	for i := 0; i < MaxProvisions; i++ {
		id, e := random("p_", 16)
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, id)
		if _, e = f.r.db.Exec(`INSERT INTO projects VALUES(?,'crashed','provisioning',1,?,0)`, id, f.who.ID); e != nil {
			t.Fatal(e)
		}
	}
	next, _ := random("p_", 16)
	if _, e := f.r.createProject(ctx, f.who, next, "must wait", "", ""); e == nil {
		t.Fatal("pending capacity not enforced")
	}
	source := filepath.Join(f.r.options.DataDir, "projects", ids[0])
	if _, e := privateRoot(source); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(source, "board.db-journal"), []byte("raw pending bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := RejectProvisioning(ctx, f.r.options.DataDir, ids[0]); e != nil {
		t.Fatal(e)
	}
	if e := RejectProvisioning(ctx, f.r.options.DataDir, ids[0]); e != nil {
		t.Fatal("recovery retry not idempotent", e)
	}
	if data, e := os.ReadFile(filepath.Join(f.r.options.DataDir, "quarantine", ids[0], "board.db-journal")); e != nil || string(data) != "raw pending bytes" {
		t.Fatal("pending bytes lost", e)
	}
	if _, e := f.r.createProject(ctx, f.who, ids[0], "old live action retry", "", ""); e == nil {
		t.Fatal("old create action revived rejected ID")
	}
	if _, e := f.r.createProject(ctx, f.who, next, "capacity recovered", "", ""); e != nil {
		t.Fatal("capacity not recovered", e)
	}
	// Missing source bytes still get a durable empty recovery set and tombstone.
	if e := RejectProvisioning(ctx, f.r.options.DataDir, ids[1]); e != nil {
		t.Fatal(e)
	}
	if info, e := os.Stat(filepath.Join(f.r.options.DataDir, "quarantine", ids[1])); e != nil || !info.IsDir() {
		t.Fatal("missing-folder rejection not durable", e)
	}
	if _, e := f.r.createProject(ctx, f.who, ids[1], "old missing-folder retry", "", ""); e == nil {
		t.Fatal("missing-folder ID revived")
	}
}

func TestPendingRejectionDoesNotCreateMissingRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-registry")
	id, _ := random("p_", 16)
	if e := RejectProvisioning(context.Background(), path, id); e == nil {
		t.Fatal("missing registry accepted")
	}
	if _, e := os.Lstat(path); !os.IsNotExist(e) {
		t.Fatal("recovery created a missing registry", e)
	}
}

func TestPendingQuarantineRenameBeforeSQLCommitCannotReviveOldCreate(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, _ := random("p_", 16)
	if _, e := f.r.db.Exec(`INSERT INTO projects VALUES(?,'crashed','provisioning',1,?,0)`, id, f.who.ID); e != nil {
		t.Fatal(e)
	}
	source := filepath.Join(f.r.options.DataDir, "projects", id)
	if _, e := privateRoot(source); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(source, "board.db-journal"), []byte("preserved"), 0600); e != nil {
		t.Fatal(e)
	}
	// Simulate power loss after the directory rename fsync, before SQL commit.
	if e := quarantinePending(f.r.options.DataDir, id); e != nil {
		t.Fatal(e)
	}
	if _, e := f.r.createProject(ctx, f.who, id, "old live action", "", ""); e == nil {
		t.Fatal("old action revived durable quarantined ID")
	}
	if _, e := os.Lstat(source); !os.IsNotExist(e) {
		t.Fatal("old action recreated source storage", e)
	}
	if e := RejectProvisioning(ctx, f.r.options.DataDir, id); e != nil {
		t.Fatal("operator retry could not complete rejection", e)
	}
	var n int
	if e := f.r.db.QueryRow(`SELECT COUNT(*) FROM rejected_provisions WHERE id=?`, id).Scan(&n); e != nil || n != 1 {
		t.Fatal("rejection tombstone missing", e)
	}
	if _, e := f.r.createProject(ctx, f.who, id, "another old retry", "", ""); e == nil {
		t.Fatal("rejected ID revived after retry")
	}
}
