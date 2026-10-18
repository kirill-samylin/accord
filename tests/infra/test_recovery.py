# Synthetic filesystem/SQLite regressions; service commands are mocked, not Linux runtime.
import os
import pathlib
import sqlite3
import stat
import subprocess
import sys
import tempfile
import types
import unittest
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parents[2]
INFRA = ROOT / 'infra/accord'
sys.path.insert(0, str(INFRA))
import recovery
import install
import rollback
import restore


class ActivationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = pathlib.Path(self.temp.name)
        self.state = self.root / 'state'
        self.state.mkdir(mode=0o700)
        self.database = self.state / 'board.db'
        self.write_database(1)
        self.old, self.new = 'a' * 40, 'b' * 40
        self.previous = 'releases/' + self.old
        self.current = self.root / 'current'
        self.current.symlink_to(self.previous)
        self.commands = []
        self.switches = []
        self.starts = 0
        self.fail_starts = set()
        self.fail_backup = False
        self.ready = []
        self.on_start = lambda: None
        self.original = self.database.read_bytes()

    def write_database(self, version):
        with sqlite3.connect(self.database) as database:
            database.execute('CREATE TABLE IF NOT EXISTS data(value TEXT)')
            database.execute("INSERT INTO data VALUES('preserved')")
            database.execute('PRAGMA user_version=' + str(version))
        self.database.chmod(0o600)

    def command(self, *args):
        self.commands.append(args)
        if args[0] == 'runuser' and self.fail_backup:
            raise subprocess.CalledProcessError(1, args)
        if args[:2] == ('systemctl', 'start'):
            self.starts += 1
            self.on_start()
            if self.starts in self.fail_starts:
                raise subprocess.CalledProcessError(1, args)

    def switch(self, revision):
        self.switches.append(revision)
        temporary = self.root / '.current.new'
        temporary.symlink_to('releases/' + revision)
        os.replace(temporary, self.current)

    def healthy(self):
        return self.ready.pop(0) if self.ready else True

    def activate(self, previous=True):
        recovery.activate(self.previous if previous else None, self.new, self.state,
                          self.command, self.switch, self.healthy)

    def assert_preserved(self):
        self.assertEqual(self.database.read_bytes(), self.original)
        self.assertFalse((self.state / 'RESTORE_REQUIRES_RESET').exists())

    def assert_inspection(self):
        marker = self.state / 'START_REQUIRES_INSPECTION'
        self.assertTrue(marker.is_file())
        self.assertEqual(stat.S_IMODE(marker.stat().st_mode), 0o600)

    def test_start_exception_restores_previous_and_reports_failure(self):
        self.fail_starts = {1}
        with self.assertRaisesRegex(RuntimeError, 'previous executable restored and ready'):
            self.activate()
        self.assertEqual(os.readlink(self.current), self.previous)
        self.assertEqual(self.starts, 2)
        self.assert_preserved()

    def test_backup_failure_restarts_unchanged_previous(self):
        self.fail_backup = True
        with self.assertRaisesRegex(RuntimeError, 'previous executable restored and ready'):
            self.activate()
        self.assertEqual(self.starts, 1)
        self.assertNotIn(self.new, self.switches)
        self.assertEqual(os.readlink(self.current), self.previous)
        self.assert_preserved()

    def test_readiness_failure_recovers_previous(self):
        self.ready = [False, True]
        with self.assertRaisesRegex(RuntimeError, 'previous executable restored and ready'):
            self.activate()
        self.assertEqual(self.switches, [self.new, self.old])
        self.assertEqual(self.starts, 2)
        self.assert_preserved()

    def test_schema_migration_fails_closed_and_preserves_migrated_data(self):
        def migrate():
            with sqlite3.connect(self.database) as database:
                database.execute('PRAGMA user_version=2')
                database.execute("UPDATE data SET value='migrated'")
        self.on_start = migrate
        self.ready = [False]
        with self.assertRaisesRegex(RuntimeError, 'compatibility unproven'):
            self.activate()
        self.assertEqual(os.readlink(self.current), 'releases/' + self.new)
        self.assertEqual(self.starts, 1)
        self.assert_inspection()
        with sqlite3.connect(self.database) as database:
            self.assertEqual(database.execute('PRAGMA user_version').fetchone()[0], 2)
            self.assertEqual(database.execute('SELECT value FROM data').fetchone()[0], 'migrated')

    def test_unknown_preexisting_schema_does_not_infer_compatibility(self):
        with sqlite3.connect(self.database) as database:
            database.execute('PRAGMA user_version=99')
        self.fail_starts = {1}
        with self.assertRaisesRegex(RuntimeError, 'compatibility unproven'):
            self.activate()
        self.assertEqual(self.starts, 1)
        self.assertEqual(os.readlink(self.current), 'releases/' + self.new)
        self.assert_inspection()

    def test_unchanged_schema2_allows_previous_executable_recovery(self):
        with sqlite3.connect(self.database) as database:
            database.execute('PRAGMA user_version=2')
        self.original = self.database.read_bytes()
        self.fail_starts = {1}
        with self.assertRaisesRegex(RuntimeError, 'previous executable restored and ready'):
            self.activate()
        self.assertEqual(self.starts, 2)
        self.assertEqual(os.readlink(self.current), self.previous)
        self.assert_preserved()

    def test_unversioned_ddl_change_also_blocks_recovery(self):
        def migrate():
            with sqlite3.connect(self.database) as database:
                database.execute('CREATE TABLE unexpected(value TEXT)')
        self.on_start = migrate
        self.fail_starts = {1}
        with self.assertRaisesRegex(RuntimeError, 'compatibility unproven'):
            self.activate()
        self.assertEqual(self.starts, 1)
        self.assert_inspection()

    def test_first_install_failure_stays_stopped(self):
        self.database.unlink()
        self.current.unlink()
        self.fail_starts = {1}
        with self.assertRaisesRegex(RuntimeError, 'first installation; service stopped'):
            self.activate(previous=False)
        self.assertEqual(self.commands[-1], ('systemctl', 'stop', 'accord.service'))
        self.assertEqual(self.starts, 1)
        self.assertFalse(self.database.exists())
        self.assert_inspection()

    def test_successful_update_selects_target_without_recovery(self):
        self.activate()
        self.assertEqual(os.readlink(self.current), 'releases/' + self.new)
        self.assertEqual(self.starts, 1)
        self.assertEqual(self.switches, [self.new])
        self.assertFalse((self.state / 'START_REQUIRES_INSPECTION').exists())
        self.assert_preserved()

    def test_recovery_start_failure_stops_previous_and_guards_boot(self):
        self.fail_starts = {1, 2}
        with self.assertRaisesRegex(RuntimeError, 'activation and recovery failed'):
            self.activate()
        self.assertEqual(os.readlink(self.current), self.previous)
        self.assertEqual(self.commands[-1], ('systemctl', 'stop', 'accord.service'))
        self.assert_inspection()
        self.assert_preserved()

    def test_existing_restore_guard_is_preserved_without_commands(self):
        marker = self.state / 'RESTORE_REQUIRES_RESET'
        marker.write_text('reset required')
        with self.assertRaisesRegex(RuntimeError, 'guard is present'):
            self.activate()
        self.assertEqual(marker.read_text(), 'reset required')
        self.assertEqual(self.commands, [])
        self.assertEqual(os.readlink(self.current), self.previous)

    def test_existing_inspection_guard_blocks_start(self):
        (self.state / 'START_REQUIRES_INSPECTION').touch()
        with self.assertRaisesRegex(RuntimeError, 'guard is present'):
            self.activate()
        self.assertEqual(self.commands, [])

    def test_private_database_symlink_is_rejected(self):
        target = self.root / 'outside.db'
        self.database.rename(target)
        self.database.symlink_to(target)
        with self.assertRaisesRegex(RuntimeError, 'compatibility unproven'):
            self.activate()
        self.assertEqual(self.starts, 0)
        self.assertEqual(target.read_bytes(), self.original)
        self.assert_inspection()

    def test_both_entrypoints_use_same_guarded_activation(self):
        self.assertIs(install.activate, recovery.activate)
        self.assertIs(rollback.activate, recovery.activate)
        self.assertIs(install.switch, recovery.switch)
        self.assertIs(rollback.switch, recovery.switch)
        self.assertIs(install.healthy, recovery.healthy)
        self.assertIs(rollback.healthy, recovery.healthy)
        self.assertIn('recovery.py', install.ADMIN_FILES)
        self.assertIn('START_REQUIRES_INSPECTION', (INFRA / 'accord.service').read_text())


