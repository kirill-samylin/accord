//go:build unix

package store

import (
	"agent-board/internal/core"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDatabasePrivateBeforeSQLiteIOInPublicDirectory(t *testing.T) {
	old := syscall.Umask(0022)
	defer syscall.Umask(old)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Special URI characters must remain filename characters after preparation.
	path := filepath.Join(dir, "board?#.db")
	s, err := Open(path, core.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	token, err := s.CreateToken(testContext, core.TokenInput{Owner: "synthetic", Machine: "test", TTL: 60000000000})
	if err != nil || token.ID == "" {
		t.Fatal("failed synthetic token insertion")
	}
	checkModes := func() {
		t.Helper()
		for _, suffix := range []string{"", "-wal", "-shm"} {
			info, statErr := os.Lstat(path + suffix)
			if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
				t.Fatalf("database component %q is not private", suffix)
			}
		}
	}
	checkModes()
	// A second connection exercises startup with existing live WAL and SHM.
	reopened, err := Open(path, core.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = reopened.Close(); err != nil {
		t.Fatal(err)
	}
	checkModes()
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path, core.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkModes()
}

func TestDatabaseRejectsUnsafeExistingFilesWithoutChangingTargets(t *testing.T) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		for _, kind := range []string{"public", "symlink", "dangling"} {
			t.Run(suffix+"/"+kind, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "board.db")
				foreign := filepath.Join(dir, "foreign")
				payload := []byte("synthetic foreign data")
				if kind != "dangling" {
					if err := os.WriteFile(foreign, payload, 0644); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "public" {
					if err := os.WriteFile(path+suffix, payload, 0644); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(path+suffix, 0644); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Symlink(foreign, path+suffix); err != nil {
					t.Fatal(err)
				}
				opened, err := Open(path, core.Config{}, nil)
				if err == nil {
					opened.Close()
					t.Fatal("unsafe path was accepted")
				}
				if kind == "dangling" {
					if _, err := os.Lstat(foreign); !os.IsNotExist(err) {
						t.Fatal("dangling target was created")
					}
					return
				}
				content, err := os.ReadFile(foreign)
				info, statErr := os.Lstat(foreign)
				if err != nil || statErr != nil || string(content) != string(payload) || info.Mode().Perm() != 0644 {
					t.Fatal("foreign target was changed")
				}
				content, err = os.ReadFile(path + suffix)
				if err != nil || string(content) != string(payload) {
					t.Fatal("unsafe existing file was changed")
				}
			})
		}
	}
}

func TestDatabaseRejectsPubliclyWritableDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(filepath.Join(dir, "board.db"), core.Config{}, nil); err == nil {
		opened.Close()
		t.Fatal("publicly writable database directory accepted")
	}
}

func TestDatabaseRejectsSQLiteURIWithoutChangingItsTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.db")
	data := []byte("synthetic foreign file")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open("file:"+path+"?mode=rwc", core.Config{}, nil); err == nil {
		opened.Close()
		t.Fatal("SQLite URI unexpectedly accepted")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != string(data) {
		t.Fatal("URI target was modified")
	}
}
