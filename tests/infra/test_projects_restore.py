"""Offline restore orchestration units, with native reset boundary mocked."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

from test_projects_snapshot import ProjectsFixture, OTHER, registry1_copy, registry2

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('projects_offline_restore', ROOT / 'infra/accord/projects/restore.py')
restore = importlib.util.module_from_spec(spec)
spec.loader.exec_module(restore)


class ProjectsRestore(ProjectsFixture, unittest.TestCase):
    def setUp(self):
        super().setUp()
        self.output = self.root / 'offline-restored'
        self.binary = self.root / 'capable-binary'

    def test_new_staging_contains_all_projects_and_keeps_boot_guard(self):
        snapshot = self.snapshot()
        before = (snapshot / 'manifest.json').read_bytes()
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run, \
                patch.object(restore.LEGACY, 'sync_dir', wraps=restore.LEGACY.sync_dir) as sync:
            run.return_value.stdout = b'{"restored_state_reset":true}'
            run.return_value.stderr = b''
            restore.restore(snapshot, self.output, self.binary)
            arguments = run.call_args.args[0]
            self.assertEqual(arguments[:3], [str(self.binary), 'projects', 'maintenance-reset'])
            self.assertEqual(arguments[arguments.index('--projects-data') + 1], str(self.output / 'projects-data'))
            self.assertIn(str(self.output / 'board.db'), arguments)
            self.assertIn(str(self.output / 'web'), arguments)
            self.assertIn(self.output.parent, [call.args[0] for call in sync.call_args_list])
        self.assertTrue((self.output / 'RESTORE_REQUIRES_RESET').is_file())
        self.assertTrue((self.output / 'RESTORE_AUTH_RESET_COMPLETE').is_file())
        self.assertEqual((snapshot / 'manifest.json').read_bytes(), before)
        self.assertTrue((self.output / 'projects-data/projects' / OTHER / 'board.db').is_file())
        self.assertTrue((self.state / 'board.db').is_file())

    def test_registry2_copies_and_marker_follow_the_bump(self):
        copy = registry1_copy(self.projects)
        registry2(self.projects, copy=False)
        snapshot = self.snapshot()
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run:
            run.return_value.stdout, run.return_value.stderr = b'{"restored_state_reset":true}', b''
            restore.restore(snapshot, self.output, self.binary)
        restored = self.output / 'projects-data/registry.sqlite.migrations' / copy.name
        self.assertEqual(restored.read_bytes(), copy.read_bytes())
        self.assertEqual(restored.stat().st_mode & 0o777, 0o600)
        self.assertEqual(restored.parent.stat().st_mode & 0o777, 0o700)
        marker = json.loads((self.output / 'RESTORE_AUTH_RESET_COMPLETE').read_text())
        self.assertEqual((marker['registry_schema'], marker['people_admin'], marker['source_registry_schema']), (2, 1, 2))

    def test_registry1_snapshot_marker_records_source_schema(self):
        snapshot = self.snapshot()
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run:
            run.return_value.stdout, run.return_value.stderr = b'{"restored_state_reset":true}', b''
            restore.restore(snapshot, self.output, self.binary)
        marker = json.loads((self.output / 'RESTORE_AUTH_RESET_COMPLETE').read_text())
        self.assertEqual((marker['registry_schema'], marker['source_registry_schema']), (2, 1))

    def test_failed_reset_keeps_staging_blocked_without_complete_marker(self):
        snapshot = self.snapshot()
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run',
                side_effect=subprocess.CalledProcessError(1, 'synthetic')):
            with self.assertRaises(subprocess.CalledProcessError):
                restore.restore(snapshot, self.output, self.binary)
        self.assertTrue((self.output / 'RESTORE_REQUIRES_RESET').is_file())
        self.assertFalse((self.output / 'RESTORE_AUTH_RESET_COMPLETE').exists())
        self.assertTrue((snapshot / 'default/board.db').is_file())

    def test_live_existing_or_nested_output_is_rejected_before_any_mutation(self):
        snapshot = self.snapshot()
        for target in (self.state, snapshot, snapshot / 'nested', self.copies):
            with self.subTest(target=target.name), patch.object(restore, 'require_capable') as guard:
                with self.assertRaises(RuntimeError):
                    restore.restore(snapshot, target, self.binary)
                guard.assert_not_called()
        self.assertFalse(self.output.exists())

    def test_tampered_snapshot_never_reaches_native_reset(self):
        snapshot = self.snapshot()
        (snapshot / 'default/board.db').write_bytes(b'changed')
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run:
            with self.assertRaises(RuntimeError):
                restore.restore(snapshot, self.output, self.binary)
            run.assert_not_called()
        self.assertFalse(self.output.exists())

    def test_zero_exit_without_explicit_reset_confirmation_keeps_guard(self):
        snapshot = self.snapshot()
        with patch.object(restore, 'verify_restored'), patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run:
            run.return_value.stdout, run.return_value.stderr = b'{"restored_state_reset":false}', b''
            with self.assertRaisesRegex(RuntimeError, 'not confirmed'):
                restore.restore(snapshot, self.output, self.binary)
        self.assertTrue((self.output / 'RESTORE_REQUIRES_RESET').exists())
        self.assertFalse((self.output / 'RESTORE_AUTH_RESET_COMPLETE').exists())

    def test_reset_ack_without_verified_database_effects_keeps_staging_blocked(self):
        snapshot = self.snapshot()
        with patch.object(restore, 'require_capable'), patch.object(restore.subprocess, 'run') as run, \
                patch.object(restore, 'verify_restored', side_effect=RuntimeError('fixture unchanged authority')):
            run.return_value.stdout, run.return_value.stderr = b'{"restored_state_reset":true}', b''
            with self.assertRaisesRegex(RuntimeError, 'unchanged authority'):
                restore.restore(snapshot, self.output, self.binary)
        self.assertTrue((self.output / 'RESTORE_REQUIRES_RESET').exists())
        self.assertFalse((self.output / 'RESTORE_AUTH_RESET_COMPLETE').exists())
