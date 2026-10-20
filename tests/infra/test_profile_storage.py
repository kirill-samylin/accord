"""Real isolated SQLite schema3 snapshots and strict migration history checks."""
from contextlib import closing
import hashlib
import sqlite3
import unittest

from test_projects_snapshot import ProjectsFixture, OTHER, db
from upgrade_storage import facts
from validate import validate


class ProfileStorage(ProjectsFixture, unittest.TestCase):
    def upgrade(self, folder):
        with closing(sqlite3.connect(folder / 'board.db')) as connection, connection:
            connection.executescript('''PRAGMA user_version=3;
                ALTER TABLE tasks ADD COLUMN key TEXT;
                ALTER TABLE tasks ADD COLUMN data TEXT;
                CREATE TABLE metadata(id INTEGER PRIMARY KEY,sequence INTEGER);
                INSERT INTO metadata VALUES(1,0);
                CREATE TABLE note_metadata(id INTEGER PRIMARY KEY,sequence INTEGER);
                INSERT INTO note_metadata VALUES(1,0);
                CREATE TABLE sessions(key TEXT PRIMARY KEY,data TEXT);
                CREATE TABLE role_generations(role TEXT PRIMARY KEY,generation INTEGER);
                CREATE TABLE events(cursor INTEGER PRIMARY KEY,data TEXT);
                CREATE TABLE idempotency(key TEXT PRIMARY KEY,hash TEXT,response TEXT);
                CREATE TABLE tokens(id TEXT PRIMARY KEY,hash TEXT,owner TEXT,machine TEXT,expires INTEGER,revoked INTEGER);
                CREATE TABLE notes(key TEXT PRIMARY KEY,data TEXT);
                CREATE TABLE agent_profiles(key TEXT PRIMARY KEY,data TEXT NOT NULL);
                CREATE TABLE agent_profile_revisions(profile_id TEXT,version INTEGER,data TEXT,
                    PRIMARY KEY(profile_id,version));
                INSERT INTO agent_profiles VALUES('ap_same','profile-content');
                INSERT INTO agent_profile_revisions VALUES('ap_same',1,'historical-content');''')

    def migration(self, folder):
        history = folder / 'board.db.migrations'
        history.mkdir(mode=0o700)
        old = history / ('schema2-' + 'c' * 64 + '.db')
        db(old, 'PRAGMA user_version=2; CREATE TABLE old(value);')
        checksum = old.with_name(old.name + '.sha256')
        checksum.write_text(hashlib.sha256(old.read_bytes()).hexdigest() + '  ' + old.name + '\n')
        checksum.chmod(0o600)
        return old, checksum

    def test_schema3_profiles_revisions_and_migration_copies_are_preserved(self):
        other = self.projects / 'projects' / OTHER
        for folder in (self.state, other):
            self.migration(folder)
            self.upgrade(folder)
        saved = self.snapshot()
        manifest = validate(saved)
        self.assertEqual(manifest['default']['board_schema'], 3)
        self.assertEqual(manifest['projects'][0]['board_schema'], 3)
        for prefix, source in [('default', self.state), ('projects/' + OTHER, other)]:
            self.assertEqual(facts(saved / prefix / 'board.db'), facts(source / 'board.db'))
            for old in (source / 'board.db.migrations').iterdir():
                self.assertEqual((saved / prefix / 'board.db.migrations' / old.name).read_bytes(), old.read_bytes())

    def test_incomplete_schema3_is_not_accepted_by_version_alone(self):
        with closing(sqlite3.connect(self.state / 'board.db')) as connection, connection:
            connection.execute('PRAGMA user_version=3')
        with self.assertRaises(sqlite3.Error):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_missing_checksum_and_unknown_history_files_are_rejected(self):
        old, checksum = self.migration(self.projects / 'projects' / OTHER)
        content = checksum.read_bytes()
        checksum.unlink()
        with self.assertRaisesRegex(RuntimeError, 'incomplete'):
            self.snapshot()
        checksum.write_bytes(content)
        checksum.chmod(0o600)
        unknown = old.parent / 'foreign.txt'
        unknown.write_bytes(b'synthetic')
        unknown.chmod(0o600)
        with self.assertRaisesRegex(RuntimeError, 'unknown'):
            self.snapshot()

    def test_migration_checksum_tamper_and_symlink_fail_closed(self):
        old, checksum = self.migration(self.state)
        old.write_bytes(b'changed snapshot')
        with self.assertRaisesRegex(RuntimeError, 'checksum'):
            self.snapshot()
        old.unlink()
        old.symlink_to(self.state / 'board.db')
        with self.assertRaises(RuntimeError):
            self.snapshot()

    def test_logical_comparison_ignores_ddl_additions_but_detects_old_row_changes(self):
        self.upgrade(self.state)
        before = facts(self.state / 'board.db')
        with closing(sqlite3.connect(self.state / 'board.db')) as connection, connection:
            connection.execute('CREATE TABLE additional_after_migration(value)')
        current = facts(self.state / 'board.db')
        self.assertTrue(all(current[key] == value for key, value in before.items()))
        with closing(sqlite3.connect(self.state / 'board.db')) as connection, connection:
            connection.execute("UPDATE tasks SET value='changed'")
        self.assertNotEqual(facts(self.state / 'board.db')['tasks'], before['tasks'])
