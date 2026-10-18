"""Unit checks of readiness budgets and schema-aware guards; no real systemd."""
import argparse
import pathlib
import sys
import tempfile
import time
import unittest
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord'))
import install
import recovery


class PerformanceReadinessTests(unittest.TestCase):
    def test_operator_migration_budget_is_bounded(self):
        self.assertEqual(install.migration_timeout('1'), 1)
        self.assertEqual(install.migration_timeout('900'), 900)
        for value in ('0', '-1', '901'):
            with self.assertRaises(argparse.ArgumentTypeError):
                install.migration_timeout(value)

    def test_only_prior_schema_uses_migration_checker(self):
        for version in (3, 4):
            regular, migration, command = mock.Mock(return_value=True), mock.Mock(return_value=True), mock.Mock()
            with mock.patch.object(recovery, 'check_guard'), mock.patch.object(
                    recovery, 'schema_snapshot', return_value=(version, ())):
                recovery.activate('releases/' + 'a' * 40, 'b' * 40, pathlib.Path('/synthetic'),
                                  command, mock.Mock(), regular, migration)
            self.assertEqual(migration.call_count, int(version == 3))
            self.assertEqual(regular.call_count, int(version == 4))

    def test_failed_migration_does_not_roll_back_across_schema(self):
        with tempfile.TemporaryDirectory() as directory:
            state = pathlib.Path(directory)
            state.chmod(0o700)
            switch, command = mock.Mock(), mock.Mock()
            with mock.patch.object(recovery, 'schema_snapshot', side_effect=[(3, ()), (4, ())]):
                with self.assertRaisesRegex(RuntimeError, 'persistent inspection guard set'):
                    recovery.activate('releases/' + 'a' * 40, 'b' * 40, state, command,
                                      switch, mock.Mock(return_value=True), mock.Mock(return_value=False))
            self.assertTrue((state / 'START_REQUIRES_INSPECTION').exists())
            self.assertEqual(switch.call_args_list, [mock.call('b' * 40)])

    def test_http_failure_and_dead_service_fail_fast(self):
        client = mock.Mock()
        client.open.side_effect = OSError('unavailable')
        with mock.patch.object(recovery.urllib.request, 'build_opener', return_value=client), \
                mock.patch.object(recovery, 'service_active', return_value=False), \
                mock.patch.object(recovery.time, 'sleep') as sleep:
            self.assertFalse(recovery.healthy(timeout=300))
            sleep.assert_not_called()

    def test_running_but_unready_service_has_finite_deadline(self):
        client = mock.Mock()
        client.open.side_effect = OSError('unavailable')
        before = time.monotonic()
        with mock.patch.object(recovery.urllib.request, 'build_opener', return_value=client), \
                mock.patch.object(recovery, 'service_active', return_value=True):
            self.assertFalse(recovery.healthy(timeout=0.01))
        self.assertLess(time.monotonic() - before, 1)
