package store

import (
	"agent-board/internal/core"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBackupOnlineWALAndRecovery(t *testing.T) {
	f := newFixture(t)
	worker := f.actor(t, "developer")
	task := f.task(t, "developer")
	f.claim(t, worker, task.Key)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "restore-user", Machine: "restore-pc", TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(f.path + "-wal"); err != nil {
		t.Fatal("test must exercise an active WAL", err)
	}
	// All writes use independent random IDs; snapshot must contain each token
	// atomically with its audit even while another connection commits writes.
	var wg sync.WaitGroup
	writerErrors := make(chan error, 1)
	started := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		for i := 0; i < 20; i++ {
			if _, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "concurrent", Machine: "writer", TTL: time.Hour}); err != nil {
				writerErrors <- err
				return
			}
		}
	}()
	<-started
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "snapshot.db")
	metadata, backupErr := Backup(testContext, f.path, target)
	wg.Wait()
	close(writerErrors)
	for err := range writerErrors {
		t.Fatal(err)
	}
	if backupErr != nil {
		t.Fatal(backupErr)
	}
	if metadata.SchemaVersion != SchemaVersion || metadata.CreatedAt.IsZero() {
		t.Fatalf("backup metadata: %+v", metadata)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != metadata.SHA256 {
		t.Fatal("snapshot checksum mismatch")
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("snapshot is not private", err)
	}
	if err = f.s.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal(err)
	}
	restored, err := OpenExisting(target, core.Config{BootstrapUsers: []string{"alice"}}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	var tokens, creates int
	if err = restored.db.QueryRow(`SELECT COUNT(*) FROM tokens`).Scan(&tokens); err != nil {
		t.Fatal(err)
	}
	if err = restored.db.QueryRow(`SELECT COUNT(*) FROM events WHERE json_extract(data,'$.type')='token_created'`).Scan(&creates); err != nil || creates != tokens {
		t.Fatalf("inconsistent snapshot tokens=%d audits=%d err=%v", tokens, creates, err)
	}
	if _, err = restored.Authenticate(testContext, token.Secret); err != nil {
		t.Fatal("snapshot did not preserve credential state before later revoke", err)
	}
	if _, err = restored.RecoverState(testContext, "restore"); err != nil {
		t.Fatal(err)
	}
	if err = restored.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err = OpenExisting(target, core.Config{BootstrapUsers: []string{"alice"}}, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	_, err = restored.Authenticate(testContext, token.Secret)
	requireCode(t, err, "UNAUTHORIZED")
}

func TestBackupRejectsUnsafePathsWithoutMutation(t *testing.T) {
	f := newFixture(t)
	private := t.TempDir()
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(private, "existing.db")
	if err := os.WriteFile(old, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(private, "missing.db")
	cases := []struct{ source, output string }{
		{missing, filepath.Join(private, "new.db")},
		{"file:" + f.path, filepath.Join(private, "uri-source.db")},
		{f.path, "file:" + filepath.Join(private, "uri-output.db")},
		{f.path, old},
		{f.path, ":memory:"},
	}
	for _, tc := range cases {
		if _, err := Backup(testContext, tc.source, tc.output); err == nil {
			t.Fatalf("accepted unsafe backup %q %q", tc.source, tc.output)
		}
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("missing source was created")
	}
	if data, err := os.ReadFile(old); err != nil || string(data) != "keep" {
		t.Fatal("existing output overwritten", err)
	}
	if _, err := OpenExisting(missing, core.Config{BootstrapUsers: []string{"alice"}}, nil); err == nil {
		t.Fatal("OpenExisting created missing source")
	}
	ctx, cancel := context.WithCancel(testContext)
	cancel()
	cancelled := filepath.Join(private, "cancelled.db")
	if _, err := Backup(ctx, f.path, cancelled); err == nil {
		t.Fatal("cancelled backup succeeded")
	}
	if _, err := os.Stat(cancelled); !os.IsNotExist(err) {
		t.Fatal("failed backup left output")
	}
	unsafe := filepath.Join(private, "unsupported.db")
	if err := os.WriteFile(unsafe, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(unsafe, core.Config{BootstrapUsers: []string{"alice"}}, nil); err == nil {
		t.Fatal("empty file silently bootstrapped by admin")
	}
	if info, _ := os.Stat(unsafe); info.Size() != 0 {
		t.Fatal("empty source changed on rejected admin open")
	}
	// A filename containing URI punctuation is a literal path, not query flags.
	literal := filepath.Join(private, "snapshot?mode=memory#literal.db")
	if _, err := Backup(testContext, f.path, literal); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(private)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "-wal") || strings.HasSuffix(file.Name(), "-shm") {
			t.Fatalf("snapshot left temporary sidecar %s", file.Name())
		}
	}
}

func TestInterruptedBackupNeverPublishesPartialSnapshot(t *testing.T) {
	f := newFixture(t)
	// A bounded synthetic payload keeps VACUUM busy long enough to interrupt
	// after its private output exists; no production database is involved.
	if _, err := f.s.db.Exec(`CREATE TABLE payload(data BLOB); INSERT INTO payload VALUES(zeroblob(67108864))`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "cancelled.db")
	ctx, cancel := context.WithCancel(testContext)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Backup(ctx, f.path, target)
		done <- err
	}()
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	observedStage := false
	for !observedStage {
		select {
		case err := <-done:
			t.Fatalf("backup finished before interruption: %v", err)
		case <-deadline:
			t.Fatal("backup did not create a private output within deadline")
		case <-tick.C:
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				observedStage = observedStage || strings.HasPrefix(file.Name(), ".accord-backup-")
			}
		}
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("partial output was published before validation")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("interrupted backup succeeded")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelled backup did not stop")
	}
	if files, err := os.ReadDir(dir); err != nil || len(files) != 0 {
		t.Fatal("interrupted backup left final or staging files", err)
	}
}
