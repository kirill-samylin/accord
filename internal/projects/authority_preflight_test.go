package projects

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivatedRegistryLossNeverInitializesNewAuthority(t *testing.T) {
	for _, kind := range []string{"missing", "empty", "no-meta", "corrupt", "missing-memberships", "missing-marker"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t)
			root := f.r.options.DataDir
			options := f.r.options
			options.Offline = true
			if _, e := f.r.db.Exec(`UPDATE memberships SET active=0,access_epoch=access_epoch+1 WHERE project_id=? AND identity_id=?`, f.r.defaultID, f.who.ID); e != nil {
				t.Fatal(e)
			}
			f.r.Close()
			f.browser.Close()
			path := filepath.Join(root, "registry.sqlite")
			switch kind {
			case "missing":
				if e := os.Remove(path); e != nil {
					t.Fatal(e)
				}
			case "empty":
				if e := os.Truncate(path, 0); e != nil {
					t.Fatal(e)
				}
			case "corrupt":
				if e := os.WriteFile(path, []byte("corrupt authority bytes"), 0600); e != nil {
					t.Fatal(e)
				}
			case "missing-marker":
				if e := os.Remove(filepath.Join(root, "MULTIPROJECT_REQUIRES_CAPABILITY")); e != nil {
					t.Fatal(e)
				}
			default:
				db, e := sql.Open("sqlite", path)
				if e != nil {
					t.Fatal(e)
				}
				statement := `DELETE FROM registry_meta`
				if kind == "missing-memberships" {
					statement = `DROP TABLE memberships`
				}
				if _, e = db.Exec(statement); e != nil {
					t.Fatal(e)
				}
				db.Close()
			}
			before, _ := os.ReadFile(path)
			if reopened, e := Open(options); e == nil {
				reopened.Close()
				t.Fatal("lost authority was recreated")
			}
			after, e := os.ReadFile(path)
			if kind == "missing" {
				if !os.IsNotExist(e) {
					t.Fatal("missing registry recreated")
				}
			} else if e != nil || string(after) != string(before) {
				t.Fatal("lost authority was mutated", e)
			}
			if kind == "missing-memberships" {
				db, e := sql.Open("sqlite", path)
				if e != nil {
					t.Fatal(e)
				}
				defer db.Close()
				var count int
				if e = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='memberships'`).Scan(&count); e != nil || count != 0 {
					t.Fatal("membership table silently repaired", e)
				}
			}
		})
	}
}
func TestMarkerOnlyInterruptedActivationRefusesWithoutCreatingDatabase(t *testing.T) {
	f := setup(t)
	root := filepath.Join(f.root, "interrupted")
	if _, e := privateRoot(root); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "MULTIPROJECT_REQUIRES_CAPABILITY"), []byte("registry=1 acl=1\n"), 0600); e != nil {
		t.Fatal(e)
	}
	options := f.r.options
	options.DataDir = root
	options.Offline = true
	if reopened, e := Open(options); e == nil {
		reopened.Close()
		t.Fatal("interrupted activation resumed with guessed authority")
	}
	if _, e := os.Lstat(filepath.Join(root, "registry.sqlite")); !os.IsNotExist(e) {
		t.Fatal("interrupted activation created DB", e)
	}
}
func TestValidAuthorityReopenPreservesRevokedDefaultMembership(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, e := f.r.db.Exec(`UPDATE memberships SET active=0,access_epoch=access_epoch+1 WHERE project_id=? AND identity_id=?`, f.r.defaultID, f.who.ID); e != nil {
		t.Fatal(e)
	}
	options := f.r.options
	options.Offline = true
	id := f.r.defaultID
	f.r.Close()
	f.browser.Close()
	reopened, e := Open(options)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if reopened.defaultID != id {
		t.Fatal("default identity changed")
	}
	who, e := reopened.Authenticate(ctx, f.token.Secret)
	if e != nil {
		t.Fatal(e)
	}
	list, e := reopened.List(ctx, who)
	if e != nil || len(list) != 0 {
		t.Fatal("revoked default grant resurrected", e)
	}
}
func TestPublishedStorageLossRefusesBeforeRuntimeOrphanRecovery(t *testing.T) {
	for _, kind := range []string{"web-missing", "board-missing", "design-table-missing"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t)
			p := f.create(t, f.who, "published")
			v, e := f.r.runtime(p.ID)
			if e != nil {
				t.Fatal(e)
			}
			board, dir, e := f.r.projectPaths(p.ID)
			if e != nil {
				t.Fatal(e)
			}
			blob := filepath.Join(dir, "blobs", strings.Repeat("a", 48))
			if e = os.WriteFile(blob, []byte("old design must survive"), 0600); e != nil {
				t.Fatal(e)
			}
			v.browser.Close()
			v.store.Close()
			f.r.mu.Lock()
			delete(f.r.runtimes, p.ID)
			f.r.mu.Unlock()
			webDB := filepath.Join(dir, "web.db")
			switch kind {
			case "web-missing":
				e = os.Remove(webDB)
			case "board-missing":
				e = os.Remove(board)
			default:
				db, err := sql.Open("sqlite", webDB)
				if err != nil {
					t.Fatal(err)
				}
				_, e = db.Exec(`DROP TABLE design_assets`)
				db.Close()
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = f.r.runtime(p.ID); e == nil {
				t.Fatal("published storage reconstructed")
			}
			if e = Preflight(f.r.options.DataDir, f.db, f.webDir); e == nil {
				t.Fatal("startup preflight accepted published storage loss")
			}
			bytes, e := os.ReadFile(blob)
			if e != nil || string(bytes) != "old design must survive" {
				t.Fatal("orphan recovery deleted design after metadata loss", e)
			}
			if kind == "web-missing" {
				if _, e = os.Lstat(webDB); !os.IsNotExist(e) {
					t.Fatal("missing web DB recreated", e)
				}
			}
			if kind == "board-missing" {
				if _, e = os.Lstat(board); !os.IsNotExist(e) {
					t.Fatal("missing board DB recreated", e)
				}
			}
		})
	}
}
