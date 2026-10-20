"""Coordination schema4 snapshots, historical bytes and offline restore units."""
from contextlib import closing
import hashlib
import sqlite3
import unittest
from unittest.mock import patch

import test_profile_storage as profiles
import test_projects_restore as offline
from test_projects_snapshot import ProjectsFixture, OTHER
from upgrade_storage import facts
from validate import validate


class CoordinationStorage(ProjectsFixture, unittest.TestCase):
    def upgrade(self, folder):
        profiles.ProfileStorage.upgrade(self, folder)
        with closing(sqlite3.connect(folder / 'board.db')) as db, db:
            db.executescript('''PRAGMA user_version=4;
                CREATE TABLE coordination_metadata(id INTEGER PRIMARY KEY CHECK(id=1),version INTEGER NOT NULL);
                INSERT INTO coordination_metadata VALUES(1,1);''')
            db.execute('UPDATE tasks SET data=?', ('{"assignment":{"profile_id":"synthetic","profile_version":1}}',))

    def schema3_history(self, folder):
        path = folder / 'board.db.migrations'
        path.mkdir(mode=0o700)
        old = path / ('schema3-' + 'c' * 64 + '.db')
        old.write_bytes(b'immutable synthetic schema3 migration evidence')
        old.chmod(0o600)
        checksum = old.with_name(old.name + '.sha256')
        checksum.write_text(hashlib.sha256(old.read_bytes()).hexdigest() + '  ' + old.name + '\n')
        checksum.chmod(0o600)

    def test_schema4_preserves_assignment_profiles_history_and_blobs_for_all_projects(self):
        for folder in (self.state, self.projects / 'projects' / OTHER):
            self.upgrade(folder)
            self.schema3_history(folder)
        saved = self.snapshot()
        manifest = validate(saved)
        self.assertEqual(manifest['default']['board_schema'], 4)
        self.assertEqual(manifest['projects'][0]['board_schema'], 4)
        for source, prefix in ((self.state, 'default'),
                               (self.projects / 'projects' / OTHER, 'projects/' + OTHER)):
            self.assertEqual(facts(source / 'board.db'), facts(saved / prefix / 'board.db'))
            for entry in (source / 'board.db.migrations').iterdir():
                self.assertEqual(entry.read_bytes(), (saved / prefix / 'board.db.migrations' / entry.name).read_bytes())

    def test_schema4_requires_metadata_table_before_publishing_snapshot(self):
        self.upgrade(self.state)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('DROP TABLE coordination_metadata')
        with self.assertRaises(sqlite3.Error):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_schema4_rejects_missing_or_wrong_metadata_without_repair(self):
        self.upgrade(self.state)
        for sql in ('DELETE FROM coordination_metadata',
                    'INSERT INTO coordination_metadata VALUES(1,2)'):
            with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
                db.execute(sql)
            before = facts(self.state / 'board.db')
            with self.assertRaisesRegex(RuntimeError, 'coordination metadata'):
                self.snapshot()
            self.assertEqual(facts(self.state / 'board.db'), before)
            self.assertEqual(list(self.copies.iterdir()), [])

    def test_mixed_schema3_and4_snapshot_is_valid(self):
        self.upgrade(self.state)
        profiles.ProfileStorage.upgrade(self, self.projects / 'projects' / OTHER)
        manifest = validate(self.snapshot())
        self.assertEqual(manifest['default']['board_schema'], 4)
        self.assertEqual(manifest['projects'][0]['board_schema'], 3)

    def test_schema3_migration_checksum_is_not_skipped(self):
        self.upgrade(self.state)
        self.schema3_history(self.state)
        next((self.state / 'board.db.migrations').glob('*.db')).write_bytes(b'tampered')
        with self.assertRaisesRegex(RuntimeError, 'checksum'):
            self.snapshot()

    def test_unknown_schema5_and_historical_schema4_require_inspection(self):
        self.upgrade(self.state)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('PRAGMA user_version=5')
        with self.assertRaises(RuntimeError):
            self.snapshot()
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('PRAGMA user_version=4')
        self.schema3_history(self.state)
        path = self.state / 'board.db.migrations' / ('schema4-' + 'f' * 64 + '.db')
        path.write_bytes(b'unknown future migration evidence')
        path.chmod(0o600)
        with self.assertRaisesRegex(RuntimeError, 'unknown'):
            self.snapshot()

    def test_offline_restore4_preserves_exact_bytes_before_mocked_auth_reset(self):
        for folder in (self.state, self.projects / 'projects' / OTHER):
            self.upgrade(folder)
            self.schema3_history(folder)
        saved = self.snapshot()
        source_manifest = (saved / 'manifest.json').read_bytes()
        output = self.root / 'restored4'
        with patch.object(offline.restore, 'require_capable'), patch.object(offline.restore.subprocess, 'run') as run:
            run.return_value.stdout = b'{"restored_state_reset":true}'
            run.return_value.stderr = b''
            offline.restore.restore(saved, output, self.root / 'synthetic-binary')
        self.assertEqual((saved / 'manifest.json').read_bytes(), source_manifest)
        self.assertEqual((output / 'board.db').read_bytes(), (saved / 'default/board.db').read_bytes())
        self.assertEqual((output / 'projects-data/projects' / OTHER / 'board.db').read_bytes(),
                         (saved / 'projects' / OTHER / 'board.db').read_bytes())
        self.assertTrue((output / 'RESTORE_REQUIRES_RESET').exists())
        self.assertTrue((output / 'RESTORE_AUTH_RESET_COMPLETE').exists())
