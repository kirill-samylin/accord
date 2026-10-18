"""Runtime evidence must survive failures without exposing child output."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from backup_process_support import Evidence


class RuntimeTest(unittest.TestCase):
    def evidence(self, directory, timeout=2):
        output = Path(directory) / 'result.json'
        return Evidence(output, timeout, {'cases': [], 'processes': []})

    def test_failure_preserves_prior_case_and_child_exit(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = self.evidence(directory)
            evidence.case('prior', lambda: {'checked': True})
            with self.assertRaisesRegex(AssertionError, '^unexpected_exit:failing_child$'):
                evidence.case('failure', lambda: evidence.command('failing_child', [sys.executable, '-c', 'raise SystemExit(7)']))
            saved = json.loads(evidence.output.read_text())
            self.assertEqual(saved['counts'], {'PASS': 1, 'FAIL': 1, 'NOT-RUN': 0})
            self.assertEqual(saved['processes'][0]['exit_code'], 7)
            self.assertEqual(saved['cases'][1]['reason'], 'unexpected_exit:failing_child')

    def test_hidden_output_is_discarded_before_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = self.evidence(directory)
            out, err = evidence.command('private_child', [sys.executable, '-c', 'print("private-sentinel")'], hidden=True)
            self.assertEqual((out, err), (b'', b''))
            self.assertNotIn('private-sentinel', evidence.output.read_text())
            saved = json.loads(evidence.output.read_text())
            self.assertTrue(saved['processes'][0]['hidden'])
            self.assertEqual(saved['processes'][0]['stdout_bytes'], 0)
            self.assertEqual(evidence.output.stat().st_mode & 0o777, 0o600)

    def test_timeout_reaps_only_started_child_and_records_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = self.evidence(directory, timeout=0.05)
            with self.assertRaises(subprocess.TimeoutExpired):
                evidence.case('timeout', lambda: evidence.command('timed_child', [sys.executable, '-c', 'import time; time.sleep(10)']))
            saved = json.loads(evidence.output.read_text())
            child = saved['processes'][0]
            self.assertEqual(child['status'], 'INTERRUPTED')
            self.assertIsInstance(child['exit_code'], int)
            self.assertEqual(saved['cases'][0]['error_type'], 'TimeoutExpired')
            self.assertEqual(saved['counts']['FAIL'], 1)

    def test_existing_report_is_never_overwritten(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = self.evidence(directory)
            before = evidence.output.read_bytes()
            with self.assertRaises(FileExistsError):
                self.evidence(directory)
            self.assertEqual(evidence.output.read_bytes(), before)


if __name__ == '__main__':
    unittest.main()
