"""All-project backup units: synthetic isolated DBs, immutable files and recovery state."""
from contextlib import closing
import hashlib
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
import snapshot as projects_snapshot
from layout import inventory
from validate import validate

DEFAULT = 'p_' + 'a' * 32
OTHER = 'p_' + 'b' * 32
PENDING = 'p_' + 'c' * 32
QUARANTINE = 'p_' + 'f' * 32


def db(path, script):
    with closing(sqlite3.connect(path)) as connection, connection:
        connection.executescript(script)
    path.chmod(0o600)


def board(folder, value, version=2):
    for directory in (folder, folder / 'web', folder / 'web/blobs', folder / 'web/staging'):
        directory.mkdir(mode=0o700, exist_ok=True)
    db(folder / 'board.db', "PRAGMA user_version=2; CREATE TABLE tasks(id,value); INSERT INTO tasks VALUES('T-1','" + value + "');")
    db(folder / 'web/web.db', 'PRAGMA user_version=' + str(version) + '; CREATE TABLE design_assets(blob,size,sha256,preview,preview_size,status);')
    content = value.encode()
    blob = folder / 'web/blobs' / ('d' * 48)
    blob.write_bytes(content)
    blob.chmod(0o600)
    with closing(sqlite3.connect(folder / 'web/web.db')) as connection, connection:
        connection.execute('INSERT INTO design_assets VALUES(?,?,?,?,?,?)',
                           (blob.name, len(content), hashlib.sha256(content).hexdigest(), '', 0, 'ready'))


class ProjectsFixture:
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.state, self.copies = self.root / 'state', self.root / 'copies'
        self.state.mkdir(mode=0o700)
        self.copies.mkdir(mode=0o700)
        self.projects = self.state / 'projects-data'
        self.projects.mkdir(mode=0o700)
        (self.projects / 'projects').mkdir(mode=0o700)
        board(self.state, 'default-original', version=1)
        board(self.projects / 'projects' / OTHER, 'other-isolated')
        marker = self.projects / 'MULTIPROJECT_REQUIRES_CAPABILITY'
        marker.write_text('project registry capability required\n')
        marker.chmod(0o600)
        db(self.projects / 'registry.sqlite', '''PRAGMA user_version=1;
            CREATE TABLE registry_meta(id,default_id,default_db,default_web);
            CREATE TABLE projects(id,state);
            CREATE TABLE identities(id TEXT PRIMARY KEY,source TEXT NOT NULL,token_id TEXT NOT NULL,
                owner TEXT NOT NULL,machine TEXT NOT NULL);''')
        with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as connection, connection:
            connection.execute('INSERT INTO registry_meta VALUES(1,?,?,?)',
                               (DEFAULT, str(self.state / 'board.db'), str(self.state / 'web')))
            connection.executemany('INSERT INTO projects VALUES(?,?)', [(DEFAULT, 'active'), (OTHER, 'active')])
            connection.executemany('INSERT INTO identities VALUES(?,?,?,?,?)',
                                   [('i_one', 'token', 't_one', 'alice', 'laptop'),
                                    ('i_two', 'token', 't_two', 'alice', 'server')])

    def snapshot(self):
        return projects_snapshot.snapshot(self.state / 'board.db', self.state / 'web', self.projects, self.copies)


REGISTRY2_DDL = '''
CREATE TABLE IF NOT EXISTS people(id TEXT PRIMARY KEY, name TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '', role TEXT NOT NULL DEFAULT 'member'
  CHECK(role IN ('member','admin')), active INTEGER NOT NULL DEFAULT 1,
  version INTEGER NOT NULL DEFAULT 1, created INTEGER NOT NULL, updated INTEGER NOT NULL);
ALTER TABLE identities ADD COLUMN person_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS registry_identity_person ON identities(person_id);
CREATE TABLE IF NOT EXISTS admin_audit(id INTEGER PRIMARY KEY, event TEXT NOT NULL,
  actor_id TEXT NOT NULL, person_id TEXT NOT NULL DEFAULT '',
  identity_id TEXT NOT NULL DEFAULT '', details TEXT NOT NULL DEFAULT '', at INTEGER NOT NULL);
PRAGMA user_version=2;
'''


