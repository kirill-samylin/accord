#!/usr/bin/python3
"""Bounded local snapshots. --plan verifies and prints a plan without writes."""
import argparse
import contextlib
import datetime as dt
import fcntl
import hashlib
import json
import os
import pathlib
import re
import sqlite3
import stat
import subprocess
import sys
import uuid
from dataclasses import dataclass

STATE = pathlib.Path('/var/lib/accord')
BACKUPS = pathlib.Path('/var/backups/accord')
BINARY = '/opt/accord/current/agent-board'
NAME = re.compile(r'^board-([0-9]{8}T[0-9]{6}Z)-[0-9a-f]{32}\.db$')
BUDGET = 5 * 1024 ** 3
FLOOR = 512 * 1024 ** 2
MARGIN = 16 * 1024 ** 2
UTC = dt.timezone.utc


class BackupError(RuntimeError):
    """Only fixed, non-sensitive reason codes are logged by the entry point."""


def regular(info):
    return (stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid()
            and stat.S_IMODE(info.st_mode) == 0o600 and info.st_nlink == 1)


def identity(info):
    return info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns


def open_regular(directory, name, flags=os.O_RDONLY):
    fd = os.open(name, flags | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600, dir_fd=directory)
    if not regular(os.fstat(fd)):
        os.close(fd)
        raise BackupError('unsafe_regular_file')
    return os.fdopen(fd, 'rb' if flags == os.O_RDONLY else 'r+b')


def open_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    info = os.fstat(fd)
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        os.close(fd)
        raise BackupError('unsafe_directory')
    return fd


def digest(stream):
    value = hashlib.sha256()
    for block in iter(lambda: stream.read(1024 * 1024), b''):
        value.update(block)
    return value.hexdigest()


def check_sqlite(stream):
    # This OS descriptor alias is intentional; user-controlled paths were opened
    # O_NOFOLLOW first. Immutable snapshots must have no live SQLite sidecars.
    uri = 'file:/dev/fd/' + str(stream.fileno()) + '?mode=ro&immutable=1'
    with contextlib.closing(sqlite3.connect(uri, uri=True)) as database:
        rows = database.execute('PRAGMA integrity_check')
        if rows.fetchone() != ('ok',) or rows.fetchone() is not None:
            raise BackupError('snapshot_integrity_failed')


@dataclass(frozen=True)
class Snapshot:
    name: str
    created: dt.datetime
    db: os.stat_result
    checksum: os.stat_result

    @property
    def size(self):
        return self.db.st_size + self.checksum.st_size

    @property
    def reclaimable(self):
        return sum(min(info.st_size, info.st_blocks * 512) for info in (self.db, self.checksum))


def verify(directory, name):
    match = NAME.fullmatch(name)
    if match is None:
        raise BackupError('invalid_snapshot_name')
    created = dt.datetime.strptime(match[1], '%Y%m%dT%H%M%SZ').replace(tzinfo=UTC)
    for suffix in ('-wal', '-shm', '-journal'):
        try:
            os.stat(name + suffix, dir_fd=directory, follow_symlinks=False)
        except FileNotFoundError:
            continue
        raise BackupError('snapshot_sidecar_present')
    with open_regular(directory, name) as data, open_regular(directory, name + '.sha256') as checksum:
        before, metadata = os.fstat(data.fileno()), os.fstat(checksum.fileno())
        expected = (digest(data) + '  ' + name + '\n').encode('ascii')
        if checksum.read(len(expected) + 1) != expected:
            raise BackupError('snapshot_checksum_failed')
        check_sqlite(data)
        if identity(before) != identity(os.fstat(data.fileno())):
            raise BackupError('snapshot_changed')
        if identity(metadata) != identity(os.fstat(checksum.fileno())):
            raise BackupError('checksum_changed')
        return Snapshot(name, created, before, metadata)


def inventory(directory):
    snapshots, ignored = [], 0
    for name in os.listdir(directory):
        if NAME.fullmatch(name):
            try:
                snapshots.append(verify(directory, name))
            except (OSError, ValueError, sqlite3.Error, BackupError):
                ignored += 1
    return sorted(snapshots, key=lambda item: (item.created, item.db.st_mtime_ns, item.name), reverse=True), ignored


def preferred(snapshots, now):
    """Newest in each of 24 UTC hour buckets and 7 UTC calendar dates."""
    keep = {item.name for item in snapshots[:2]}
    hours, days = set(), set()
    hour_start = now.replace(minute=0, second=0, microsecond=0) - dt.timedelta(hours=23)
    day_start = now.date() - dt.timedelta(days=6)
    for item in snapshots:
        if item.created > now:
            continue
        hour = item.created.replace(minute=0, second=0, microsecond=0)
        if hour >= hour_start and hour not in hours:
            hours.add(hour)
            keep.add(item.name)
        day = item.created.date()
        if day >= day_start and day not in days:
            days.add(day)
            keep.add(item.name)
    return keep


def plan(snapshots, now, estimate, free, budget=BUDGET, floor=FLOOR):
    """Decide all deletions before touching anything; two newest are inviolable."""
    protected = {item.name for item in snapshots[:2]}
    keep = preferred(snapshots, now)
    removed = [item for item in reversed(snapshots) if item.name not in keep]
    total = sum(item.size for item in snapshots)
    reclaimed = sum(item.size for item in removed)
    reclaimed_space = sum(item.reclaimable for item in removed)
    reserve = 2 * estimate  # one full output plus SQLite working headroom
    remaining = [item for item in reversed(snapshots) if item.name in keep and item.name not in protected]
    while total - reclaimed + estimate > budget or free + reclaimed_space < reserve + floor:
        if not remaining:
            raise BackupError('insufficient_space_preserving_two')
        item = remaining.pop(0)
        removed.append(item)
        reclaimed += item.size
        reclaimed_space += item.reclaimable
    return removed


