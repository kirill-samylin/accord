from contextlib import closing
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("web_backup", ROOT / "infra/accord/web/backup.py")
backup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(backup)


class WebBackupTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.state = self.root / "state"
        self.copies = self.root / "copies"
        for directory in [self.state, self.copies, self.state / "web", self.state / "web/blobs", self.state / "web/staging"]:
            directory.mkdir(mode=0o700)
        for file, version in [(self.state / "board.db", 2), (self.state / "web/web.db", 1)]:
            with closing(sqlite3.connect(file)) as db, db:
                db.execute("PRAGMA user_version=" + str(version))
                db.execute("CREATE TABLE data(value)")
                db.execute("INSERT INTO data VALUES('synthetic')")
            file.chmod(0o600)
        with closing(sqlite3.connect(self.state / "web/web.db")) as db, db:
            db.executescript("CREATE TABLE design_assets(blob,size,sha256,preview,preview_size,status); CREATE TABLE browser_sessions(value); CREATE TABLE preauth(value); CREATE TABLE web_actions(value);")
            for table in ["browser_sessions", "preauth", "web_actions"]:
                db.execute("INSERT INTO " + table + " VALUES('synthetic')")

    def test_snapshot_is_self_contained_and_resets_restored_web_auth(self):
        result = backup.snapshot(self.state, self.copies)
        manifest = backup.validate_snapshot(result)
        self.assertEqual(set(manifest["files"]), {"board.db", "web/web.db"})
        backup.prepare_restored_auth(result)
        with closing(sqlite3.connect(result / "web/web.db")) as db, db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM browser_sessions").fetchone()[0], 0)
        with closing(sqlite3.connect(self.state / "web/web.db")) as db, db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM browser_sessions").fetchone()[0], 1)

    def test_ready_blob_checksum_and_snapshot_tampering(self):
        file = self.state / "web/blobs" / ("a" * 48)
        file.write_bytes(b"synthetic-design")
        file.chmod(0o600)
        with closing(sqlite3.connect(self.state / "web/web.db")) as db, db:
            db.execute("INSERT INTO design_assets VALUES(?,?,?,?,?,?)", (file.name, file.stat().st_size, backup.sha(file), "", 0, "ready"))
        result = backup.snapshot(self.state, self.copies)
        backup.validate_snapshot(result)
        (result / "web/blobs" / file.name).write_bytes(b"changed")
        with self.assertRaises(RuntimeError):
            backup.validate_snapshot(result)

    def test_failed_new_copy_preserves_last_good(self):
        good = backup.snapshot(self.state, self.copies)
        with patch.object(backup.shutil, "disk_usage", return_value=type("Disk", (), {"free": 0})()):
            with self.assertRaises(RuntimeError):
                backup.snapshot(self.state, self.copies)
        backup.validate_snapshot(good)
        self.assertEqual(len(list(self.copies.iterdir())), 1)

    def test_no_symlink_inputs(self):
        bad = self.state / "web/blobs" / ("b" * 48)
        bad.symlink_to(self.state / "board.db")
        with self.assertRaises(RuntimeError):
            backup.snapshot(self.state, self.copies)

    def test_retention_only_after_new_verified_copy(self):
        with patch.object(backup, "RETAIN", 2):
            for _ in range(3):
                backup.snapshot(self.state, self.copies)
        self.assertEqual(len(list(self.copies.iterdir())), 2)
        for item in self.copies.iterdir():
            backup.validate_snapshot(item)