def registry1_copy(projects, sidecar=False):
    """Consistent pre-DDL copy as the capable binary writes it: 0700 dir, 0600 file."""
    folder = projects / 'registry.sqlite.migrations'
    folder.mkdir(mode=0o700, exist_ok=True)
    staging = folder / 'registry1-pending.db'
    with closing(sqlite3.connect(projects / 'registry.sqlite')) as source, \
            closing(sqlite3.connect(staging)) as target:
        source.backup(target)
        target.execute('PRAGMA journal_mode=DELETE')
    staging.chmod(0o600)
    digest = hashlib.sha256(staging.read_bytes()).hexdigest()
    copy = folder / ('registry1-' + ('e' * 64 if sidecar else digest) + '.db')
    staging.rename(copy)
    if sidecar:
        checksum = copy.with_name(copy.name + '.sha256')
        checksum.write_text(digest + '  ' + copy.name + '\n')
        checksum.chmod(0o600)
    return copy


def registry2(projects, copy=True, sidecar=False, bootstrap=True):
    """Apply the documented registry 1->2 migration to a fixture registry."""
    if copy:
        registry1_copy(projects, sidecar=sidecar)
    with closing(sqlite3.connect(projects / 'registry.sqlite')) as connection, connection:
        connection.executescript(REGISTRY2_DDL)
        if bootstrap:
            for index, (identity, owner, machine) in enumerate(
                    connection.execute('SELECT id,owner,machine FROM identities ORDER BY id').fetchall()):
                person = 'u_' + identity
                connection.execute('INSERT INTO people VALUES(?,?,?,?,?,?,?,?)',
                                   (person, owner + ' ' + machine, '', 'admin' if index == 0 else 'member', 1, 1, 10, 10))
                connection.execute('UPDATE identities SET person_id=? WHERE id=?', (person, identity))


