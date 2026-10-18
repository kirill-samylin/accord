package acceptance

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"agent-board/internal/core"
)

func TestCLIOnlineSnapshotAndOfflineRecovery(t *testing.T) {
	b := newBoard(t, "30s")
	coordinator := newClient(t, b, "coordinator")
	worker := newClient(t, b, "worker")
	b.start()
	coordinator.open(t, "coordinator", "coordinator")
	worker.open(t, "developer", "worker")
	epic := coordinator.epic(t)
	input := coordinator.prepare(t, payload{"request_id": "snapshot-create", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Snapshot process fixture", "description": "Isolated historical state"})
	created := coordinator.raw("create", input)
	if created.err != nil {
		t.Fatal("snapshot fixture create failed")
	}
	task := decode[core.Task](t, created.data)
	claim := worker.claim(t, task.Key)
	worker.ok(t, "comment", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "text": "Synthetic persisted comment"})
	worker.ok(t, "note", payload{"action": "create", "scope_type": "role", "scope_id": "developer", "source_task_key": task.Key, "text": "Synthetic schema4 backup note"})
	beforeNotes := decode[core.NotesPage](t, worker.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer"}))
	beforeConfig := configView(t, b)
	beforeHistory := command(nil, "config", "history", "--db", b.db)
	if beforeHistory.err != nil {
		t.Fatal("config history before backup failed")
	}
	before := decode[core.Task](t, coordinator.ok(t, "get", payload{"task_key": task.Key}))
	events := decode[core.EventsPage](t, coordinator.ok(t, "events", payload{"since_cursor": 0, "limit": 200}))
	if wal, err := os.Stat(b.db + "-wal"); err != nil || wal.Size() == 0 {
		t.Fatal("online backup fixture did not contain a live nonempty WAL")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(dir, "snapshot.db")
	backup := command(nil, "backup", "--db", b.db, "--output", snapshot)
	if backup.err != nil {
		t.Fatal("CLI backup failed while real server held live WAL")
	}
	metadata := decode[core.BackupResult](t, backup.data)
	data, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.SchemaVersion != expectedCurrentSchema || metadata.CreatedAt.IsZero() || metadata.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("snapshot checksum/schema metadata does not describe published file")
	}
	if info, err := os.Stat(snapshot); err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("snapshot is not private0600")
	}
	if duplicate := command(nil, "backup", "--db", b.db, "--output", snapshot); duplicate.err == nil {
		t.Fatal("backup overwrote existing snapshot")
	}
	if unchanged, err := os.ReadFile(snapshot); err != nil || !bytes.Equal(data, unchanged) {
		t.Fatal("rejected overwrite changed original snapshot")
	}
	// Revoke after snapshot: recovery must not resurrect its older active token.
	if revoke := command(nil, "token", "revoke", "--db", b.db, "--id", worker.tokenID); revoke.err != nil {
		t.Fatal("post-snapshot revocation failed")
	}
	b.stop()
	restored := newBoard(t, "30s")
	if err = os.WriteFile(restored.db, data, 0600); err != nil {
		t.Fatal(err)
	}
	oldCoordinator := &client{b: restored, token: coordinator.token, session: coordinator.session}
	oldWorker := &client{b: restored, token: worker.token, session: worker.session}
	// This private loopback fixture first proves exact snapshot contents; it is
	// not the deployment procedure, which must reset before exposing the copy.
	restored.start()
	restoredConfig := configView(t, restored)
	restoredHistory := command(nil, "config", "history", "--db", restored.db)
	if restoredConfig.SHA256 != beforeConfig.SHA256 || restoredConfig.ConfigVersion != beforeConfig.ConfigVersion || restoredHistory.err != nil || !bytes.Equal(beforeHistory.data, restoredHistory.data) {
		t.Fatal("snapshot omitted config/history")
	}
	copyTask := decode[core.Task](t, oldCoordinator.ok(t, "get", payload{"task_key": task.Key}))
	copyEvents := decode[core.EventsPage](t, oldCoordinator.ok(t, "events", payload{"since_cursor": 0, "limit": 200}))
	copyNotes := decode[core.NotesPage](t, oldCoordinator.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer"}))
	if !reflect.DeepEqual(before, copyTask) || !reflect.DeepEqual(events, copyEvents) || !reflect.DeepEqual(beforeNotes, copyNotes) {
		t.Fatal("WAL snapshot omitted task/lease/history/comment/events/schema4 notes")
	}
	replay := oldCoordinator.raw("create", input)
	if replay.err != nil || !bytes.Equal(created.data, replay.data) {
		t.Fatal("snapshot lost persisted idempotency response")
	}
	restored.stop() // No process writes the restored database during reset.
	reset := command(nil, "maintenance", "reset", "--db", restored.db, "--reason", "restore")
	if reset.err != nil {
		t.Fatal("offline recovery reset failed")
	}
	counts := decode[core.RecoveryResult](t, reset.data)
	if counts.RevokedTokens != 2 || counts.Sessions != 2 || counts.Leases != 1 {
		t.Fatal("offline reset did not invalidate all snapshot credentials/sessions/leases")
	}
	resetConfig := configView(t, restored)
	resetHistory := command(nil, "config", "history", "--db", restored.db)
	if resetConfig.SHA256 != beforeConfig.SHA256 || resetConfig.ConfigVersion != beforeConfig.ConfigVersion || resetHistory.err != nil || !bytes.Equal(beforeHistory.data, resetHistory.data) {
		t.Fatal("reset changed config/history")
	}
	freshCoordinator := newClient(t, restored, "new-coordinator")
	freshWorker := newClient(t, restored, "new-worker")
	restored.start()
	oldCoordinator.fails(t, "create", "UNAUTHORIZED", payload{"type": "epic", "title": "Forbidden", "description": "old snapshot identity"})
	oldWorker.fails(t, "heartbeat", "UNAUTHORIZED", payload{"task_key": task.Key, "generation": claim.Lease.Generation})
	freshWorker.session = worker.session
	freshWorker.fails(t, "heartbeat", "UNAUTHORIZED", payload{"task_key": task.Key, "generation": claim.Lease.Generation})
	freshCoordinator.open(t, "coordinator", "new-coordinator")
	freshWorker.session = ""
	freshWorker.open(t, "developer", "new-worker")
	resetNotes := decode[core.NotesPage](t, freshWorker.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer"}))
	if len(resetNotes.Notes) != 1 || !reflect.DeepEqual(resetNotes.Notes, beforeNotes.Notes) {
		t.Fatal("schema4 recovery reset discarded note/provenance")
	}
	recovered := decode[core.Task](t, freshCoordinator.ok(t, "get", payload{"task_key": task.Key}))
	if recovered.Lease != nil || recovered.Generation != claim.Lease.Generation || len(recovered.Comments) != 1 || len(recovered.Attempts) != 1 || recovered.Attempts[0].Reason != "restore" || recovered.Attempts[0].EndedAt == nil {
		t.Fatal("reset discarded history, reset fencing or retained live ownership")
	}
	fresh := freshWorker.claim(t, task.Key)
	if fresh.Lease.Generation != claim.Lease.Generation+1 || len(fresh.Task.Attempts) != 2 {
		t.Fatal("new identity did not advance restored fencing monotonically")
	}
	audit := decode[core.EventsPage](t, freshCoordinator.ok(t, "events", payload{"since_cursor": events.HighWater, "limit": 200, "types": []string{"database_recovered", "tokens_revoked_all", "lease_recovered"}}))
	if len(audit.Events) != 3 {
		t.Fatal("recovery/revocation/lease audit missing from restored copy")
	}
	t.Log("real live-WAL CLI snapshot retained exact task/events/replay; offline reset denied old token/session and advanced fencing")
}