def source_estimate(state):
    size = 0
    for name in ('board.db', 'board.db-wal'):
        try:
            with open_regular(state, name) as stream:
                size += os.fstat(stream.fileno()).st_size
        except FileNotFoundError:
            if name == 'board.db':
                raise BackupError('source_missing')
    return size + MARGIN


def available(directory):
    info = os.fstatvfs(directory)
    return info.f_bavail * info.f_frsize


def remove_pair(directory, item):
    # Reverify immediately before deleting, then compare both saved identities.
    current = verify(directory, item.name)
    if identity(current.db) != identity(item.db) or identity(current.checksum) != identity(item.checksum):
        raise BackupError('snapshot_replaced')
    # Data first: a crash can leave an orphan checksum, never a trusted partial DB.
    os.unlink(item.name, dir_fd=directory)
    os.unlink(item.name + '.sha256', dir_fd=directory)
    os.fsync(directory)


def create_snapshot(directory, name, limit):
    output = BACKUPS / name
    created = {}
    complete = False
    try:
        for suffix in ('', '.sha256', '-wal', '-shm', '-journal'):
            try:
                os.stat(name + suffix, dir_fd=directory, follow_symlinks=False)
            except FileNotFoundError:
                continue
            raise BackupError('output_already_exists')
        try:
            subprocess.run([BINARY, 'backup', '--db', str(STATE / 'board.db'), '--output', str(output)],
                           check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        finally:
            try:
                with open_regular(directory, name) as data:
                    created[name] = os.fstat(data.fileno())
            except FileNotFoundError:
                pass
        with open_regular(directory, name) as data:
            created[name] = os.fstat(data.fileno())
            check_sqlite(data)
            value = digest(data)
            os.fsync(data.fileno())
        checksum = name + '.sha256'
        fd = os.open(checksum, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=directory)
        with os.fdopen(fd, 'wb') as stream:
            created[checksum] = os.fstat(stream.fileno())
            stream.write((value + '  ' + name + '\n').encode('ascii'))
            stream.flush()
            os.fsync(stream.fileno())
            created[checksum] = os.fstat(stream.fileno())
        if (sum(info.st_size for info in created.values()) > limit
                or available(directory) < FLOOR):
            raise BackupError('copy_exceeded_reserve')
        os.fsync(directory)
        result = verify(directory, name)
        complete = True
        return result
    finally:
        if not complete:
            # CLI cleans failed staging. Only outputs whose inode we observed
            # are ours; never remove replacements or pre-existing artifacts.
            for entry, original in created.items():
                try:
                    current = os.stat(entry, dir_fd=directory, follow_symlinks=False)
                    if (current.st_dev, current.st_ino) == (original.st_dev, original.st_ino):
                        os.unlink(entry, dir_fd=directory)
                except FileNotFoundError:
                    pass
            os.fsync(directory)


def run(plan_only=False):
    os.umask(0o077)
    state = open_directory(STATE)
    directory = None
    try:
        directory = open_directory(BACKUPS)
        flags = os.O_RDONLY if plan_only else os.O_RDWR | os.O_CREAT
        with open_regular(state, '.maintenance.lock', flags) as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            now = dt.datetime.now(UTC)
            snapshots, ignored = inventory(directory)
            estimate = source_estimate(state)
            free = available(directory)
            removed = plan(snapshots, now, estimate, free)
            report = {'mode': 'plan' if plan_only else 'apply', 'budget_bytes': BUDGET,
                      'estimate_bytes': estimate, 'free_bytes': free, 'floor_bytes': FLOOR,
                      'reserve_bytes': 2 * estimate, 'verified': len(snapshots), 'ignored': ignored,
                      'protected': [item.name for item in snapshots[:2]],
                      'delete': [item.name for item in removed]}
            if plan_only:
                print(json.dumps(report, sort_keys=True))
                return
            for item in removed:
                remove_pair(directory, item)
            # Live writers may grow DB/WAL during verification and rotation.
            estimate = source_estimate(state)
            retained = [item for item in snapshots if item not in removed]
            if (sum(item.size for item in retained) + estimate > BUDGET
                    or available(directory) < 2 * estimate + FLOOR):
                raise BackupError('space_changed_before_copy')
            name = 'board-' + now.strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex + '.db'
            result = create_snapshot(directory, name, estimate)
            completed = sorted([result] + retained, key=lambda item: (item.created, item.db.st_mtime_ns, item.name), reverse=True)
            expired = plan(completed, now, 0, available(directory))
            for item in expired:
                remove_pair(directory, item)
            print('Accord backup completed; retained snapshots:', len(completed) - len(expired),
                  '; redundancy:', str(min(2, len(completed))) + '/2',
                  '; deleted:', len(removed) + len(expired), '; ignored:', ignored)
    finally:
        if directory is not None:
            os.close(directory)
        os.close(state)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan', action='store_true')
    arguments = parser.parse_args()
    try:
        run(arguments.plan)
    except (OSError, RuntimeError, sqlite3.Error, subprocess.SubprocessError) as error:
        reason = str(error) if isinstance(error, BackupError) else type(error).__name__
        print('Accord backup failed:', reason, file=sys.stderr)
        sys.exit(1)
