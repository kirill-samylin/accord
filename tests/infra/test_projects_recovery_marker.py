"""Never erase a manual inspection marker while recovering a project cutover."""
from pathlib import Path
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/projects'))
import startup_hold as recovery


class RecoveryMarker(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.revision = 'a' * 40
        self.marker = self.root / 'START_REQUIRES_INSPECTION'

    def test_owned_hold_is_idempotent_and_release_requires_exact_owner(self):
        recovery.hold_startup(self.root, self.revision)
        original = self.marker.read_bytes()
        self.assertEqual(self.marker.stat().st_mode & 0o777, 0o600)
        recovery.hold_startup(self.root, self.revision)
        self.assertEqual(self.marker.read_bytes(), original)
        with self.assertRaises(RuntimeError):
            recovery.release_startup(self.root, 'b' * 40)
        self.assertEqual(self.marker.read_bytes(), original)
        recovery.release_startup(self.root, self.revision)
        self.assertFalse(self.marker.exists())

    def test_unknown_marker_or_symlink_is_preserved(self):
        self.marker.write_bytes(b'operator inspection')
        self.marker.chmod(0o600)
        for call in (recovery.hold_startup, recovery.release_startup):
            with self.assertRaises(RuntimeError):
                call(self.root, self.revision)
            self.assertEqual(self.marker.read_bytes(), b'operator inspection')
        self.marker.unlink()
        self.marker.symlink_to(self.root / 'missing')
        with self.assertRaises(RuntimeError):
            recovery.hold_startup(self.root, self.revision)
        self.assertTrue(self.marker.is_symlink())
