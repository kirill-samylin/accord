"""Managed command state must survive complete, validated schema5 snapshots."""
from contextlib import closing
import hashlib
import sqlite3
import unittest
from unittest.mock import patch

import test_coordination_storage as coordination
from test_projects_snapshot import ProjectsFixture, OTHER
import test_projects_restore as offline
from upgrade_storage import facts
from validate import validate


class ManagedStorage(ProjectsFixture, unittest.TestCase):
    def upgrade(self, folder):
        coordination.CoordinationStorage.upgrade(self, folder)
        with closing(sqlite3.connect(folder / 'board.db')) as db, db:
            db.executescript('''
                CREATE TABLE managed_metadata(id INTEGER PRIMARY KEY CHECK(id=1), version INTEGER NOT NULL CHECK(version=1));
                INSERT INTO managed_metadata VALUES(1,1);
                CREATE TABLE managed_hosts(key TEXT PRIMARY KEY,data TEXT NOT NULL);
                CREATE TABLE managed_runs(key TEXT PRIMARY KEY,data TEXT NOT NULL);
                CREATE TABLE managed_commands(key TEXT PRIMARY KEY,data TEXT NOT NULL);
                PRAGMA user_version=5;
            ''')
            db.execute('INSERT INTO managed_hosts VALUES(?,?)', ('host-test', '{"state":"active","worker_epoch":2}'))
            db.execute('INSERT INTO managed_runs VALUES(?,?)', ('run-test', '{"state":"needs_recovery","generation":3}'))
            db.execute('INSERT INTO managed_commands VALUES(?,?)', ('cmd-test', '{"status":"accepted","dispatch_id":"synthetic"}'))

    def history(self, folder):
        directory = folder / 'board.db.migrations'
        directory.mkdir(mode=0o700)
        source = directory / ('schema4-' + 'd' * 64 + '.db')
        source.write_bytes(b'immutable synthetic schema4 evidence')
        source.chmod(0o600)
        checksum = source.with_name(source.name + '.sha256')
        checksum.write_text(hashlib.sha256(source.read_bytes()).hexdigest() + '  ' + source.name + '\n')
        checksum.chmod(0o600)

    def test_all_project_command_state_and_schema4_history_are_preserved(self):
        other = self.projects / 'projects' / OTHER
        for folder in (self.state, other):
            self.upgrade(folder)
            self.history(folder)
        saved = self.snapshot()
        manifest = validate(saved)
        self.assertEqual(manifest['default']['board_schema'], 5)
        self.assertEqual(manifest['projects'][0]['board_schema'], 5)
        for source, relative in ((self.state, 'default'), (other, 'projects/' + OTHER)):
            self.assertEqual(facts(source / 'board.db'), facts(saved / relative / 'board.db'))
            for entry in (source / 'board.db.migrations').iterdir():
                self.assertEqual(entry.read_bytes(), (saved / relative / 'board.db.migrations' / entry.name).read_bytes())

    def test_missing_managed_command_table_cannot_publish_partial_snapshot(self):
        self.upgrade(self.state)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('DROP TABLE managed_commands')
        before = facts(self.state / 'board.db')
        with self.assertRaises(sqlite3.Error):
            self.snapshot()
        self.assertEqual(facts(self.state / 'board.db'), before)
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_empty_managed_metadata_is_not_repaired(self):
        self.upgrade(self.state)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('DELETE FROM managed_metadata')
        before = facts(self.state / 'board.db')
        with self.assertRaisesRegex(RuntimeError, 'managed agents metadata'):
            self.snapshot()
        self.assertEqual(facts(self.state / 'board.db'), before)
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_coordination_floor_still_required_in_schema5(self):
        self.upgrade(self.state)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('DELETE FROM coordination_metadata')
        with self.assertRaisesRegex(RuntimeError, 'coordination metadata'):
            self.snapshot()

    def test_schema4_history_checksum_tamper_rejected(self):
        self.upgrade(self.state)
        self.history(self.state)
        next((self.state / 'board.db.migrations').glob('*.db')).write_bytes(b'tampered')
        with self.assertRaisesRegex(RuntimeError, 'checksum'):
            self.snapshot()

    def test_mixed_schema4_and5_snapshot_is_valid(self):
        self.upgrade(self.state)
        coordination.CoordinationStorage.upgrade(self, self.projects / 'projects' / OTHER)
        manifest = validate(self.snapshot())
        self.assertEqual(manifest['default']['board_schema'], 5)
        self.assertEqual(manifest['projects'][0]['board_schema'], 4)

    def test_offline_restore5_preserves_source_before_mocked_auth_reset(self):
        for folder in (self.state, self.projects / 'projects' / OTHER):
            self.upgrade(folder)
            self.history(folder)
        saved = self.snapshot()
        original = (saved / 'manifest.json').read_bytes()
        output = self.root / 'restored5'
        with patch.object(offline.restore, 'require_capable'), patch.object(offline.restore.subprocess, 'run') as run:
            run.return_value.stdout = b'{"restored_state_reset":true}'
            run.return_value.stderr = b''
            offline.restore.restore(saved, output, self.root / 'synthetic-binary')
        self.assertEqual((saved / 'manifest.json').read_bytes(), original)
        self.assertEqual((output / 'board.db').read_bytes(), (saved / 'default/board.db').read_bytes())
        self.assertEqual((output / 'projects-data/projects' / OTHER / 'board.db').read_bytes(),
                         (saved / 'projects' / OTHER / 'board.db').read_bytes())
        self.assertTrue((output / 'RESTORE_REQUIRES_RESET').exists())
        self.assertTrue((output / 'RESTORE_AUTH_RESET_COMPLETE').exists())
