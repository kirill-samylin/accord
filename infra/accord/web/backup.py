#!/usr/bin/python3
"""Offline, self-contained board + web snapshots. Never prints stored content.

The systemd wrapper runs as root because stopping the single writer is required.
All other scheduled administration shares Accord's existing maintenance lock.
"""
from contextlib import closing
import datetime
import fcntl
import hashlib
import json
import os
import pathlib
import re
import shutil
import signal
import sqlite3
import stat
import subprocess
import sys
import uuid

FLOOR = 512 << 20
RETAIN = 7
NAME = re.compile(r"^web-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}$")
BLOB = re.compile(r"^[0-9a-f]{48}(?:\.preview)?$")


def private(path, directory=False):
    info = path.lstat()
    kind = stat.S_ISDIR if directory else stat.S_ISREG
    if not kind(info.st_mode) or stat.S_IMODE(info.st_mode) != (0o700 if directory else 0o600):
        raise RuntimeError("unsafe snapshot input")
    return info.st_size


def sync_dir(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def sha(path):
    value = hashlib.sha256()
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as stream:
        for block in iter(lambda: stream.read(1 << 20), b""):
            value.update(block)
    return value.hexdigest()


def db_copy(source, target, version):
    private(source)
    fd = os.open(target, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    os.close(fd)
    # The sole writer has stopped and closed SQLite. Refuse an uncheckpointed
    # WAL rather than ignoring committed frames. Immutable reads avoid creating
    # root-owned sidecars in the service account's source directory.
    for suffix in ("-wal", "-journal"):
        journal = source.with_name(source.name + suffix)
        if (journal.exists() or journal.is_symlink()) and private(journal) != 0:
            raise RuntimeError("offline source still has an unresolved transaction journal")
    with closing(sqlite3.connect(source.as_uri() + "?mode=ro&immutable=1", uri=True)) as src, src:
        with closing(sqlite3.connect(target)) as dst, dst:
            src.backup(dst)
            # Backup copies the WAL header too. Normalize the offline snapshot
            # so every supported SQLite version can inspect it without sidecars.
            if dst.execute("PRAGMA journal_mode=DELETE").fetchone()[0] != "delete":
                raise RuntimeError("snapshot journal normalization failed")
            if dst.execute("PRAGMA integrity_check").fetchone()[0] != "ok":
                raise RuntimeError("snapshot integrity failed")
            if dst.execute("PRAGMA user_version").fetchone()[0] != version:
                raise RuntimeError("unexpected snapshot schema")
    with target.open("rb") as stream:
        header = stream.read(20)
        if header[:16] != b"SQLite format 3\x00" or header[18:20] != b"\x01\x01":
            raise RuntimeError("snapshot is not a self-contained rollback-journal database")
        os.fsync(stream.fileno())
    # SQLite 3.54 may retain a stale SHM after leaving WAL mode. Only the newly
    # created, closed target is cleaned; the live source is never modified.
    for suffix in ["-wal", "-shm"]:
        sidecar = target.with_name(target.name + suffix)
        try:
            size = private(sidecar)
        except FileNotFoundError:
            continue
        if sidecar.lstat().st_uid != os.getuid() or (suffix == "-wal" and size != 0):
            raise RuntimeError("unsafe or nonempty snapshot sidecar")
        sidecar.unlink()
    sync_dir(target.parent)


def validate_snapshot(path):
    """Check the complete published manifest and all ready design references."""
    private(path, True)
    private(path / "manifest.json")
    for entry in path.rglob("*"):
        if entry.is_symlink():
            raise RuntimeError("symlink in snapshot")
        relative = str(entry.relative_to(path))
        if entry.is_dir():
            private(entry, True)
            if relative not in {"web", "web/blobs", "web/staging"}:
                raise RuntimeError("unexpected snapshot directory")
        else:
            private(entry)
            if relative not in {"manifest.json", "board.db", "web/web.db"} and not (
                    len(entry.relative_to(path).parts) == 3
                    and entry.relative_to(path).parts[:2] in {("web", "blobs"), ("web", "staging")}
                    and BLOB.fullmatch(entry.name)):
                raise RuntimeError("unexpected snapshot file")
    manifest = json.loads((path / "manifest.json").read_text())
    actual = {str(p.relative_to(path)) for p in path.rglob("*") if p.is_file() and p.name != "manifest.json"}
    if set(manifest["files"]) != actual:
        raise RuntimeError("snapshot file set mismatch")
    for name, digest in manifest["files"].items():
        rel = pathlib.PurePosixPath(name)
        if rel.is_absolute() or ".." in rel.parts:
            raise RuntimeError("unsafe manifest path")
        file = path / name
        private(file)
        if sha(file) != digest:
            raise RuntimeError("snapshot checksum mismatch")
    for name, version in [("board.db", 2), ("web/web.db", 1)]:
        with closing(sqlite3.connect((path / name).as_uri() + "?mode=ro&immutable=1", uri=True)) as db, db:
            if db.execute("PRAGMA integrity_check").fetchone()[0] != "ok" or db.execute("PRAGMA user_version").fetchone()[0] != version:
                raise RuntimeError("invalid snapshot database")
    with closing(sqlite3.connect((path / "web/web.db").as_uri() + "?mode=ro&immutable=1", uri=True)) as db, db:
        for blob, size, digest, preview, preview_size in db.execute("SELECT blob,size,sha256,preview,preview_size FROM design_assets WHERE status='ready'"):
            if not BLOB.fullmatch(blob) or (preview and not BLOB.fullmatch(preview)):
                raise RuntimeError("invalid design reference")
            file = path / "web/blobs" / blob
            if private(file) != size or sha(file) != digest:
                raise RuntimeError("design integrity failed")
            if preview and private(path / "web/blobs" / preview) != preview_size:
                raise RuntimeError("preview integrity failed")
    return manifest


def snapshot(state, backups):
    """Caller holds maintenance lock and has stopped the only writer."""
    projects = state / "projects-data"
    if any((projects / name).exists() or (projects / name).is_symlink()
           for name in ("registry.sqlite", "MULTIPROJECT_REQUIRES_CAPABILITY")):
        raise RuntimeError("registry requires the all-project snapshot operator")
    private(state, True)
    private(state / "web", True)
    private(backups, True)
    inputs = [state / "board.db", state / "web/web.db"]
    for subdir in ["blobs", "staging"]:
        folder = state / "web" / subdir
        private(folder, True)
        for entry in folder.iterdir():
            if not BLOB.fullmatch(entry.name):
                raise RuntimeError("unexpected blob entry")
            private(entry)
            inputs.append(entry)
    required = sum(private(p) for p in inputs) + (16 << 20)
    if shutil.disk_usage(backups).free < required + FLOOR:
        raise RuntimeError("insufficient snapshot disk reserve")
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    name = "web-" + stamp + "-" + uuid.uuid4().hex
    pending = backups / (".pending-" + name)
    published = backups / name
    pending.mkdir(mode=0o700)
    try:
        (pending / "web").mkdir(mode=0o700)
        db_copy(state / "board.db", pending / "board.db", 2)
        db_copy(state / "web/web.db", pending / "web/web.db", 1)
        for subdir in ["blobs", "staging"]:
            folder = pending / "web" / subdir
            folder.mkdir(mode=0o700)
            for source in (state / "web" / subdir).iterdir():
                # Source was validated above while the sole writer is stopped.
                fd = os.open(folder / source.name, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
                srcfd = os.open(source, os.O_RDONLY | os.O_NOFOLLOW)
                with os.fdopen(fd, "wb") as dst, os.fdopen(srcfd, "rb") as src:
                    shutil.copyfileobj(src, dst, 1 << 20)
                    dst.flush()
                    os.fsync(dst.fileno())
            sync_dir(folder)
        files = {str(f.relative_to(pending)): sha(f) for f in pending.rglob("*") if f.is_file()}
        manifest = {"format": 1, "files": files, "restore_requires_auth_reset": True}
        manifest_fd = os.open(pending / "manifest.json", os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
        with os.fdopen(manifest_fd, "w") as stream:
            json.dump(manifest, stream, sort_keys=True)
            stream.flush()
            os.fsync(stream.fileno())
        validate_snapshot(pending)
        sync_dir(pending / "web")
        sync_dir(pending)
        pending.rename(published)
        sync_dir(backups)
    except BaseException:
        shutil.rmtree(pending)
        raise
    complete = sorted([p for p in backups.iterdir() if NAME.fullmatch(p.name)], reverse=True)
    # Never discard a last good copy until the new one has passed validation.
    for old in complete[RETAIN:]:
        validate_snapshot(old)
        shutil.rmtree(old)
    sync_dir(backups)
    return published


def prepare_restored_auth(path):
    """Use only on an offline restored fixture before exposing any listener.

    Existing native recovery revokes board credentials and leases separately;
    old browser/preauth/action state must also never resume after restore.
    """
    with closing(sqlite3.connect(path / "web/web.db")) as db, db:
        for table in ["web_actions", "browser_sessions", "preauth"]:
            db.execute("DELETE FROM " + table)
        db.commit()


def interrupted(_signum, _frame):
    # systemd sends TERM on startup timeout. Turn it into an exception so the
    # sole-writer stop is always paired with the finally restart below.
    raise RuntimeError("snapshot interrupted")


def run():
    os.umask(0o077)
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    state = pathlib.Path("/var/lib/accord")
    backups = pathlib.Path("/var/backups/accord-web")
    private(state, True)
    private(backups, True)
    fd = os.open(state / ".maintenance.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        was_active = subprocess.run(["systemctl", "is-active", "--quiet", "accord.service"]).returncode == 0
        if not was_active:
            raise RuntimeError("snapshot requires a healthy running service")
        try:
            subprocess.run(["systemctl", "stop", "accord.service"], check=True)
            snapshot(state, backups)
        finally:
            subprocess.run(["systemctl", "start", "accord.service"], check=True)
    print("Accord board and web snapshot verified; retained at most seven daily copies")


if __name__ == "__main__":
    try:
        run()
    except (OSError, RuntimeError, sqlite3.Error, subprocess.SubprocessError):
        print("Accord web snapshot failed; inspect private state", file=sys.stderr)
        sys.exit(1)
