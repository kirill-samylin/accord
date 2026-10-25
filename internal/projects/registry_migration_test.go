package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/web"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// schema1 is the historical registry DDL (TASK009) used to build a real
// pre-migration authority; it must stay byte-compatible with old deployments.
const schema1 = `
CREATE TABLE registry_meta(id INTEGER PRIMARY KEY CHECK(id=1),default_id TEXT NOT NULL,default_db TEXT NOT NULL,default_web TEXT NOT NULL,signing_key TEXT NOT NULL);
CREATE TABLE rejected_provisions(id TEXT PRIMARY KEY,at INTEGER NOT NULL);
CREATE TABLE identities(id TEXT PRIMARY KEY,source TEXT NOT NULL,token_id TEXT NOT NULL,owner TEXT NOT NULL,machine TEXT NOT NULL);
CREATE TABLE credentials(identity_id TEXT PRIMARY KEY,hash TEXT UNIQUE NOT NULL,expires INTEGER NOT NULL,revoked INTEGER NOT NULL DEFAULT 0);
CREATE TABLE projects(id TEXT PRIMARY KEY,name TEXT NOT NULL,state TEXT NOT NULL,version INTEGER NOT NULL,creator TEXT NOT NULL,created INTEGER NOT NULL);
CREATE TABLE memberships(project_id TEXT NOT NULL,identity_id TEXT NOT NULL,role TEXT NOT NULL,local_id TEXT NOT NULL,active INTEGER NOT NULL,version INTEGER NOT NULL,access_epoch INTEGER NOT NULL,PRIMARY KEY(project_id,identity_id));
CREATE TABLE repositories(id TEXT PRIMARY KEY,project_id TEXT NOT NULL,url TEXT NOT NULL,branch TEXT NOT NULL,version INTEGER NOT NULL);
CREATE TABLE sessions(hash TEXT PRIMARY KEY,identity_id TEXT NOT NULL,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE session_tombstones(hash TEXT PRIMARY KEY,expires INTEGER NOT NULL);
CREATE TABLE session_epochs(project_id TEXT NOT NULL,session_id TEXT NOT NULL,identity_id TEXT NOT NULL,access_epoch INTEGER NOT NULL,PRIMARY KEY(project_id,session_id));
CREATE TABLE preauth(hash TEXT PRIMARY KEY,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE actions(id TEXT PRIMARY KEY,session_hash TEXT NOT NULL,operation TEXT NOT NULL,project_id TEXT NOT NULL,access_epoch INTEGER NOT NULL,payload TEXT NOT NULL DEFAULT '',result TEXT NOT NULL DEFAULT '',created INTEGER NOT NULL);
CREATE TABLE audit(id INTEGER PRIMARY KEY,event TEXT NOT NULL,identity_id TEXT NOT NULL,project_id TEXT NOT NULL,at INTEGER NOT NULL);
PRAGMA user_version=1;
`

