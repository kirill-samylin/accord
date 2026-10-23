"""Capability guard and ingress unit contracts; no host or service operations."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
import guard


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, ROOT / path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


public = load('project_public_render', 'infra/accord/public/render_public.py')
next_render = load('project_next_render', 'infra/accord/next/render.py')
render = load('project_render', 'infra/accord/projects/ingress.py')


class ProjectGuard(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.folder = Path(temporary.name).resolve()
        self.binary = self.folder / 'binary'
        self.binary.write_bytes(b'synthetic fixture')
        self.binary.chmod(0o700)
        self.root = self.folder / 'projects-data'

    def response(self, value, stderr=b''):
        return type('Result', (), {'stdout': json.dumps(value).encode(), 'stderr': stderr})()

    def test_guard_requires_capability_before_registry_activation(self):
        value = {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1],
                 'agent_profiles': 1, 'coordination': 1, 'deduplication': 1, 'managed_agents': 1, 'automation': 1, 'board_schema': [6]}
        with patch.object(guard.subprocess, 'run', return_value=self.response(value)) as run:
            self.assertEqual(guard.require_capable(self.binary, self.root), value)
            self.assertEqual(run.call_args.args[0], [str(self.binary), 'capabilities', '--json'])
            self.assertEqual(run.call_args.kwargs['timeout'], 5)
        self.assertFalse(self.root.exists())

    def test_old_registry_capable_binary_and_invalid_profile_floor_are_denied(self):
        current = {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1],
                   'agent_profiles': 1, 'coordination': 1, 'deduplication': 1, 'managed_agents': 1, 'automation': 1, 'board_schema': [6]}
        for change in ({'agent_profiles': None}, {'agent_profiles': True},
                       {'agent_profiles': 0}, {'board_schema': [2]},
                       {'board_schema': [True]}, {'board_schema': [3]},
                       {'board_schema': [3, 4]}, {'coordination': None},
                       {'coordination': True}, {'coordination': 0},
                       {'board_schema': [4]}, {'board_schema': [4, 5]},
                       {'managed_agents': None}, {'managed_agents': True},
                       {'managed_agents': 0}, {'managed_agents': '1'},
                       {'managed_agents': 2}, {'automation': None}, {'automation': True},
                       {'automation': 0}, {'automation': '1'}, {'automation': 2},
                       {'board_schema': [5]}, {'board_schema': [5, 6]}):
            with self.subTest(change=change), patch.object(guard.subprocess, 'run',
                    return_value=self.response(current | change)):
                with self.assertRaises(RuntimeError):
                    guard.require_capable(self.binary, self.root)

    def test_schema5_binary_without_deduplication_is_denied_before_any_write(self):
        current = {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1],
                   'agent_profiles': 1, 'coordination': 1, 'managed_agents': 1, 'automation': 1, 'board_schema': [6]}
        for value in (current, current | {'deduplication': None},
                      current | {'deduplication': True}, current | {'deduplication': 0},
                      current | {'deduplication': '1'}, current | {'deduplication': 2}):
            with self.subTest(value=value), patch.object(guard.subprocess, 'run',
                    return_value=self.response(value)):
                with self.assertRaises(RuntimeError):
                    guard.require_capable(self.binary, self.root)
                self.assertFalse(self.root.exists())

    def test_capability_check_preserves_service_authority_marker(self):
        self.root.mkdir(mode=0o700)
        marker = self.root / guard.MARKER
        marker.write_bytes(b'synthetic service-owned authority\n')
        marker.chmod(0o600)
        before = marker.stat()
        content = marker.read_bytes()
        current = {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1],
                   'agent_profiles': 1, 'coordination': 1, 'deduplication': 1, 'managed_agents': 1, 'automation': 1, 'board_schema': [6]}
        for value in (current, current | {'deduplication': 0}, current | {'managed_agents': 0}):
            with self.subTest(value=value), patch.object(guard.subprocess, 'run',
                    return_value=self.response(value)):
                if value['deduplication'] == 1 and value['managed_agents'] == 1:
                    guard.require_capable(self.binary, self.root)
                else:
                    with self.assertRaises(RuntimeError):
                        guard.require_capable(self.binary, self.root)
            after = marker.stat()
            self.assertEqual(marker.read_bytes(), content)
            self.assertEqual((after.st_uid, after.st_gid, after.st_mode, after.st_ino,
                              after.st_mtime_ns, after.st_ctime_ns),
                             (before.st_uid, before.st_gid, before.st_mode, before.st_ino,
                              before.st_mtime_ns, before.st_ctime_ns))

    def test_marker_loss_never_allows_legacy_binary(self):
        self.root.mkdir(mode=0o700)
        registry = self.root / 'registry.sqlite'
        registry.write_bytes(b'synthetic')
        registry.chmod(0o600)
        for response in ({}, {'project_registry': 1, 'project_acl': 0, 'registry_schema': [1]},
                         {'project_registry': True, 'project_acl': 1, 'registry_schema': [1]},
                         {'project_registry': 1, 'project_acl': 1, 'registry_schema': [True]},
                         {'project_registry': 1, 'project_acl': 1, 'registry_schema': [2]}):
            with self.subTest(response=response), patch.object(guard.subprocess, 'run', return_value=self.response(response)):
                with self.assertRaises(RuntimeError):
                    guard.require_capable(self.binary, self.root)
        with patch.object(guard.subprocess, 'run', side_effect=subprocess.CalledProcessError(1, 'synthetic')):
            with self.assertRaises(subprocess.SubprocessError):
                guard.require_capable(self.binary, self.root)

    def test_unsafe_storage_and_binary_refused(self):
        self.root.mkdir(mode=0o755)
        self.root.chmod(0o755)
        with self.assertRaises(RuntimeError):
            guard.require_capable(self.binary, self.root)
        self.root.chmod(0o700)
        marker = self.root / guard.MARKER
        marker.symlink_to(self.binary)
        with self.assertRaises(RuntimeError):
            guard.require_capable(self.binary, self.root)
        marker.unlink()
        self.binary.chmod(0o777)
        with self.assertRaises(RuntimeError):
            guard.require_capable(self.binary, self.root)

    def test_dropin_pins_scope_and_keeps_capability_check(self):
        source = (ROOT / 'infra/accord/projects/accord-projects.conf').read_text()
        self.assertIn('ExecStartPre=/usr/bin/python3 /opt/accord/admin/projects/guard.py', source)
        self.assertIn('ExecStart=\nExecStart=', source)
        self.assertIn('serve --db /var/lib/accord/board.db', source)
        self.assertIn('--projects-data /var/lib/accord/projects-data', source)
        self.assertNotIn('ExecStartPre=\n', source)
        self.assertNotIn('ACCORD_PROJECTS_DATA', source)


class ProjectIngress(unittest.TestCase):
    def setUp(self):
        self.before = next_render.render(public.render('1.1.1.1', 'tls', web=True), 7893)
        self.after = render.render(self.before)

    def test_existing_bytes_preserved_except_two_bounded_additions(self):
        start = self.after.index(render.SCOPED_UPLOAD)
        end = self.after.index('\n    }', start) + len('\n    }')
        reverted = (self.after[:start - 1] + self.after[end:]).replace(render.SCOPED_NATIVE, render.NATIVE)
        self.assertEqual(reverted, self.before)
        upload = self.after[start:end]
        self.assertIn('client_max_body_size 52494336;', upload)
        self.assertIn('proxy_set_header X-Accord-Client-IP $remote_addr;', upload)
        self.assertNotIn('proxy_set_header Origin', upload)
        self.assertNotIn('proxy_set_header Authorization', upload)
        self.assertIn('mcp(?:$|/p/p_[0-9a-f]{32}$)', self.after)
        self.assertIn('location ~ "^/web/p/p_[0-9a-f]{32}/designs$"', self.after)

    def test_repeated_or_unknown_source_cannot_silently_widen_ingress(self):
        for source in (self.after, self.before.replace('52494336', '100m'),
                       self.before.replace(render.NATIVE, ''),
                       self.before.replace('limit_req zone=accord_next_assets burst=100 nodelay;', '')):
            with self.subTest(source=source[:20]), self.assertRaises(ValueError):
                render.render(source)
