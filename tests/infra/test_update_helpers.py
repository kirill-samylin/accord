"""Installed operator trust follows the completed prior release, never adoption."""
from contextlib import ExitStack
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'infra/accord/projects'))
import upgrade_verify as verify


class UpdateHelpers(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.admin, self.work = self.root / 'admin', self.root / 'work'
        (self.admin / 'web').mkdir(parents=True)
        (self.admin / 'projects').mkdir()
        self.revision = 'b' * 40
        (self.work / self.revision).mkdir(parents=True)
        (self.admin / 'web/backup.py').write_bytes(b'reviewed-backup')
        (self.admin / 'web-backup.py').write_bytes(b'reviewed-backup')
        (self.admin / 'projects/guard.py').write_bytes(b'reviewed-guard')
        self.prior = {name: hashlib.sha256((self.admin / name).read_bytes()).hexdigest()
                      for name in ('web/backup.py', 'projects/guard.py')}
        self.journal = self.work / self.revision / 'state.json'
        self.write_journal('complete')
        self.candidate = self.prior | {'projects/new.py': 'c' * 64}

    def write_journal(self, phase):
        self.journal.write_text(json.dumps({'revision': self.revision, 'phase': phase, 'helpers': self.prior}))

    def baseline(self):
        with ExitStack() as stack:
            stack.enter_context(patch.object(verify, 'ADMIN', self.admin))
            stack.enter_context(patch.object(verify, 'WORK', self.work))
            stack.enter_context(patch.object(verify.host, 'regular'))
            stack.enter_context(patch.object(verify.host, 'pointer', return_value='releases/' + self.revision))
            return verify.baseline_helpers(self.candidate)

    def test_completed_prior_helpers_match_and_new_missing_candidate_is_allowed(self):
        result = self.baseline()
        self.assertEqual(result['projects/guard.py'], self.prior['projects/guard.py'])
        self.assertNotIn('projects/new.py', result)
        self.assertEqual(result['web-backup.py'], self.prior['web/backup.py'])

    def test_drift_is_not_silently_adopted(self):
        (self.admin / 'projects/guard.py').write_bytes(b'unknown drift')
        with self.assertRaisesRegex(RuntimeError, 'completed previous'):
            self.baseline()

    def test_unknown_new_helper_collision_is_preserved(self):
        path = self.admin / 'projects/new.py'
        path.write_bytes(b'unknown pre-existing root file')
        with self.assertRaisesRegex(RuntimeError, 'new helper path'):
            self.baseline()
        self.assertEqual(path.read_bytes(), b'unknown pre-existing root file')

    def test_incomplete_prior_journal_is_rejected(self):
        self.write_journal('blocked-roll-forward')
        with self.assertRaisesRegex(RuntimeError, 'complete'):
            self.baseline()

    def test_unknown_backup_alias_is_preserved(self):
        alias = self.admin / 'web-backup.py'
        alias.write_bytes(b'unknown backup alias')
        with self.assertRaisesRegex(RuntimeError, 'alias'):
            self.baseline()
        self.assertEqual(alias.read_bytes(), b'unknown backup alias')

    def test_compiled_cache_is_rejected_before_helper_hash_adoption(self):
        cache = self.admin / 'projects/__pycache__'
        cache.mkdir()
        (cache / 'guard.cpython-fixture.pyc').write_bytes(b'synthetic compiled')
        with self.assertRaisesRegex(RuntimeError, 'compiled'):
            self.baseline()
