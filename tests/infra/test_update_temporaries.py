"""Real killed-write leftovers: only complete expected bytes may be resumed."""
from pathlib import Path
import os
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'infra/accord/projects'))
from host_ops import host


class UpdateTemporaries(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def test_complete_expected_private_atomic_file_resumes(self):
        target = self.root / 'state.json'
        temporary = self.root / '.state.json.next-new'
        temporary.write_bytes(b'expected complete state')
        temporary.chmod(0o600)
        with patch.object(host, 'regular', return_value=temporary.stat()):
            host.atomic(target, b'expected complete state', 0o600)
        self.assertEqual(target.read_bytes(), b'expected complete state')
        self.assertFalse(temporary.exists())

    def test_unknown_or_incomplete_atomic_file_is_preserved(self):
        target = self.root / 'guard.py'
        temporary = self.root / '.guard.py.next-new'
        temporary.write_bytes(b'incomplete')
        temporary.chmod(0o644)
        with patch.object(host, 'regular', return_value=temporary.stat()):
            with self.assertRaisesRegex(RuntimeError, 'inspection'):
                host.atomic(target, b'expected complete helper')
        self.assertEqual(temporary.read_bytes(), b'incomplete')
        self.assertFalse(target.exists())

    def test_expected_release_symlink_resumes_unknown_target_is_preserved(self):
        temporary = self.root / '.current.next-new'
        expected = 'releases/' + 'a' * 40
        temporary.symlink_to(expected)
        info = temporary.lstat()
        # Source requires root owner; the test filesystem belongs to this user.
        with patch.object(host.os, 'getuid', return_value=0):
            if os.geteuid() != 0:
                with patch.object(Path, 'lstat', lambda path: type('Info', (), {
                        'st_mode': info.st_mode, 'st_uid': 0})() if path == temporary else os.lstat(path)):
                    host.switch(self.root, expected)
            else:
                host.switch(self.root, expected)
        self.assertEqual(os.readlink(self.root / 'current'), expected)
        temporary.symlink_to('releases/' + 'b' * 40)
        with self.assertRaises(RuntimeError):
            host.switch(self.root, expected)
        self.assertEqual(os.readlink(temporary), 'releases/' + 'b' * 40)
