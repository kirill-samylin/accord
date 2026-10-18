"""Runtime evidence must survive failures without exposing child output."""
import contextlib
import datetime as dt
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import unittest

from backup_process import Acceptance
from backup_process_support import (Evidence, KnownSnapshot, database, expected_at_interval,
                                    expected_retained, private_write, sha)


UTC = dt.timezone.utc


class RetentionTest(unittest.TestCase):
    now = dt.datetime(2026, 10, 18, 12, 0, 1, tzinfo=UTC)

    def record(self, name, when, mtime=1, size=10):
        return KnownSnapshot(name, when, mtime, size, 'synthetic-hash', 'second')

    def records(self, first):
        return [self.record('first', first, 1), self.record('second', self.now, 2),
                self.record('new', self.now, 3),
                self.record('hour-old', self.now - dt.timedelta(hours=2, seconds=1), 1),
                self.record('hour-new', self.now - dt.timedelta(hours=2), 2),
                self.record('day-old', self.now - dt.timedelta(days=3, hours=1)),
                self.record('day-new', self.now - dt.timedelta(days=3)),
                self.record('day-six', self.now - dt.timedelta(days=6)),
                self.record('expired', self.now - dt.timedelta(days=40))]

    def test_hour_rollover_preserves_first_origin_and_exact_six(self):
        records = self.records(self.now - dt.timedelta(seconds=2))
        self.assertEqual(expected_retained(records, self.now),
                         {'first', 'second', 'new', 'hour-new', 'day-new', 'day-six'})

    def test_same_hour_retires_first_and_exact_five(self):
        self.assertEqual(expected_retained(self.records(self.now), self.now),
                         {'second', 'new', 'hour-new', 'day-new', 'day-six'})

    def test_midnight_retires_old_seventh_date_and_brackets_plan(self):
        start = self.now.replace(hour=23, minute=59, second=59)
        end = start + dt.timedelta(seconds=1)
        records = [self.record('first', start, 1), self.record('second', start, 2),
                   self.record('day-six', start - dt.timedelta(days=6)),
                   self.record('day-seven', start - dt.timedelta(days=7))]
        self.assertEqual(expected_retained(records, start), {'first', 'second', 'day-six'})
        self.assertEqual(expected_retained(records, end), {'first', 'second'})
        self.assertEqual(expected_at_interval(records, start, end, 100, 0),
                         {frozenset({'first', 'second', 'day-six'}), frozenset({'first', 'second'})})
        records.append(self.record('new', end, 3))
        self.assertEqual(expected_retained(records, end), {'new', 'second'})

    def test_twenty_fourth_hour_expires_even_when_calendar_day_remains(self):
        start = self.now.replace(minute=59, second=59)
        records = [self.record('first', start, 1), self.record('second', start, 2),
                   self.record('old-hour', start - dt.timedelta(hours=23)),
                   self.record('newer-same-date', start - dt.timedelta(hours=22))]
        self.assertEqual(expected_retained(records, start), {'first', 'second', 'old-hour', 'newer-same-date'})
        self.assertEqual(expected_retained(records, start + dt.timedelta(seconds=1)),
                         {'first', 'second', 'newer-same-date'})

    def test_same_second_orders_mtime_before_name_then_name_for_exact_tie(self):
        records = [self.record('z-older-mtime', self.now, 1),
                   self.record('a', self.now, 2), self.record('b', self.now, 2)]
        self.assertEqual(expected_retained(records, self.now), {'a', 'b'})
        records[0] = self.record('z-older-mtime', self.now, 3)
        self.assertEqual(expected_retained(records, self.now), {'z-older-mtime', 'b'})

    def test_budget_removes_oldest_unprotected_and_never_latest_two(self):
        records = [self.record('first', self.now, 1), self.record('second', self.now, 2),
                   self.record('history', self.now - dt.timedelta(hours=1))]
        self.assertEqual(expected_retained(records, self.now, budget=25, estimate=5), {'first', 'second'})
        with self.assertRaisesRegex(AssertionError, '^oracle_budget_preserving_two$'):
            expected_retained(records, self.now, budget=24, estimate=5)

    def test_future_snapshots_only_fill_protected_slots(self):
        records = [self.record('current', self.now)] + [
            self.record('future-' + str(i), self.now + dt.timedelta(hours=i)) for i in (1, 2, 3)]
        self.assertEqual(expected_retained(records, self.now), {'current', 'future-2', 'future-3'})


