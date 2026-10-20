"""Effective launch regression: a capable executable may still run legacy mode."""
import hashlib
from pathlib import Path
import sys
import unittest
from unittest.mock import Mock, patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
import launch
import host_ops


def property_value(argv, ignored='no'):
    return '{ path=' + argv[0] + ' ; argv[]=' + ' '.join(argv) + ' ; ignore_errors=' + ignored + ' ; pid=0 ; }'


class EffectiveLaunch(unittest.TestCase):
    def runner(self, argv=None, guards=None):
        return Mock(side_effect=[property_value(argv or launch.ARGV),
                                 property_value(launch.GUARD) if guards is None else guards])

    def test_later_web_dropin_overrides_90_but_new_guard_is_last(self):
        current = (ROOT / 'infra/accord/projects/accord-projects.conf').read_text()
        legacy = current.replace(' --unit accord.service', '')
        web = '[Service]\nExecStart=\nExecStart=' + ' '.join(launch.ARGV[:-2]) + '\n'

        def resolved(files):
            command, pre = None, []
            for name in sorted(files):
                for line in files[name].splitlines():
                    if line.startswith('ExecStart='):
                        command = line.removeprefix('ExecStart=') or None
                    if line.startswith('ExecStartPre='):
                        pre.append(line.removeprefix('ExecStartPre='))
            return Mock(side_effect=[property_value(command.split()),
                                     ' '.join(property_value(args.split()) for args in pre)])

        with self.assertRaises(RuntimeError):
            launch.require_launch(resolved({'90-projects.conf': legacy, 'web.conf': web}))
        self.assertGreater(host_ops.DROPIN.name, 'web.conf')
        launch.require_launch(resolved({'web.conf': web, host_ops.DROPIN.name: current}))

    def test_exact_command_and_required_guard_pass(self):
        launch.require_launch(self.runner())

    def test_missing_extra_duplicate_or_alternate_arguments_fail(self):
        for argv in (launch.ARGV[:-2], launch.ARGV + ['--projects-data', launch.PROJECTS],
                     launch.ARGV + ['--db', '/other.db'], ['/other/agent-board'] + launch.ARGV[1:]):
            with self.subTest(argv=argv), self.assertRaises(RuntimeError):
                launch.require_launch(self.runner(argv))
        for text in (property_value(launch.ARGV) * 2, property_value(launch.ARGV, 'yes'), ''):
            with self.subTest(text=text), self.assertRaises(RuntimeError):
                launch.require_launch(Mock(return_value=text))

    def test_absent_or_ignored_startup_guard_fails(self):
        alternate = property_value(launch.GUARD).replace('path=/usr/bin/python3', 'path=/bin/true')
        for guards in ('', property_value(launch.GUARD, 'yes'), property_value(launch.GUARD[:-2]), alternate):
            with self.subTest(guards=guards), self.assertRaises(RuntimeError):
                launch.require_launch(self.runner(guards=guards))

    def test_effective_backup_cannot_be_overridden_to_legacy(self):
        expected = ['/usr/bin/python3', '/opt/accord/admin/projects/backup.py']
        launch.require_backup(Mock(return_value=property_value(expected)))
        for argv in (expected + ['--other'], [expected[0], '/opt/accord/admin/web-backup.py']):
            with self.subTest(argv=argv), self.assertRaises(RuntimeError):
                launch.require_backup(Mock(return_value=property_value(argv)))

    def test_owned_process_requires_same_argv_binary_and_stable_pid(self):
        executable = b'synthetic executable'
        digest = hashlib.sha256(executable).hexdigest()
        raw = b'\0'.join(arg.encode() for arg in launch.ARGV) + b'\0'
        read = lambda path: raw if path.name == 'cmdline' else executable
        launch.require_process(123, digest, Mock(return_value='123\n'), read)
        for pid, sha, run, reader in (
                (0, digest, Mock(return_value='123'), read),
                (123, digest, Mock(return_value='456'), read),
                (123, '0' * 64, Mock(return_value='123'), read),
                (123, digest, Mock(side_effect=['123', '456']), read),
                (123, digest, Mock(return_value='123'), lambda path: b'legacy\0')):
            with self.subTest(pid=pid, sha=sha[:4]), self.assertRaises(RuntimeError):
                launch.require_process(pid, sha, run, reader)

    def test_unknown_existing_owned_override_is_not_replaced(self):
        for selected in (host_ops.DROPIN, host_ops.LEGACY_DROPIN):
            with self.subTest(selected=selected), patch.object(Path, 'exists', lambda path: path == selected), \
                    patch.object(Path, 'is_symlink', return_value=False), \
                    patch.object(Path, 'read_bytes', side_effect=lambda path=None: b'unknown'), \
                    patch.object(host_ops.host, 'regular'), patch.object(host_ops.host, 'atomic') as atomic:
                # Source reads also need a distinct known template; patch HERE only.
                template = Mock()
                template.__truediv__ = Mock(return_value=Mock(read_bytes=lambda: b'known template'))
                with patch.object(host_ops, 'HERE', template), self.assertRaises(RuntimeError):
                    host_ops.install_helpers({})
                atomic.assert_not_called()
