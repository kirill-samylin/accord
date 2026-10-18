"""Wrapper integration with a tiny synthetic SQLite copier, never real service."""
import contextlib
import hashlib
import io
import json
import os
import sqlite3
import subprocess
import sys
import unittest
from unittest import mock

from test_backup_retention import BackupFixture, backup


class WrapperTests(BackupFixture):
    def setUp(self):
        super().setUp()
        self.helper = self.root / 'synthetic-copy'
        self.write_helper("""
with contextlib.closing(sqlite3.connect(source)) as original, contextlib.closing(sqlite3.connect(output)) as destination:
    original.backup(destination)
os.chmod(output, 0o600)
""")
        patcher = mock.patch.object(backup, 'BINARY', str(self.helper))
        patcher.start()
        self.addCleanup(patcher.stop)
        self.free = mock.patch.object(backup, 'available', return_value=1024 ** 3)
        self.free.start()
        self.addCleanup(self.free.stop)

    def write_helper(self, body):
        self.helper.write_text('#!' + sys.executable + '\n' +
                               'import contextlib, os, sqlite3, sys\n' +
                               "source=sys.argv[sys.argv.index('--db')+1]\n" +
                               "output=sys.argv[sys.argv.index('--output')+1]\n" + body)
        self.helper.chmod(0o700)

    def create(self, name=None, limit=10 ** 8):
        name = name or ('board-20261018T120000Z-' + 'f' * 32 + '.db')
        return backup.create_snapshot(self.directory, name, limit)

    def test_real_synthetic_child_publishes_durable_checksum_pair(self):
        item = self.create()
        data = (self.copies / item.name).read_bytes()
        checksum = (self.copies / (item.name + '.sha256')).read_text()
        self.assertEqual(checksum, hashlib.sha256(data).hexdigest() + '  ' + item.name + '\n')
        self.assertEqual(backup.inventory(self.directory), ([item], 0))
        self.assertEqual((self.copies / item.name).stat().st_mode & 0o777, 0o600)

    def test_failed_child_removes_only_new_private_partial_output(self):
        existing = self.snapshot()
        self.write_helper("open(output, 'wb').write(b'partial')\nos.chmod(output, 0o600)\nsys.exit(9)\n")
        with self.assertRaises(subprocess.CalledProcessError):
            self.create()
        self.assertEqual(backup.inventory(self.directory), ([existing], 0))
        self.assertEqual(len(list(self.copies.iterdir())), 2)

    def test_corrupt_successful_child_and_over_reserve_outputs_cleaned(self):
        self.write_helper("open(output, 'wb').write(b'bad database')\nos.chmod(output, 0o600)\n")
        with self.assertRaises(sqlite3.DatabaseError):
            self.create()
        self.assertEqual(list(self.copies.iterdir()), [])
        self.write_helper("with contextlib.closing(sqlite3.connect(source)) as src, contextlib.closing(sqlite3.connect(output)) as dst: src.backup(dst)\nos.chmod(output, 0o600)\n")
        with self.assertRaisesRegex(backup.BackupError, 'copy_exceeded_reserve'):
            self.create(limit=1)
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_low_disk_after_copy_cleans_uncommitted_new_output(self):
        existing = self.sorted_snapshots(2)
        with mock.patch.object(backup, 'available', return_value=0):
            with self.assertRaisesRegex(backup.BackupError, 'copy_exceeded_reserve'):
                self.create()
        self.assertEqual(backup.inventory(self.directory), (existing, 0))

    def test_fsync_failure_at_each_publication_stage_cleans_pair(self):
        for failed in (1, 2, 3):
            with self.subTest(stage=failed):
                actual = backup.os.fsync
                calls = 0

                def sync(fd):
                    nonlocal calls
                    calls += 1
                    if calls == failed:
                        raise OSError('synthetic fsync failure')
                    return actual(fd)

                with mock.patch.object(backup.os, 'fsync', side_effect=sync):
                    with self.assertRaises(OSError):
                        self.create()
                self.assertEqual(list(self.copies.iterdir()), [])

    def test_preexisting_output_or_checksum_are_never_removed(self):
        name = 'board-20261018T120000Z-' + 'f' * 32 + '.db'
        for suffix in ('', '.sha256'):
            with self.subTest(suffix=suffix):
                path = self.copies / (name + suffix)
                path.write_bytes(b'operator file')
                path.chmod(0o600)
                with self.assertRaisesRegex(backup.BackupError, 'output_already_exists'):
                    self.create(name)
                self.assertEqual(path.read_bytes(), b'operator file')
                path.unlink()

    def test_cleanup_preserves_operator_replacement(self):
        name = 'board-20261018T120000Z-' + 'f' * 32 + '.db'
        path = self.copies / name

        def replace(_stream):
            path.unlink()
            path.write_bytes(b'operator replacement')
            path.chmod(0o600)
            raise backup.BackupError('synthetic_failure')

        with mock.patch.object(backup, 'check_sqlite', side_effect=replace):
            with self.assertRaises(backup.BackupError):
                self.create(name)
        self.assertEqual(path.read_bytes(), b'operator replacement')

    def test_plan_is_readonly_and_matches_the_actual_deletion_set(self):
        snapshots = self.sorted_snapshots(8)
        (self.copies / 'operator.txt').write_text('preserve')
        before = {path.name: path.read_bytes() for path in self.copies.iterdir()}
        lock = self.state / '.maintenance.lock'
        lock_before = lock.stat()
        output = io.StringIO()
        with mock.patch.object(backup, 'MARGIN', 4096), mock.patch.object(backup, 'BUDGET', 50000):
            # Explicit budget/floor parameters bind defaults at definition time;
            # wrap with the desired synthetic policy instead of changing globals.
            actual_plan = backup.plan

            def small_plan(items, now, estimate, free):
                return actual_plan(items, now, estimate, free, budget=50000, floor=0)

            with mock.patch.object(backup, 'plan', side_effect=small_plan):
                with contextlib.redirect_stdout(output):
                    backup.run(plan_only=True)
                report = json.loads(output.getvalue())
                self.assertEqual(before, {path.name: path.read_bytes() for path in self.copies.iterdir()})
                self.assertEqual(lock_before.st_mtime_ns, lock.stat().st_mtime_ns)
                self.assertEqual(report['protected'], [item.name for item in snapshots[:2]])
                self.assertGreater(len(report['delete']), 0)
                with contextlib.redirect_stdout(io.StringIO()):
                    backup.run()
        found, ignored = backup.inventory(self.directory)
        self.assertEqual(ignored, 0)
        self.assertTrue(all((self.copies / item.name).exists() for item in snapshots[:2]))
        self.assertTrue(all(not (self.copies / name).exists() for name in report['delete']))
        self.assertLessEqual(sum(item.size for item in found), 50000)
        self.assertEqual((self.copies / 'operator.txt').read_text(), 'preserve')

    def test_impossible_preflight_preserves_all_files_and_never_starts_child(self):
        self.sorted_snapshots(4)
        before = {path.name: path.read_bytes() for path in self.copies.iterdir()}
        with mock.patch.object(backup, 'available', return_value=0):
            with mock.patch.object(backup.subprocess, 'run') as child:
                with self.assertRaisesRegex(backup.BackupError, 'insufficient_space_preserving_two'):
                    backup.run()
                child.assert_not_called()
        self.assertEqual(before, {path.name: path.read_bytes() for path in self.copies.iterdir()})

    def test_readonly_plan_never_creates_missing_lock(self):
        lock = self.state / ".maintenance.lock"
        lock.unlink()
        with self.assertRaises(FileNotFoundError):
            backup.run(plan_only=True)
        self.assertFalse(lock.exists())
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_low_disk_bootstrap_never_starts_child(self):
        with mock.patch.object(backup, "available", return_value=0):
            with mock.patch.object(backup.subprocess, "run") as child:
                with self.assertRaises(backup.BackupError):
                    backup.run()
                child.assert_not_called()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_bootstrap_first_pair_succeeds_and_reports_redundancy(self):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            backup.run()
        self.assertIn("redundancy: 1/2", output.getvalue())
        self.assertEqual(len(backup.inventory(self.directory)[0]), 1)
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            backup.run()
        self.assertIn("redundancy: 2/2", output.getvalue())
        self.assertEqual(len(backup.inventory(self.directory)[0]), 2)

    def test_source_growth_after_rotation_fails_before_child(self):
        self.sorted_snapshots(2)
        with mock.patch.object(backup, 'source_estimate', side_effect=[100, 10 ** 12]):
            with mock.patch.object(backup.subprocess, 'run') as child:
                with self.assertRaisesRegex(backup.BackupError, 'space_changed_before_copy'):
                    backup.run()
                child.assert_not_called()
        self.assertEqual(len(backup.inventory(self.directory)[0]), 2)


if __name__ == '__main__':
    unittest.main()
