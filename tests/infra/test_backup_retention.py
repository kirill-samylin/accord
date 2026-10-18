"""Synthetic unit coverage for local backup retention; no VPS or Go process."""
import contextlib
import datetime as dt
import hashlib
import importlib.util
import os
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from types import SimpleNamespace
from unittest import mock

MODULE = Path(__file__).resolve().parents[2] / 'infra/accord/backup.py'
spec = importlib.util.spec_from_file_location('accord_backup_budget', MODULE)
backup = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = backup
spec.loader.exec_module(backup)
NOW = dt.datetime(2026, 10, 18, 12, 30, tzinfo=backup.UTC)


class BackupFixture(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.state = self.root / 'state'
        self.copies = self.root / 'copies'
        self.state.mkdir(mode=0o700)
        self.copies.mkdir(mode=0o700)
        self.database(self.state / 'board.db')
        (self.state / '.maintenance.lock').touch(mode=0o600)
        self.directory = backup.open_directory(self.copies)
        self.addCleanup(os.close, self.directory)
        self.paths = mock.patch.multiple(backup, STATE=self.state, BACKUPS=self.copies)
        self.paths.start()
        self.addCleanup(self.paths.stop)

    def database(self, path):
        with contextlib.closing(sqlite3.connect(path)) as connection, connection:
            connection.execute('CREATE TABLE sample(value TEXT)')
            connection.execute("INSERT INTO sample VALUES ('synthetic')")
        path.chmod(0o600)

    def snapshot(self, when=NOW, number=1):
        name = 'board-' + when.strftime('%Y%m%dT%H%M%SZ') + '-' + f'{number:032x}' + '.db'
        path = self.copies / name
        self.database(path)
        self.checksum(path)
        return backup.verify(self.directory, name)

    def checksum(self, path):
        checksum = path.with_name(path.name + '.sha256')
        checksum.write_text(hashlib.sha256(path.read_bytes()).hexdigest() + '  ' + path.name + '\n')
        checksum.chmod(0o600)

    def sorted_snapshots(self, count):
        return [self.snapshot(NOW - dt.timedelta(hours=index), index + 1) for index in range(count)]


class PolicyTests(unittest.TestCase):
    def item(self, created, number, size=100):
        return SimpleNamespace(name=str(number), created=created, size=size, reclaimable=size)

    def test_hour_and_utc_calendar_boundaries(self):
        dates = [NOW, NOW.replace(minute=0), NOW.replace(minute=0) - dt.timedelta(seconds=1),
                 NOW.replace(hour=13, minute=0) - dt.timedelta(days=1),
                 NOW.replace(hour=12, minute=59) - dt.timedelta(days=1),
                 NOW.replace(hour=0, minute=0) - dt.timedelta(days=6),
                 NOW.replace(hour=23, minute=59) - dt.timedelta(days=7)]
        items = [self.item(when, number) for number, when in enumerate(dates)]
        # item1 is the protected duplicate hour; item4 is superseded daily and
        # outside the current24 UTC hour buckets; item6 is outside7UTC dates.
        self.assertEqual(backup.preferred(items, NOW), {'0', '1', '2', '3', '5'})

    def test_future_and_manual_duplicate_snapshots(self):
        items = [self.item(NOW + dt.timedelta(hours=1), 0), self.item(NOW, 1),
                 self.item(NOW - dt.timedelta(seconds=1), 2)]
        self.assertEqual(backup.preferred(items, NOW), {'0', '1'})

    def test_oldest_pruned_until_total_budget_fits(self):
        items = [self.item(NOW - dt.timedelta(hours=i), i) for i in range(6)]
        removed = backup.plan(items, NOW, estimate=100, free=10000, budget=450, floor=0)
        self.assertEqual([item.name for item in removed], ['5', '4', '3'])
        self.assertEqual(sum(item.size for item in items if item not in removed) + 100, 400)

    def test_low_disk_can_prune_recent_hourly_but_keeps_two(self):
        items = [self.item(NOW - dt.timedelta(hours=i), i) for i in range(4)]
        removed = backup.plan(items, NOW, estimate=100, free=50, budget=1000, floor=50)
        self.assertEqual([item.name for item in removed], ['3', '2'])

    def test_impossible_budget_or_disk_fails_without_plan(self):
        items = [self.item(NOW - dt.timedelta(hours=i), i) for i in range(5)]
        for budget, free in ((299, 10000), (1000, 0)):
            with self.subTest(budget=budget, free=free), self.assertRaises(backup.BackupError):
                backup.plan(items, NOW, estimate=100, free=free, budget=budget, floor=500)

    def test_sparse_files_do_not_promise_unallocated_disk_space(self):
        items = [self.item(NOW - dt.timedelta(hours=i), i) for i in range(4)]
        for item in items:
            item.reclaimable = 0
        with self.assertRaises(backup.BackupError):
            backup.plan(items, NOW, 100, 0, budget=1000, floor=0)

    def test_one_or_zero_existing_snapshots_never_deleted(self):
        item = self.item(NOW - dt.timedelta(days=10), 0)
        self.assertEqual(backup.plan([item], NOW, 100, 1000, budget=200, floor=0), [])
        with self.assertRaises(backup.BackupError):
            backup.plan([item], NOW, 100, 0, budget=200, floor=0)
        self.assertEqual(backup.plan([], NOW, 100, 1000, budget=100, floor=0), [])


class FileSafetyTests(BackupFixture):
    def test_verified_checksum_and_sqlite(self):
        item = self.snapshot()
        found, ignored = backup.inventory(self.directory)
        self.assertEqual(found, [item])
        self.assertEqual(ignored, 0)

    def test_same_second_order_prefers_newest_file_over_random_uuid(self):
        old = self.snapshot(number=3)
        middle = self.snapshot(number=2)
        latest = self.snapshot(number=1)
        found, ignored = backup.inventory(self.directory)
        self.assertEqual(ignored, 0)
        self.assertEqual(found, [latest, middle, old])

    def test_corrupt_data_and_matching_digest_not_restorable(self):
        item = self.snapshot()
        path = self.copies / item.name
        path.write_bytes(b'corrupt SQLite payload')
        self.checksum(path)
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        self.assertTrue(path.exists())

    def test_missing_wrong_or_oversized_checksum_untouched(self):
        item = self.snapshot()
        checksum = self.copies / (item.name + '.sha256')
        for value in (b'', b'0' * 65, b'x' * 100000):
            checksum.write_bytes(value)
            self.assertEqual(backup.inventory(self.directory), ([], 1))
            self.assertEqual(checksum.read_bytes(), value)
        checksum.unlink()
        self.assertEqual(backup.inventory(self.directory), ([], 1))

    def test_unsafe_mode_owner_hardlink_sidecar_and_symlink_untouched(self):
        item = self.snapshot()
        path = self.copies / item.name
        path.chmod(0o644)
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        path.chmod(0o600)
        real_uid = os.getuid()
        with mock.patch.object(backup.os, 'getuid', return_value=real_uid + 1):
            self.assertEqual(backup.inventory(self.directory), ([], 1))
        alias = self.copies / 'foreign-hardlink'
        os.link(path, alias)
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        alias.unlink()
        sidecar = self.copies / (item.name + '-wal')
        sidecar.symlink_to(self.state / 'board.db')
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        sidecar.unlink()
        content = path.read_bytes()
        path.unlink()
        path.symlink_to(self.state / 'board.db')
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        self.assertTrue(path.is_symlink())
        self.assertEqual((self.state / 'board.db').read_bytes(), content)

    def test_checksum_symlink_and_foreign_files_preserved(self):
        item = self.snapshot()
        checksum = self.copies / (item.name + '.sha256')
        checksum.unlink()
        checksum.symlink_to(self.state / 'board.db')
        foreign = self.copies / 'operator-copy.db'
        foreign.write_text('preserve')
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        self.assertTrue(checksum.is_symlink())
        self.assertEqual(foreign.read_text(), 'preserve')

    def test_replacement_or_corruption_before_delete_fails_closed(self):
        item = self.snapshot()
        path = self.copies / item.name
        path.write_bytes(b'corrupt')
        with self.assertRaises(backup.BackupError):
            backup.remove_pair(self.directory, item)
        self.assertEqual(path.read_bytes(), b'corrupt')

    def test_replacement_with_valid_bytes_is_not_deleted(self):
        item = self.snapshot()
        path = self.copies / item.name
        replacement = self.root / "replacement.db"
        replacement.write_bytes(path.read_bytes())
        replacement.chmod(0o600)
        os.replace(replacement, path)
        with self.assertRaisesRegex(backup.BackupError, "snapshot_replaced"):
            backup.remove_pair(self.directory, item)
        self.assertTrue(path.exists())

    def test_fifo_is_ignored_without_blocking(self):
        name = "board-20261018T120000Z-" + "a" * 32 + ".db"
        os.mkfifo(self.copies / name, mode=0o600)
        self.assertEqual(backup.inventory(self.directory), ([], 1))

    def test_invalid_timestamp_and_directory_entry_are_ignored(self):
        name = 'board-20269999T999999Z-' + '0' * 32 + '.db'
        (self.copies / name).mkdir()
        self.assertEqual(backup.inventory(self.directory), ([], 1))
        self.assertTrue((self.copies / name).is_dir())

    def test_source_estimate_includes_wal_and_rejects_symlink(self):
        source = backup.open_directory(self.state)
        self.addCleanup(os.close, source)
        wal = self.state / 'board.db-wal'
        wal.write_bytes(b'x' * 99)
        wal.chmod(0o600)
        self.assertEqual(backup.source_estimate(source), (self.state / 'board.db').stat().st_size + 99 + backup.MARGIN)
        wal.unlink()
        wal.symlink_to(self.state / 'board.db')
        with self.assertRaises(OSError):
            backup.source_estimate(source)

    def test_private_directory_and_lock_guards(self):
        alias = self.root / 'link'
        alias.symlink_to(self.copies)
        with self.assertRaises(OSError):
            backup.open_directory(alias)
        self.copies.chmod(0o755)
        with self.assertRaises(backup.BackupError):
            backup.open_directory(self.copies)
        self.copies.chmod(0o700)
        lock = self.state / '.maintenance.lock'
        lock.unlink()
        lock.symlink_to(self.state / 'board.db')
        with self.assertRaises(OSError):
            backup.run()


if __name__ == '__main__':
    unittest.main()
