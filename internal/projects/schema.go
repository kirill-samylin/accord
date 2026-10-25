package projects

// RegistrySchemaVersion is the only registry schema this binary serves. A fresh
// root is created at this version; schema1 authorities are migrated on open.
const RegistrySchemaVersion = 2

const schema = `
CREATE TABLE IF NOT EXISTS registry_meta(id INTEGER PRIMARY KEY CHECK(id=1),default_id TEXT NOT NULL,default_db TEXT NOT NULL,default_web TEXT NOT NULL,signing_key TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS rejected_provisions(id TEXT PRIMARY KEY,at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS identities(id TEXT PRIMARY KEY,source TEXT NOT NULL,token_id TEXT NOT NULL,owner TEXT NOT NULL,machine TEXT NOT NULL,person_id TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS credentials(identity_id TEXT PRIMARY KEY,hash TEXT UNIQUE NOT NULL,expires INTEGER NOT NULL,revoked INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS projects(id TEXT PRIMARY KEY,name TEXT NOT NULL,state TEXT NOT NULL,version INTEGER NOT NULL,creator TEXT NOT NULL,created INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS memberships(project_id TEXT NOT NULL,identity_id TEXT NOT NULL,role TEXT NOT NULL,local_id TEXT NOT NULL,active INTEGER NOT NULL,version INTEGER NOT NULL,access_epoch INTEGER NOT NULL,PRIMARY KEY(project_id,identity_id));
CREATE TABLE IF NOT EXISTS repositories(id TEXT PRIMARY KEY,project_id TEXT NOT NULL,url TEXT NOT NULL,branch TEXT NOT NULL,version INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(hash TEXT PRIMARY KEY,identity_id TEXT NOT NULL,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS session_tombstones(hash TEXT PRIMARY KEY,expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS session_epochs(project_id TEXT NOT NULL,session_id TEXT NOT NULL,identity_id TEXT NOT NULL,access_epoch INTEGER NOT NULL,PRIMARY KEY(project_id,session_id));
CREATE TABLE IF NOT EXISTS preauth(hash TEXT PRIMARY KEY,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS actions(id TEXT PRIMARY KEY,session_hash TEXT NOT NULL,operation TEXT NOT NULL,project_id TEXT NOT NULL,access_epoch INTEGER NOT NULL,payload TEXT NOT NULL DEFAULT '',result TEXT NOT NULL DEFAULT '',created INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY,event TEXT NOT NULL,identity_id TEXT NOT NULL,project_id TEXT NOT NULL,at INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS registry_session_expiry ON sessions(expires);
CREATE INDEX IF NOT EXISTS registry_action_session ON actions(session_hash);
CREATE INDEX IF NOT EXISTS registry_preauth_expiry ON preauth(expires);
CREATE INDEX IF NOT EXISTS registry_tombstone_expiry ON session_tombstones(expires);
` + peopleSchema + `
PRAGMA user_version=2;
`

// peopleSchema is shared by fresh creation and the 1→2 migration (docs/admin.md).
const peopleSchema = `
CREATE TABLE IF NOT EXISTS people(id TEXT PRIMARY KEY, name TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '', role TEXT NOT NULL DEFAULT 'member'
  CHECK(role IN ('member','admin')), active INTEGER NOT NULL DEFAULT 1,
  version INTEGER NOT NULL DEFAULT 1, created INTEGER NOT NULL, updated INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS registry_identity_person ON identities(person_id);
CREATE TABLE IF NOT EXISTS admin_audit(id INTEGER PRIMARY KEY, event TEXT NOT NULL,
  actor_id TEXT NOT NULL, person_id TEXT NOT NULL DEFAULT '',
  identity_id TEXT NOT NULL DEFAULT '', details TEXT NOT NULL DEFAULT '', at INTEGER NOT NULL);
`

// registryMigration1to2 runs inside one transaction after the pre-migration
// copy exists; people bootstrap follows it in the same transaction.
const registryMigration1to2 = `ALTER TABLE identities ADD COLUMN person_id TEXT NOT NULL DEFAULT '';` + peopleSchema + `PRAGMA user_version=2;`
