# Synthetic filesystem/SQLite regressions; service commands are mocked, not Linux runtime.
import os
import pathlib
import sqlite3
import stat
import sys
import tempfile
import types
import unittest
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / 'infra/accord'))
import recovery
import restore


class RecoveryFaults(unittest.TestCase):
    def setUp(self):
        previous = os.umask(0o077)
        self.addCleanup(os.umask, previous)

    def test_committed_wal_version_is_read_without_immutable_bypass(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = pathlib.Path(temporary)
            state.chmod(0o700)
            path = state / 'board.db'
            database = sqlite3.connect(path)
            try:
                database.execute('PRAGMA user_version=1')
                database.execute('PRAGMA journal_mode=WAL')
                database.execute('PRAGMA user_version=2')
                database.commit()
                for entry in state.iterdir():
                    entry.chmod(0o600)
                direct = sqlite3.connect(path.as_uri() + '?mode=ro&immutable=1', uri=True)
                try:
                    self.assertEqual(direct.execute('PRAGMA user_version').fetchone()[0], 1)
                finally:
                    direct.close()
                self.assertEqual(recovery.schema_snapshot(state)[0], 2)
            finally:
                database.close()

    def test_inspection_guard_syncs_file_and_directory_before_failure_returns(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = pathlib.Path(temporary)
            state.chmod(0o700)
            events = []
            original = os.fsync

            def fsync(descriptor):
                events.append('directory' if stat.S_ISDIR(os.fstat(descriptor).st_mode) else 'file')
                original(descriptor)

            with mock.patch.object(recovery.os, 'fsync', side_effect=fsync):
                with self.assertRaisesRegex(RuntimeError, 'persistent inspection guard set'):
                    recovery.inspection_required(state, 'expected failure', RuntimeError('cause'))
            self.assertEqual(events, ['file', 'directory'])
            self.assertTrue((state / 'START_REQUIRES_INSPECTION').is_file())

    def test_restore_guard_directory_sync_failure_never_reaches_stop_or_database(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = pathlib.Path(temporary) / 'state'
            state.mkdir(mode=0o700)
            backups = pathlib.Path(temporary) / 'backups'
            backups.mkdir(mode=0o700)
            marker = state / 'RESTORE_REQUIRES_RESET'
            source = backups / ('board-20261018T000000Z-' + 'a' * 32 + '.db')
            database = state / 'board.db'
            database.write_bytes(b'unchanged synthetic live database')
            original_open, original_fsync = os.open, os.fsync

            def open_file(path, flags, mode=0o777):
                if path == '/run/lock/accord-admin.lock':
                    path = state / '.test-admin.lock'
                return original_open(path, flags, mode)

            def fsync(descriptor):
                if stat.S_ISDIR(os.fstat(descriptor).st_mode):
                    raise OSError('injected directory fsync failure')
                original_fsync(descriptor)

            with mock.patch.multiple(restore, STATE=state, BACKUPS=backups, MARKER=marker), \
                    mock.patch.object(restore.os, 'geteuid', return_value=0), \
                    mock.patch.object(restore.sys, 'platform', 'linux'), \
                    mock.patch.object(restore.sys, 'argv', ['restore.py', '--backup', str(source)]), \
                    mock.patch.object(restore.pwd, 'getpwnam', return_value=types.SimpleNamespace(pw_uid=os.getuid())), \
                    mock.patch.object(restore, 'verify'), \
                    mock.patch.object(restore, 'command') as command, \
                    mock.patch.object(restore.os, 'open', side_effect=open_file), \
                    mock.patch.object(restore.os, 'fsync', side_effect=fsync):
                with self.assertRaisesRegex(OSError, 'directory fsync failure'):
                    restore.run()
                command.assert_not_called()
            self.assertTrue(marker.is_file())
            self.assertEqual(database.read_bytes(), b'unchanged synthetic live database')


if __name__ == '__main__':
    unittest.main(verbosity=2)
