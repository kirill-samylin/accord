"""Cutover failure units: mocks only, never invoke real host commands."""
from contextlib import nullcontext
import argparse
import importlib.util
from pathlib import Path
import sys
import socket
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
spec = importlib.util.spec_from_file_location('project_cutover', ROOT / 'infra/accord/projects/operate.py')
operator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(operator)
host_ops = sys.modules['host_ops']


class ProjectsOperations(unittest.TestCase):
    def setUp(self):
        for name in ('hold_startup', 'release_startup'):
            mocked = patch.object(operator, name)
            mocked.start()
            self.addCleanup(mocked.stop)
        self.work = Path('/synthetic/private-journal')
        self.state = {'revision': 'a' * 40, 'phase': 'staged', 'helpers': {}, 'owner_identity': 'synthetic-public-id'}

    def test_failure_never_selects_legacy_or_restores_database(self):
        switched = []
        calls = []

        def run(*args):
            calls.append(args)
            if args == ('systemctl', 'start', 'accord.service'):
                raise RuntimeError('synthetic readiness failure')

        with patch.object(operator, 'validate'), patch.object(operator, 'lock', return_value=nullcontext()), \
                patch.object(operator, 'install_helpers'), patch.object(operator, 'require_capable'), \
                patch.object(operator, 'port_free'), patch.object(operator, 'owned_listener'), \
                patch.object(operator, 'require_launch'), patch.object(operator, 'require_backup'), \
                patch.object(operator, 'require_process'), \
                patch.object(operator.host, 'run', side_effect=run), patch.object(operator.host, 'ready'), \
                patch.object(operator.host, 'save_json'), \
                patch.object(operator.host, 'switch', side_effect=lambda base, target: switched.append(target)):
            with self.assertRaises(RuntimeError):
                operator.apply(self.work, self.state)
        self.assertEqual(self.state['phase'], 'blocked-roll-forward')
        self.assertEqual(switched, ['releases/' + 'a' * 40] * 2)
        self.assertIn(('systemctl', 'stop', 'accord.service'), calls)
        self.assertIn(('systemctl', 'stop', 'accord-next.service'), calls)
        self.assertNotIn('rollback', str(calls))

    def test_killed_activation_can_resume_without_old_backup(self):
        for phase in ('activation-started', 'blocked-roll-forward'):
            self.state['phase'] = phase
            with self.subTest(phase=phase), patch.object(operator, 'validate'), \
                    patch.object(operator, 'lock', return_value=nullcontext()), patch.object(operator, 'port_free'), \
                    patch.object(operator, 'install_helpers', side_effect=RuntimeError('synthetic stop after validate')), \
                    patch.object(operator.host, 'run') as run, patch.object(operator.host, 'save_json'):
                with self.assertRaisesRegex(RuntimeError, 'synthetic'):
                    operator.apply(self.work, self.state, resume=True)
                self.assertNotIn(('systemctl', 'start', 'accord-web-backup.service'),
                                 [call.args for call in run.call_args_list])
                self.assertEqual(self.state['phase'], 'blocked-roll-forward')

    def test_stop_failure_is_recorded_and_other_stop_is_still_attempted(self):
        calls = []

        def run(*args):
            calls.append(args)
            if args[-1] == 'accord.service':
                raise RuntimeError('synthetic stop failure')

        with patch.object(operator.host, 'run', side_effect=run), patch.object(operator.host, 'save_json'):
            operator.fail_closed(self.work, self.state)
        self.assertEqual(self.state['stop_failures'], ['accord.service'])
        self.assertEqual(len(calls), 2)
        self.assertEqual(self.state['phase'], 'blocked-roll-forward')

    def test_failed_cutover_holds_cold_boot_before_stopping_processes(self):
        calls = []
        with patch.object(operator, 'hold_startup', side_effect=lambda *_: calls.append('guard')), \
                patch.object(operator.host, 'run', side_effect=lambda *args: calls.append(args)), \
                patch.object(operator.host, 'save_json'):
            operator.fail_closed(self.work, self.state)
        self.assertEqual(calls[0], 'guard')
        self.assertEqual(len(calls), 3)

    def test_non_staged_apply_and_non_failed_resume_are_refused(self):
        for phase in ('complete', 'pending-https-smoke', 'staging'):
            for resume in (True, False):
                self.state['phase'] = phase
                with self.subTest(phase=phase, resume=resume), patch.object(operator, 'validate') as validate:
                    with self.assertRaises(RuntimeError):
                        operator.apply(self.work, self.state, resume=resume)
                    validate.assert_not_called()

    def test_owner_parameter_cannot_store_bearer_as_identity(self):
        self.assertEqual(operator.owner_identity('a' * 64), 'a' * 64)
        for invalid in ('ab_' + 'a' * 64, 'owner', '', 'A' * 64):
            with self.subTest(invalid=invalid[:3]), self.assertRaises(argparse.ArgumentTypeError):
                operator.owner_identity(invalid)

    def test_ready_port_must_belong_to_owned_main_pid(self):
        good = 'LISTEN 0 128 127.0.0.1:7894 0.0.0.0:* users:(("node",pid=321,fd=1))'
        with patch.object(host_ops.host, 'run', side_effect=['321\n', good]):
            self.assertEqual(host_ops.owned_listener(7894, 'synthetic.service'), 321)
        for pid, lines in [('0', good), ('321', good.replace('pid=321,', 'pid=999,')),
                           ('321', good + '\n' + good), ('321', ''),
                           ('321', good + ',("other",pid=999,fd=2)'),
                           ('321', good.replace('127.0.0.1:', '0.0.0.0:'))]:
            with self.subTest(pid=pid, listeners=len(lines)), patch.object(host_ops.host, 'run', side_effect=[pid, lines]):
                with self.assertRaises(RuntimeError):
                    host_ops.owned_listener(7894, 'synthetic.service')

    def test_occupied_port_rejected_before_starting_candidate(self):
        with socket.socket() as existing:
            existing.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            existing.bind(('127.0.0.1', 0))
            existing.listen()
            port = existing.getsockname()[1]
            with self.assertRaises(OSError):
                host_ops.port_free(port)
        host_ops.port_free(port)