class RestoreOrderingTests(unittest.TestCase):
    def setUp(self):
        previous = os.umask(0o077)
        self.addCleanup(os.umask, previous)

    def test_guard_file_and_directory_synced_before_first_stop(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = pathlib.Path(temporary) / 'state'
            state.mkdir(mode=0o700)
            backups = pathlib.Path(temporary) / 'backups'
            backups.mkdir(mode=0o700)
            marker = state / 'RESTORE_REQUIRES_RESET'
            source = backups / ('board-20261018T000000Z-' + 'a' * 32 + '.db')
            database = state / 'board.db'
            database.write_bytes(b'unchanged synthetic live database')
            events = []
            original_fsync = os.fsync
            original_open = os.open

            def open_file(path, flags, mode=0o777):
                if path == '/run/lock/accord-admin.lock':
                    path = state / '.test-admin.lock'
                return original_open(path, flags, mode)

            def fsync(descriptor):
                kind = 'directory' if stat.S_ISDIR(os.fstat(descriptor).st_mode) else 'file'
                events.append(kind)
                original_fsync(descriptor)

            def command(*args):
                events.append('stop')
                self.assertTrue(marker.exists())
                raise RuntimeError('stop here before any DB mutation')

            with mock.patch.multiple(restore, STATE=state, BACKUPS=backups, MARKER=marker), \
                    mock.patch.object(restore.os, 'geteuid', return_value=0), \
                    mock.patch.object(restore.sys, 'platform', 'linux'), \
                    mock.patch.object(restore.sys, 'argv', ['restore.py', '--backup', str(source)]), \
                    mock.patch.object(restore.pwd, 'getpwnam', return_value=types.SimpleNamespace(pw_uid=os.getuid())), \
                    mock.patch.object(restore, 'verify'), \
                    mock.patch.object(restore, 'command', side_effect=command), \
                    mock.patch.object(restore.os, 'open', side_effect=open_file), \
                    mock.patch.object(restore.os, 'fsync', side_effect=fsync):
                with self.assertRaisesRegex(RuntimeError, 'stop here'):
                    restore.run()
            self.assertEqual(events, ['file', 'directory', 'stop'])
            self.assertEqual(database.read_bytes(), b'unchanged synthetic live database')


if __name__ == '__main__':
    unittest.main(verbosity=2)
