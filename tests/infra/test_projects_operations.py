"""Cutover failure units: mocks only, never invoke real host commands."""
from contextlib import nullcontext
import argparse
import importlib.util
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
spec = importlib.util.spec_from_file_location('project_cutover', ROOT / 'infra/accord/projects/operate.py')
operator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(operator)


class ProjectsOperations(unittest.TestCase):
    def setUp(self):
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
                    patch.object(operator, 'lock', return_value=nullcontext()), \
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