class ProjectsSnapshot(ProjectsFixture, unittest.TestCase):
    def test_both_projects_same_ids_preserve_distinct_rows_and_blobs(self):
        result = self.snapshot()
        manifest = validate(result)
        self.assertEqual(manifest['format'], 2)
        self.assertEqual(manifest['default']['web_schema'], 1)
        self.assertEqual(manifest['projects'][0]['web_schema'], 2)
        for relative, expected in [('default', 'default-original'), ('projects/' + OTHER, 'other-isolated')]:
            with closing(sqlite3.connect(result / relative / 'board.db')) as connection:
                self.assertEqual(connection.execute('SELECT * FROM tasks').fetchall(), [('T-1', expected)])
            self.assertEqual((result / relative / 'web/blobs' / ('d' * 48)).read_text(), expected)
        self.assertFalse(list(result.rglob('*-wal')))
        self.assertFalse(list(result.rglob('*-shm')))

    def test_pending_unpublished_storage_is_preserved(self):
        with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as connection, connection:
            connection.execute('INSERT INTO projects VALUES(?,?)', (PENDING, 'provisioning'))
        folder = self.projects / 'projects' / PENDING
        folder.mkdir(mode=0o700)
        for name, content in [('board.db', b'incomplete fixture'), ('board.db-journal', b'private journal')]:
            path = folder / name
            path.write_bytes(content)
            path.chmod(0o600)
        result = self.snapshot()
        self.assertEqual((result / 'projects' / PENDING / 'board.db-journal').read_bytes(), b'private journal')
        self.assertEqual(validate(result)['projects'][-1]['state'], 'provisioning')

    def test_missing_pending_storage_is_recorded(self):
        with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as connection, connection:
            connection.execute('INSERT INTO projects VALUES(?,?)', (PENDING, 'provisioning'))
        result = self.snapshot()
        self.assertEqual(list((result / 'projects' / PENDING).iterdir()), [])
        validate(result)

    def test_quarantined_provisioning_is_preserved_without_runtime_registration(self):
        quarantine = self.projects / 'quarantine'
        quarantine.mkdir(mode=0o700)
        folder = quarantine / QUARANTINE
        folder.mkdir(mode=0o700)
        private = folder / 'board.db-journal'
        private.write_bytes(b'unpublished incomplete evidence')
        private.chmod(0o600)
        result = self.snapshot()
        manifest = validate(result)
        self.assertEqual(manifest['quarantine'], [QUARANTINE])
        self.assertNotIn(QUARANTINE, [entry['id'] for entry in manifest['projects']])
        self.assertEqual((result / 'quarantine' / QUARANTINE / private.name).read_bytes(), private.read_bytes())
        private.unlink()
        folder.rmdir()
        (quarantine / OTHER).mkdir(mode=0o700)
        with self.assertRaisesRegex(RuntimeError, 'quarantined'):
            self.snapshot()

    def test_manifest_tampering_and_foreign_files_fail(self):
        result = self.snapshot()
        file = result / 'projects' / OTHER / 'web/blobs' / ('d' * 48)
        file.write_bytes(b'changed')
        with self.assertRaises(RuntimeError):
            validate(result)
        foreign = result / 'default' / 'foreign.db'
        foreign.write_bytes(b'foreign')
        foreign.chmod(0o600)
        with self.assertRaises(RuntimeError):
            validate(result)

    def test_unaccounted_web_file_is_not_silently_omitted(self):
        unknown = self.state / 'web/unaccounted.db'
        unknown.write_bytes(b'synthetic')
        unknown.chmod(0o600)
        with self.assertRaisesRegex(RuntimeError, 'unaccounted'):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_unaccounted_active_project_file_is_not_silently_omitted(self):
        unknown = self.projects / 'projects' / OTHER / 'foreign.db'
        unknown.write_bytes(b'synthetic')
        unknown.chmod(0o600)
        with self.assertRaisesRegex(RuntimeError, 'unaccounted'):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_unresolved_rollback_journal_requires_recovery(self):
        for file in (self.projects / 'registry.sqlite', self.state / 'board.db',
                     self.projects / 'projects' / OTHER / 'web/web.db'):
            with self.subTest(database=file.name):
                journal = file.with_name(file.name + '-journal')
                journal.write_bytes(b'synthetic unresolved journal')
                journal.chmod(0o600)
                with self.assertRaisesRegex(RuntimeError, 'journal'):
                    self.snapshot()
                journal.unlink()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_unknown_project_missing_storage_and_wrong_binding_fail(self):
        orphan = self.projects / 'projects' / PENDING
        orphan.mkdir(mode=0o700)
        with self.assertRaises(RuntimeError):
            self.snapshot()
        orphan.rmdir()
        with self.assertRaises(RuntimeError):
            inventory(self.projects, self.state / 'different.db', self.state / 'web')
        (self.projects / 'projects' / OTHER).rename(self.root / 'missing-project')
        with self.assertRaises(FileNotFoundError):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_private_symlink_cannot_hide_in_inventory_or_snapshot(self):
        path = self.projects / 'projects' / OTHER / 'web/blobs' / ('e' * 48)
        path.symlink_to(self.state / 'board.db')
        with self.assertRaises(RuntimeError):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_low_reserve_preserves_last_good_and_other_backup_formats(self):
        good = self.snapshot()
        old = self.copies / ('web-' + '0' * 16)
        old.mkdir(mode=0o700)
        with patch.object(projects_snapshot.shutil, 'disk_usage', return_value=type('Disk', (), {'free': 0})()):
            with self.assertRaises(RuntimeError):
                self.snapshot()
        validate(good)
        with patch.object(projects_snapshot.LEGACY, 'RETAIN', 1):
            self.snapshot()
        self.assertTrue(old.is_dir())
        self.assertEqual(len([p for p in self.copies.iterdir() if p.name.startswith('projects-')]), 1)

    def test_uncheckpointed_registry_wal_is_rejected(self):
        with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as connection:
            connection.execute('PRAGMA journal_mode=WAL')
            connection.execute('INSERT INTO projects VALUES(?,?)', (PENDING, 'provisioning'))
            connection.commit()
            with self.assertRaises(RuntimeError):
                self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_legacy_full_snapshot_refuses_activated_registry(self):
        legacy = projects_snapshot.LEGACY
        with self.assertRaisesRegex(RuntimeError, 'all-project'):
            legacy.snapshot(self.state, self.copies)


