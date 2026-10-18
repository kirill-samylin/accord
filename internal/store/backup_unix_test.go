//go:build !windows

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFilesystemGuards(t *testing.T) {
	f := newFixture(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(dir, "source-link.db")
	if err := os.Symlink(f.path, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := Backup(testContext, symlink, filepath.Join(dir, "copy.db")); err == nil {
		t.Fatal("backup followed source symlink")
	}
	outputLink := filepath.Join(dir, "output-link.db")
	if err := os.Symlink(f.path, outputLink); err != nil {
		t.Fatal(err)
	}
	if _, err := Backup(testContext, f.path, outputLink); err == nil {
		t.Fatal("backup followed output symlink")
	}
	output := filepath.Join(dir, "blocked.db")
	if err := os.WriteFile(output+"-journal", []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Backup(testContext, f.path, output); err == nil {
		t.Fatal("backup accepted existing output sidecar")
	}
	if data, err := os.ReadFile(output + "-journal"); err != nil || string(data) != "preserve" {
		t.Fatal("backup changed preexisting output sidecar")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Backup(testContext, f.path, filepath.Join(dir, "readable.db")); err == nil {
		t.Fatal("backup accepted readable output directory")
	}
	if err := os.Chmod(f.path, 0644); err != nil {
		t.Fatal(err)
	}
	private := t.TempDir()
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Backup(testContext, f.path, filepath.Join(private, "copy.db")); err == nil {
		t.Fatal("backup accepted public source")
	}
	if info, err := os.Stat(f.path); err != nil || info.Mode().Perm() != 0644 {
		t.Fatal("backup repaired rejected source permissions")
	}
}

func TestFailedVacuumRemovesOwnOutput(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.db.Exec(`PRAGMA writable_schema=ON`); err != nil {
		t.Fatal(err)
	}
	// Corrupt only this synthetic schema so failure occurs inside VACUUM, after
	// the private destination has been reserved, rather than during path checks.
	if _, err := f.s.db.Exec(`UPDATE sqlite_schema SET sql='CREATE TABLE tasks(' WHERE name='tasks'`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "failed.db")
	if _, err := Backup(testContext, f.path, output); err == nil {
		t.Fatal("backup of malformed schema unexpectedly succeeded")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed backup left a partial output")
	}
	if files, err := os.ReadDir(dir); err != nil || len(files) != 0 {
		t.Fatal("failed backup left a private staging file", err)
	}
	if _, err := os.Stat(f.path); err != nil {
		t.Fatal("backup removed source database", err)
	}
}