class PruneMethodTest(unittest.TestCase):
    """Scripted file fixture tests method assertions, never stands in for process QA."""

    def scenario(self, directory, fault=None, midnight=False):
        now = RetentionTest.now.replace(hour=0, minute=0, second=0) if midnight else RetentionTest.now
        history = now - dt.timedelta(seconds=1) if midnight else now
        acceptance = Acceptance(None, Path(directory), None, clock=lambda: now)
        first, second = acceptance.work / 'first.db', acceptance.work / 'second.db'
        for path, count in ((first, 1), (second, 2)):
            with contextlib.closing(sqlite3.connect(path)) as db:
                db.execute('CREATE TABLE fixture(value INTEGER)')
                db.executemany('INSERT INTO fixture VALUES (?)', [(i,) for i in range(count)])
                db.commit()
            path.chmod(0o600)
        acceptance.images = {'first': database(first), 'second': database(second)}
        roles = {}

        def copy(role, created, origin, mtime):
            name = 'board-' + created.strftime('%Y%m%dT%H%M%SZ') + '-' + format(len(roles) + 1, '032x') + '.db'
            path = acceptance.backups / name
            shutil.copyfile(first if origin == 'first' else second, path)
            path.chmod(0o600)
            os.utime(path, ns=(mtime, mtime))
            private_write(path.with_name(name + '.sha256'), (sha(path) + '  ' + name + '\n').encode())
            roles[role] = path
            return path

        fixtures = [('first', now - dt.timedelta(seconds=2), 'first'), ('second', now, 'second'),
                    ('hour', history - dt.timedelta(hours=2), 'second'),
                    ('day', history - dt.timedelta(days=3), 'second'),
                    ('day-six', history - dt.timedelta(days=6), 'second'),
                    ('expired', history - dt.timedelta(days=40), 'second')]
        for index, (role, created, origin) in enumerate(fixtures):
            acceptance.declare(copy(role, created, origin, index + 1), origin)
        acceptance.unsafe, acceptance.unsafe_before = [], {}
        acceptance.plan = {'budget_bytes': 5 * 1024 ** 3, 'estimate_bytes': 16 * 1024 ** 2}

        def wrapper(label, plan=False, expected=0):
            if label == 'retention_apply':
                for role in (('expired', 'day-six') if midnight else ('expired',)):
                    roles[role].unlink()
                    roles[role].with_name(roles[role].name + '.sha256').unlink()
                copy('new', now, 'first' if fault == 'new-content' else 'second', 100)
                if fault == 'first-content':
                    shutil.copyfile(second, roles['first'])
                if fault == 'missing-first':
                    roles['first'].unlink()
                if fault == 'extra':
                    copy('unexpected', now - dt.timedelta(days=10), 'second', 101)
                return b'', b''
            report = {'mode': 'plan', 'verified': 5 if midnight else 6, 'ignored': 2, 'delete': [],
                      'protected': [roles['new'].name, roles['second'].name],
                      'budget_bytes': 5 * 1024 ** 3, 'floor_bytes': 512 * 1024 ** 2,
                      'estimate_bytes': 16 * 1024 ** 2, 'reserve_bytes': 32 * 1024 ** 2, 'free_bytes': 1024 ** 3}
            return json.dumps(report).encode(), b''

        acceptance.wrapper = wrapper
        return acceptance

    def test_prune_accepts_realistic_hour_rollover_and_checks_each_origin(self):
        with tempfile.TemporaryDirectory() as directory:
            acceptance = self.scenario(directory)
            details = acceptance.prune()
            self.assertEqual(details['verified_retained'], 6)
            self.assertEqual([item['origin'] for item in details['provenance']].count('first'), 1)

    def test_prune_applies_new_utc_date_to_history_created_before_midnight(self):
        with tempfile.TemporaryDirectory() as directory:
            details = self.scenario(directory, midnight=True).prune()
            self.assertEqual(details['verified_retained'], 5)
            self.assertEqual(details['obsolete_deleted'], 2)

    def test_prune_rejects_wrong_content_missing_or_extra_identity(self):
        for fault, reason in [('first-content', 'declared_snapshot_hash'),
                              ('new-content', 'snapshot_full_schema_rows'),
                              ('missing-first', 'retained_exact_identities'), ('extra', 'one_new_snapshot')]:
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                with self.assertRaisesRegex(AssertionError, '^' + reason + '$'):
                    self.scenario(directory, fault).prune()


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
