package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Regressions from the independent QA of the admin section (25.10.2026).

func TestPersonFieldsRejectInvalidUTF8(t *testing.T) {
	for _, bad := range []string{"qa\xff\xfe", "\xc3\x28"} {
		if validPersonName(bad) || validPersonNote(bad) {
			t.Fatalf("invalid UTF-8 accepted: %q", bad)
		}
	}
	if !validPersonName("Тестовый Человек") || !validPersonNote("заметка") {
		t.Fatal("valid UTF-8 rejected")
	}
}

func TestUniquePersonNameTerminatesForLongCollidingOwners(t *testing.T) {
	taken := map[string]bool{}
	owner, machine := strings.Repeat("x", 70), "m"
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		done := make(chan string, 1)
		go func() { done <- uniquePersonName(owner, machine, taken) }()
		select {
		case name := <-done:
			if utf8.RuneCountInString(name) > MaxPersonName || !validPersonName(name) || seen[name] {
				t.Fatalf("bad name %q", name)
			}
			seen[name] = true
		case <-time.After(2 * time.Second):
			t.Fatal("uniquePersonName did not terminate")
		}
	}
	if name := uniquePersonName(" spaced owner ", "pc", map[string]bool{}); name != "spaced owner" {
		t.Fatalf("owner label not trimmed: %q", name)
	}
}

func TestMigrationCopyRefusesForeignFileAndDropsStaleStage(t *testing.T) {
	f := setup(t)
	root := t.TempDir()
	dir := filepath.Join(root, "registry.sqlite.migrations")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	stale := filepath.Join(dir, ".registry-copy-123.db")
	if e := os.WriteFile(stale, []byte("partial"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := registryMigrationCopy(root, f.r.db, 2); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Lstat(stale); !os.IsNotExist(e) {
		t.Fatal("stale staging file kept")
	}
	copies, _ := filepath.Glob(filepath.Join(dir, "registry2-*.db"))
	if len(copies) != 1 {
		t.Fatalf("copies %v", copies)
	}
	// Same content again is accepted; foreign bytes under the name are not.
	if e := registryMigrationCopy(root, f.r.db, 2); e != nil {
		t.Fatal("identical copy refused", e)
	}
	if e := os.WriteFile(copies[0], []byte("garbage"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := registryMigrationCopy(root, f.r.db, 2); e == nil {
		t.Fatal("foreign file accepted as migration copy")
	}
}

func TestAdminCannotRevokeOrRelinkOwnOrLastAdminCredential(t *testing.T) {
	f := setup(t)
	_, other, e := f.r.CreateCredential(context.Background(), "B", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	admin := f.makeAdmin(t)
	page := f.admin(t, admin)
	self := page.Page.Registry.Identity.ID
	var member Person
	for _, p := range page.Page.People {
		if p.Role == "member" {
			member = p
		}
	}
	if code, _ := f.post(t, admin, page, "admin:identity:"+self+":revoke", "/web/catalog/admin/identities/"+self, url.Values{"action": {"revoke"}}); code != 403 {
		t.Fatalf("own credential revoke %d", code)
	}
	page = f.admin(t, admin)
	if code, _ := f.post(t, admin, page, "admin:identity:"+self+":link", "/web/catalog/admin/identities/"+self, url.Values{"action": {"link"}, "person_id": {member.ID}}); code != 403 {
		t.Fatalf("own credential relink %d", code)
	}
	// A second admin person whose only credential is `other`: revoking it is
	// fine while the first admin keeps a credential.
	page = f.admin(t, admin)
	_, second := f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Второй"}, "role": {"admin"}})
	page = f.admin(t, admin)
	if code, _ := f.post(t, admin, page, "admin:identity:"+other.ID+":link", "/web/catalog/admin/identities/"+other.ID, url.Values{"action": {"link"}, "person_id": {second.Person.ID}}); code != 200 {
		t.Fatalf("link to second admin %d", code)
	}
	page = f.admin(t, admin)
	if code, _ := f.post(t, admin, page, "admin:identity:"+other.ID+":revoke", "/web/catalog/admin/identities/"+other.ID, url.Values{"action": {"revoke"}}); code != 200 {
		t.Fatalf("revoke another admin's credential %d", code)
	}
	// With `other` revoked, the acting admin's own credential is the last
	// usable admin access: the guard refuses to count anything else.
	_, err := transaction(f.r, context.Background(), func(tx *sql.Tx) (bool, error) {
		return false, f.r.requireAdminCredentialBesides(context.Background(), tx, self)
	})
	if err == nil {
		t.Fatal("last admin credential not protected")
	}
}

func TestNoopActivationKeepsVersionAndAudit(t *testing.T) {
	f := setup(t)
	admin := f.makeAdmin(t)
	page := f.admin(t, admin)
	code, created := f.post(t, admin, page, "admin:person:create", "/web/catalog/admin/people", url.Values{"name": {"Человек"}, "role": {"member"}})
	if code != 200 {
		t.Fatal(code)
	}
	before := len(f.admin(t, admin).Page.Audit)
	page = f.admin(t, admin)
	path := "/web/catalog/admin/people/" + created.Person.ID
	code, again := f.post(t, admin, page, "admin:person:"+created.Person.ID+":update", path, url.Values{"action": {"activate"}, "version": {strconv.FormatInt(created.Person.Version, 10)}})
	if code != 200 || again.Person.Version != created.Person.Version {
		t.Fatalf("no-op activate %d %+v", code, again.Person)
	}
	if after := len(f.admin(t, admin).Page.Audit); after != before {
		t.Fatalf("no-op activate audited: %d → %d", before, after)
	}
}

func TestNonMemberAdminCannotArchiveForeignProject(t *testing.T) {
	f := setup(t)
	_, owner, e := f.r.CreateCredential(context.Background(), "B", "machine", time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	foreign := f.create(t, owner, "foreign")
	admin := f.makeAdmin(t)
	out := f.request("GET", "/web/catalog/projects/"+foreign.ID, admin, nil)
	var page struct {
		Page struct {
			Registry struct {
				CSRF    string            `json:"csrf"`
				Actions map[string]string `json:"actions"`
			} `json:"registry"`
			Project Project `json:"project"`
		} `json:"page"`
	}
	json.Unmarshal(out.Body.Bytes(), &page)
	if page.Page.Project.CanArchive {
		t.Fatal("non-member admin may archive")
	}
	update := f.request("POST", "/web/catalog/projects/"+foreign.ID+"/update", admin, url.Values{"csrf": {page.Page.Registry.CSRF}, "action_id": {page.Page.Registry.Actions["update"]}, "name": {"foreign"}, "state": {"archived"}, "version": {strconv.FormatInt(page.Page.Project.Version, 10)}})
	if update.Code == 200 {
		t.Fatalf("non-member admin archived a project: %s", update.Body.String())
	}
}
