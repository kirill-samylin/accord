"""Real SQLite and symlinks; service/readiness functions are test doubles."""
import os
import pathlib
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / 'infra/accord'))
import recovery


class ActivationAtomicTests(unittest.TestCase):
    def test_failed_candidate_writes_survive_previous_executable_recovery(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = pathlib.Path(temporary)
            state.chmod(0o700)
            database = state / 'board.db'
            with sqlite3.connect(database) as connection:
                connection.executescript('PRAGMA user_version=1; CREATE TABLE data(n); INSERT INTO data VALUES(7);')
            database.chmod(0o600)
            commands, switches = [], []
            starts = 0

            def command(*args):
                nonlocal starts
                commands.append(args)
                if args == ('systemctl', 'start', 'accord.service'):
                    starts += 1
                    if starts == 1:
                        with sqlite3.connect(database) as connection:
                            connection.execute('INSERT INTO data VALUES(8)')

            ready = iter((False, True))
            with self.assertRaisesRegex(RuntimeError, 'previous executable restored and ready'):
                recovery.activate('releases/' + 'a' * 40, 'b' * 40, state,
                                  command, switches.append, lambda: next(ready))
            self.assertEqual(starts, 2)
            self.assertEqual(switches, ['b' * 40, 'a' * 40])
            with sqlite3.connect(database) as connection:
                self.assertEqual(connection.execute('SELECT n FROM data ORDER BY n').fetchall(), [(7,), (8,)])
            self.assertFalse((state / 'START_REQUIRES_INSPECTION').exists())
            self.assertEqual(commands[-1], ('systemctl', 'start', 'accord.service'))

    def test_switch_atomically_replaces_pointer_and_removes_staging(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = pathlib.Path(temporary)
            (base / 'current').symlink_to('releases/' + 'a' * 40)
            recovery.switch(base, 'b' * 40)
            self.assertEqual(os.readlink(base / 'current'), 'releases/' + 'b' * 40)
            self.assertFalse((base / '.current.new').exists())


if __name__ == '__main__':
    unittest.main(verbosity=2)
