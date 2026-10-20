"""Update ordering/failure units. Real Linux process rehearsal is separate QA."""
from contextlib import ExitStack, nullcontext
import unittest
from pathlib import Path
import sys
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'infra/accord/projects'))
import upgrade


class CapableUpdate(unittest.TestCase):
    def setUp(self):
        self.state = {'kind': 'capable-update', 'revision': 'c' * 40, 'phase': 'staged',
                      'helpers': {}, 'go_sha256': 'a' * 64}
        self.work = Path('/synthetic/update')

    def run_update(self, fail=None, resume=False):
        calls = []

        def action(name, result=None):
            def invoke(*_args, **_kwargs):
                calls.append(name)
                if name == fail:
                    raise RuntimeError('synthetic failure')
                return result
            return invoke

        with ExitStack() as stack:
            for name in ('check', 'port_free', 'require_helpers', 'require_runtime', 'require_capable', 'require_process'):
                stack.enter_context(patch.object(upgrade, name, side_effect=action(name)))
            stack.enter_context(patch.object(upgrade, 'lock', return_value=nullcontext()))
            for name in ('hold_startup', 'release_startup', 'install_helpers'):
                stack.enter_context(patch.object(upgrade, name, side_effect=action(name)))
            stack.enter_context(patch.object(upgrade, 'snapshot', side_effect=action('snapshot', Path('/var/backups/accord-web/projects-synthetic'))))
            stack.enter_context(patch.object(upgrade, 'validate_snapshot', side_effect=action('validate_snapshot', {})))
            stack.enter_context(patch.object(upgrade, 'migrate_storage', side_effect=action('migrate_storage', {'logical_rows_preserved': True})))
            stack.enter_context(patch.object(upgrade, 'owned_listener', return_value=432))
            for name in ('save_json', 'ready'):
                stack.enter_context(patch.object(upgrade.host, name, side_effect=action(name)))
            stack.enter_context(patch.object(upgrade.host, 'sha', return_value='d' * 64))
            stack.enter_context(patch.object(upgrade.host, 'switch', side_effect=lambda _base, target: calls.append(('switch', target))))
            stack.enter_context(patch.object(upgrade.host, 'run', side_effect=lambda *args: calls.append(args)))
            stack.enter_context(patch.object(upgrade, 'fail_closed', side_effect=lambda _work, state: state.update(phase='blocked-roll-forward')))
            if fail:
                with self.assertRaisesRegex(RuntimeError, 'synthetic'):
                    upgrade.apply(self.work, self.state, resume)
            else:
                upgrade.apply(self.work, self.state, resume)
        return calls

    def test_snapshot_and_guard_precede_migration_no_bootstrap_or_data_rollback(self):
        calls = self.run_update()
        order = ['hold_startup', ('systemctl', 'stop', 'accord.service'), 'snapshot',
                 'install_helpers', 'migrate_storage', 'release_startup',
                 ('systemctl', 'start', 'accord.service')]
        self.assertEqual(sorted(order, key=calls.index), order)
        self.assertEqual(self.state['phase'], 'pending-https-smoke')
        self.assertTrue(self.state['storage_migrated'])
        self.assertNotIn('bootstrap-owner', str(calls))
        self.assertNotIn('maintenance-reset', str(calls))
        self.assertNotIn('rollback', str(calls))

    def test_every_cutover_failure_blocks_and_never_restores_old_data(self):
        for name in ('hold_startup', 'snapshot', 'install_helpers', 'migrate_storage', 'release_startup'):
            with self.subTest(failure=name):
                self.state.update(phase='staged')
                self.state.pop('pre_snapshot', None)
                self.state.pop('storage_migrated', None)
                calls = self.run_update(fail=name)
                self.assertEqual(self.state['phase'], 'blocked-roll-forward')
                self.assertNotIn('maintenance-reset', str(calls))

    def test_interrupted_resume_uses_original_snapshot_and_no_second_migration_after_success(self):
        self.state.update(phase='activation-started', storage_migrated=True,
                          pre_snapshot='/var/backups/accord-web/projects-synthetic',
                          pre_manifest_sha256='d' * 64)
        calls = self.run_update(resume=True)
        self.assertNotIn('snapshot', calls)
        self.assertNotIn('migrate_storage', calls)
        self.assertIn('install_helpers', calls)

    def test_other_phase_never_touches_runtime(self):
        for phase in ('complete', 'pending-https-smoke', 'staging'):
            for resume in (True, False):
                self.state['phase'] = phase
                with self.subTest(phase=phase, resume=resume), patch.object(upgrade, 'check') as check:
                    with self.assertRaises(RuntimeError):
                        upgrade.apply(self.work, self.state, resume)
                    check.assert_not_called()