func TestRegistrySchema1MigratesToPeopleWithVerifiedCopy(t *testing.T) {
	root := t.TempDir()
	os.Chmod(root, 0700)
	db, webDir := filepath.Join(root, "board.db"), filepath.Join(root, "web")
	cfg := core.Config{Project: "legacy", LeaseTTL: time.Minute, SessionTTL: time.Minute}
	s, e := store.Open(db, cfg, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	browser, e := web.New(s, web.Options{Origin: "http://127.0.0.1:8080", DataDir: webDir})
	if e != nil {
		t.Fatal(e)
	}
	defer browser.Close()
	token, e := s.CreateToken(context.Background(), core.TokenInput{Owner: "operator", Machine: "machine", TTL: time.Hour})
	if e != nil {
		t.Fatal(e)
	}
	registryRoot := filepath.Join(root, "registry")
	os.Mkdir(registryRoot, 0700)
	os.WriteFile(filepath.Join(registryRoot, "MULTIPROJECT_REQUIRES_CAPABILITY"), []byte("registry=1 acl=1\n"), 0600)
	path := filepath.Join(registryRoot, "registry.sqlite")
	os.WriteFile(path, nil, 0600)
	old, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defaultID, key, secret := "p_"+strings.Repeat("0", 32), strings.Repeat("ab", 32), "ac_"+strings.Repeat("cd", 32)
	legacy := "legacy_" + token.ID
	statements := []string{schema1,
		`INSERT INTO registry_meta VALUES(1,'` + defaultID + `','` + db + `','` + webDir + `','` + key + `')`,
		`INSERT INTO projects VALUES('` + defaultID + `','legacy','active',1,'',1)`,
		`INSERT INTO identities VALUES('i_` + strings.Repeat("0", 32) + `','catalog','','operator','laptop')`,
		`INSERT INTO identities VALUES('i_` + strings.Repeat("1", 32) + `','catalog','','B','m')`,
		`INSERT INTO identities VALUES('i_` + strings.Repeat("2", 32) + `','catalog','','operator','laptop')`,
		`INSERT INTO identities VALUES('` + legacy + `','legacy','` + token.ID + `','operator','machine')`,
		`INSERT INTO credentials VALUES('i_` + strings.Repeat("1", 32) + `','` + hash(secret) + `',` + "9223372036854775807" + `,0)`,
		`INSERT INTO memberships VALUES('` + defaultID + `','` + legacy + `','owner','` + token.ID + `',1,2,0)`,
		`INSERT INTO memberships VALUES('` + defaultID + `','i_` + strings.Repeat("1", 32) + `','viewer','local',1,1,1)`,
	}
	for _, q := range statements {
		if _, e = old.Exec(q); e != nil {
			t.Fatal(q, e)
		}
	}
	old.Close()
	open := func() *Registry {
		r, e := Open(Options{DataDir: registryRoot, DefaultDB: db, DefaultWeb: webDir, Default: s, Browser: browser, Origin: "http://127.0.0.1:8080", Config: cfg})
		if e != nil {
			t.Fatal("open schema1 authority", e)
		}
		return r
	}
	r := open()
	var version, people, bound, empty int
	r.db.QueryRow(`PRAGMA user_version`).Scan(&version)
	r.db.QueryRow(`SELECT COUNT(*) FROM people`).Scan(&people)
	r.db.QueryRow(`SELECT COUNT(DISTINCT person_id) FROM identities`).Scan(&bound)
	r.db.QueryRow(`SELECT COUNT(*) FROM identities WHERE person_id=''`).Scan(&empty)
	if version != 2 || people != 4 || bound != 4 || empty != 0 {
		t.Fatalf("bootstrap version=%d people=%d bound=%d unbound=%d", version, people, bound, empty)
	}
	roles := map[string]string{}
	rows, _ := r.db.Query(`SELECT i.id,p.name,p.role FROM identities i JOIN people p ON p.id=i.person_id ORDER BY i.id`)
	names := []string{}
	for rows.Next() {
		var id, name, role string
		rows.Scan(&id, &name, &role)
		roles[id] = role
		names = append(names, name)
	}
	rows.Close()
	if strings.Join(names, "|") != "operator|B|operator (laptop)|operator (machine)" {
		t.Fatalf("bootstrap names %q", names)
	}
	if roles[legacy] != "admin" || roles["i_"+strings.Repeat("1", 32)] != "member" || roles["i_"+strings.Repeat("0", 32)] != "member" {
		t.Fatalf("default owner admin bootstrap %v", roles)
	}
	var storedHash string
	if r.db.QueryRow(`SELECT hash FROM credentials WHERE identity_id=?`, "i_"+strings.Repeat("1", 32)).Scan(&storedHash); storedHash != hash(secret) {
		t.Fatal("pre-existing credential changed")
	}
	who, e := r.Authenticate(context.Background(), token.Secret)
	if e != nil || !who.Admin {
		t.Fatal("legacy default owner must be admin", e, who)
	}
	if who, e = r.Authenticate(context.Background(), secret); e != nil || who.Admin {
		t.Fatal("viewer credential must stay member", e, who)
	}
	r.Close()
	dir := filepath.Join(registryRoot, "registry.sqlite.migrations")
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("migration copy directory", e, info)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || !regexp.MustCompile(`^registry1-[0-9a-f]{64}\.db$`).MatchString(entries[0].Name()) {
		t.Fatalf("migration copy entries %v", entries)
	}
	copyPath := filepath.Join(dir, entries[0].Name())
	copyInfo, _ := os.Lstat(copyPath)
	raw, _ := os.ReadFile(copyPath)
	sum := sha256.Sum256(raw)
	if !copyInfo.Mode().IsRegular() || copyInfo.Mode().Perm() != 0600 || !strings.Contains(entries[0].Name(), hex.EncodeToString(sum[:])) {
		t.Fatal("copy must be a private file named by its sha256", copyInfo.Mode())
	}
	snapshot, _ := sql.Open("sqlite", copyPath)
	var copyVersion, copyIdentities int
	snapshot.QueryRow(`PRAGMA user_version`).Scan(&copyVersion)
	snapshot.QueryRow(`SELECT COUNT(*) FROM identities`).Scan(&copyIdentities)
	_, noPeople := snapshot.Exec(`SELECT 1 FROM people`)
	snapshot.Close()
	if copyVersion != 1 || copyIdentities != 4 || noPeople == nil {
		t.Fatal("copy must be the schema1 state", copyVersion, copyIdentities, noPeople)
	}
	r = open()
	r.db.QueryRow(`SELECT COUNT(*) FROM people`).Scan(&people)
	r.Close()
	if entries, _ = os.ReadDir(dir); len(entries) != 1 || people != 4 {
		t.Fatal("second open must not copy or bootstrap again", len(entries), people)
	}
}
