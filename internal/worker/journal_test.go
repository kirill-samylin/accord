package worker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJournalLockAndAmbiguousRestart(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private-state")
	d, j, err := openDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	oldID := j.WorkerID
	if second, _, err := openDisk(dir); err == nil {
		second.close()
		t.Fatal("allowed two owners of one state")
	}
	j.HostID = "synthetic-host"
	j.Runs["run_test"] = &RunState{ID: "run_test", State: "launching", Provider: "codex", Workspace: "test"}
	if err = d.save(j); err != nil {
		t.Fatal(err)
	}
	d.close()
	d, j, err = openDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	if j.WorkerID == oldID {
		t.Fatal("reused process incarnation after restart")
	}
	if j.Runs["run_test"].State != "needs_recovery" {
		t.Fatal("ambiguous launch was not quarantined")
	}
	if j.HostID != "synthetic-host" {
		t.Fatal("lost host enrollment")
	}
	if len(j.HostSecret) != 64 {
		t.Fatal("host proof has unexpected entropy")
	}
}

func TestIncompleteJournalCannotSilentlyRecover(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private-state")
	d, j, err := openDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "state.next"), []byte(`{"incomplete":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = d.save(j); err == nil || !strings.Contains(err.Error(), "recovery") {
		t.Fatal("partial journal overwritten")
	}
	d.close()
	if reopened, _, err := openDisk(dir); err == nil {
		reopened.close()
		t.Fatal("started with incomplete durable write")
	}
}

func TestPrivateTokenAndConfigRejectUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("synthetic-token"), 0600); err != nil {
		t.Fatal(err)
	}
	if token, err := ReadToken(path); err != nil || token != "synthetic-token" {
		t.Fatal("private token unavailable")
	}
	if err := os.WriteFile(path, []byte("synthetic\ninvalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadToken(path); err == nil {
		t.Fatal("accepted multiline credential")
	}
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, []byte(`{"name":"one","name":"two"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(config); err == nil {
		t.Fatal("accepted duplicate configuration keys")
	}
}
