"""Independent rollback steps with real config bytes and preserved state bytes."""
import pathlib
import sys
import tempfile
import unittest
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/next'))
import host
import recover


class NextRecovery(unittest.TestCase):
    def test_canonical_absolute_existing_pointer_supported(self):
        with tempfile.TemporaryDirectory() as temporary:
            base = pathlib.Path(temporary)
            previous = str(base / 'releases' / ('a' * 40))
            (base / 'current').symlink_to(previous)
            self.assertEqual(host.pointer(base), previous)
            host.switch(base, 'releases/' + 'b' * 40)
            self.assertEqual(host.pointer(base), 'releases/' + 'b' * 40)
            host.switch(base, previous)
            self.assertEqual(host.pointer(base), previous)
            for bad in ('/tmp/releases/' + 'a' * 40, 'releases/../secrets', '/etc/accord'):
                with self.assertRaises(RuntimeError):
                    host.switch(base, bad)

    def rollback_fixture(self, failure=None):
        with tempfile.TemporaryDirectory() as temporary:
            root = pathlib.Path(temporary)
            work = root / 'work'
            work.mkdir()
            (work / 'previous-nginx').write_bytes(b'old nginx')
            (work / 'previous-unit').write_bytes(b'old unit')
            database = root / 'board.db'
            database.write_bytes(b'new user writes must survive')
            state = {'old_go': 'releases/' + 'a' * 40, 'old_next': None,
                     'old_unit': True, 'old_next_enabled': False, 'old_next_active': False}
            calls = []

            def write(path, data):
                calls.append(('write', path, data))
                if failure and path == root / failure:
                    raise OSError('simulated filesystem failure')

            patches = [mock.patch.object(recover, 'BASE', root / 'next'),
                       mock.patch.object(recover, 'GO', root / 'go'),
                       mock.patch.object(recover, 'NGINX', root / 'nginx'),
                       mock.patch.object(recover, 'UNIT', root / 'unit'),
                       mock.patch.object(recover, 'atomic', side_effect=write),
                       mock.patch.object(recover, 'switch', side_effect=lambda *args: calls.append(('switch', *args))),
                       mock.patch.object(recover, 'run', side_effect=lambda *args: calls.append(('run', *args))),
                       mock.patch.object(recover, 'ready', side_effect=lambda *args: calls.append(('ready', *args))),
                       mock.patch.object(recover, 'save_json', side_effect=lambda *args: calls.append(('save', *args)))]
            for patch in patches:
                patch.start()
            try:
                if failure:
                    with self.assertRaisesRegex(RuntimeError, 'rollback incomplete'):
                        recover.rollback(work, state)
                else:
                    recover.rollback(work, state)
            finally:
                for patch in reversed(patches):
                    patch.stop()
            self.assertEqual(database.read_bytes(), b'new user writes must survive')
            self.assertIn(('run', 'systemctl', 'start', 'accord.service'), calls)
            self.assertIn(('ready', 7892), calls)
            self.assertIn(('run', 'systemctl', 'daemon-reload'), calls)
            self.assertIn(('write', root / 'unit', b'old unit'), calls)
            self.assertIn(('switch', root / 'go', state['old_go']), calls)
            self.assertIn(('run', 'systemctl', 'disable', 'accord-next.service'), calls)
            return state

    def test_successful_rollback_preserves_database(self):
        self.assertEqual(self.rollback_fixture()['phase'], 'rolled-back')

    def test_nginx_restore_failure_still_restores_writer_unit_and_reports_failure(self):
        state = self.rollback_fixture('nginx')
        self.assertEqual(state['phase'], 'rollback-incomplete')
        self.assertEqual(state['rollback_errors'], ['nginx file: OSError'])


if __name__ == '__main__':
    unittest.main()