class RegistrySchema2(ProjectsFixture, unittest.TestCase):
    def test_registry2_with_pre_ddl_copy_is_snapshotted_and_restored_byte_exact(self):
        copy = registry1_copy(self.projects)
        registry2(self.projects, copy=False)
        self.assertEqual(inventory(self.projects, self.state / 'board.db', self.state / 'web')['registry_schema'], 2)
        result = self.snapshot()
        manifest = validate(result)
        self.assertEqual(manifest['registry_schema'], 2)
        saved = result / 'registry/registry.sqlite.migrations' / copy.name
        self.assertEqual(saved.read_bytes(), copy.read_bytes())
        self.assertEqual(saved.stat().st_mode & 0o777, 0o600)
        self.assertIn('registry/registry.sqlite.migrations/' + copy.name, manifest['files'])
        with closing(sqlite3.connect(saved)) as historical, closing(sqlite3.connect(result / 'registry/registry.sqlite')) as live:
            self.assertEqual(historical.execute('PRAGMA user_version').fetchone()[0], 1)
            self.assertEqual(live.execute('PRAGMA user_version').fetchone()[0], 2)
            self.assertEqual(live.execute("SELECT count(*) FROM identities WHERE person_id=''").fetchone()[0], 0)

    def test_registry1_snapshot_before_the_bump_still_validates(self):
        manifest = validate(self.snapshot())
        self.assertEqual(manifest['registry_schema'], 1)
        self.assertNotIn('registry/registry.sqlite.migrations', {Path(name).parent.as_posix() for name in manifest['files']})

    def test_board_style_sidecar_copy_is_accepted(self):
        registry2(self.projects, sidecar=True)
        manifest = validate(self.snapshot())
        self.assertEqual(len([name for name in manifest['files'] if 'registry.sqlite.migrations' in name]), 2)

    def test_incomplete_or_undeclared_registry2_is_never_published(self):
        for sql in ('DROP TABLE people', 'DROP TABLE admin_audit', 'DROP INDEX registry_identity_person',
                    "UPDATE identities SET person_id='u_missing' WHERE id='i_one'", 'PRAGMA user_version=3'):
            with self.subTest(sql=sql):
                fixture = RegistrySchema2('test_registry1_snapshot_before_the_bump_still_validates')
                fixture.setUp()
                try:
                    registry2(fixture.projects)
                    with closing(sqlite3.connect(fixture.projects / 'registry.sqlite')) as connection, connection:
                        connection.execute(sql)
                    with self.assertRaises((RuntimeError, sqlite3.Error)):
                        fixture.snapshot()
                    self.assertEqual(list(fixture.copies.iterdir()), [])
                finally:
                    fixture.doCleanups()
        with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as connection, connection:
            connection.execute('CREATE TABLE people(id)')
        with self.assertRaisesRegex(RuntimeError, 'undeclared'):
            self.snapshot()

    def test_registry_copy_checksum_name_and_unknown_entries_are_enforced(self):
        copy = registry1_copy(self.projects)
        registry2(self.projects, copy=False)
        good = self.snapshot()
        copy.write_bytes(copy.read_bytes() + b'\0')
        with self.assertRaisesRegex(RuntimeError, 'checksum'):
            self.snapshot()
        copy.unlink()
        folder = self.projects / 'registry.sqlite.migrations'
        for name in ('schema1-' + 'a' * 64 + '.db', 'registry2-' + 'a' * 64 + '.db', 'registry1-' + 'a' * 64 + '.db.sha256'):
            with self.subTest(name=name):
                foreign = folder / name
                foreign.write_bytes(b'foreign')
                foreign.chmod(0o600)
                with self.assertRaises(RuntimeError):
                    self.snapshot()
                foreign.unlink()
        sidecar = registry1_copy(self.projects, sidecar=True)
        sidecar.with_name(sidecar.name + '.sha256').write_text('0' * 64 + '  ' + sidecar.name + '\n')
        with self.assertRaisesRegex(RuntimeError, 'checksum'):
            self.snapshot()
        folder.chmod(0o750)
        with self.assertRaises(RuntimeError):
            self.snapshot()
        folder.chmod(0o700)
        validate(good)
        tampered = good / 'registry/registry.sqlite.migrations' / copy.name
        tampered.write_bytes(b'rewritten history')
        with self.assertRaises(RuntimeError):
            validate(good)

    def test_manifest_registry_schema_must_match_the_copied_database(self):
        registry2(self.projects)
        result = self.snapshot()
        manifest = result / 'manifest.json'
        text = manifest.read_text()
        manifest.write_text(text.replace('"registry_schema": 2', '"registry_schema": 1'))
        with self.assertRaises(RuntimeError):
            validate(result)
        manifest.write_text(text.replace('"registry_schema": 2', '"registry_schema": 3'))
        with self.assertRaisesRegex(RuntimeError, 'format'):
            validate(result)
