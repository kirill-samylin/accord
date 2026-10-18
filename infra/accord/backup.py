#!/usr/bin/python3
"""Run the application's online snapshot; publish only validated complete copies."""
import datetime
import fcntl
import hashlib
import os
import pathlib
import re
import stat
import subprocess
import sys
import uuid

STATE = pathlib.Path('/var/lib/accord')
BACKUPS = pathlib.Path('/var/backups/accord')
BINARY = '/opt/accord/current/agent-board'
NAME = re.compile(r'^board-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}\.db$')
RETAIN = 168


def private_directory(path):
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid():
        raise RuntimeError('expected an owned private directory')
    if stat.S_IMODE(info.st_mode) != 0o700:
        raise RuntimeError('directory permissions must be 0700')


def run():
    os.umask(0o077)
    private_directory(STATE)
    private_directory(BACKUPS)
    # All offline administration and scheduled snapshots share this lock.
    fd = os.open(STATE / '.maintenance.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as lock:
        if stat.S_IMODE(os.fstat(lock.fileno()).st_mode) != 0o600:
            raise RuntimeError('maintenance lock must be private')
        fcntl.flock(lock, fcntl.LOCK_EX)
        timestamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
        name = 'board-' + timestamp + '-' + uuid.uuid4().hex + '.db'
        output = BACKUPS / name
        # CLI owns consistent SQLite snapshot, integrity verification and fsync.
        subprocess.run([BINARY, 'backup', '--db', str(STATE / 'board.db'),
                        '--output', str(output)], check=True, stdout=subprocess.DEVNULL)
        info = output.lstat()
        if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600:
            raise RuntimeError('backup output is not a private regular file')
        value = hashlib.sha256()
        with output.open('rb') as stream:
            for block in iter(lambda: stream.read(1024 * 1024), b''):
                value.update(block)
        digest = value.hexdigest()
        checksum = output.with_suffix('.db.sha256')
        with checksum.open('x', encoding='ascii') as stream:
            stream.write(digest + '  ' + name + '\n')
            stream.flush()
            os.fsync(stream.fileno())
        # Directory fsync persists both completed snapshot and checksum names.
        sync_directory(BACKUPS)
        complete = []
        for entry in BACKUPS.iterdir():
            if NAME.fullmatch(entry.name) and entry.with_suffix('.db.sha256').is_file():
                if entry.is_symlink() or not entry.is_file():
                    raise RuntimeError('unexpected backup entry')
                metadata = entry.with_suffix('.db.sha256').lstat()
                if not stat.S_ISREG(metadata.st_mode) or metadata.st_uid != os.getuid() or stat.S_IMODE(metadata.st_mode) != 0o600:
                    raise RuntimeError('unexpected backup checksum entry')
                complete.append(entry)
        for old in sorted(complete, reverse=True)[RETAIN:]:
            old.unlink()
            old.with_suffix('.db.sha256').unlink()
        sync_directory(BACKUPS)
        print('Accord backup completed; retained snapshots:', min(len(complete), RETAIN))


def sync_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print('Accord backup failed:', type(error).__name__, file=sys.stderr)
        sys.exit(1)
