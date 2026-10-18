#!/usr/bin/python3
"""Offline restore: preserve current state and leave service blocked for token reset."""
import argparse
import datetime
import fcntl
import hashlib
import os
import pathlib
import pwd
import re
import shutil
import sqlite3
import stat
import subprocess
import sys
import uuid

STATE = pathlib.Path('/var/lib/accord')
BACKUPS = pathlib.Path('/var/backups/accord')
MARKER = STATE / 'RESTORE_REQUIRES_RESET'
NAME = re.compile(r'^board-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}\.db$')


def command(*args):
    subprocess.run(args, check=True, stdout=subprocess.DEVNULL)


def private(path, owner, directory=False):
    info = path.lstat()
    correct_type = stat.S_ISDIR(info.st_mode) if directory else stat.S_ISREG(info.st_mode)
    expected = 0o700 if directory else 0o600
    if not correct_type or info.st_uid != owner or stat.S_IMODE(info.st_mode) != expected:
        raise RuntimeError('expected owned private non-symlink path')


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(block)
    return value.hexdigest()


def sync_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def verify(path, owner):
    private(path, owner)
    checksum = path.with_suffix('.db.sha256')
    private(checksum, owner)
    expected = checksum.read_text(encoding='ascii').strip()
    if expected != digest(path) + '  ' + path.name:
        raise RuntimeError('backup SHA256 mismatch')
    # Read SQLite through its own API; do not interpret or rewrite application data.
    with sqlite3.connect(path.as_uri() + '?mode=ro&immutable=1', uri=True) as database:
        if database.execute('PRAGMA user_version').fetchone()[0] not in (1, 2, 3, 4):
            raise RuntimeError('unsupported backup schema')
        if database.execute('PRAGMA integrity_check').fetchall() != [('ok',)]:
            raise RuntimeError('backup integrity check failed')


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--backup', type=pathlib.Path, required=True)
    parser.add_argument('--check-only', action='store_true', help='verify copy; no service or data changes')
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on the Linux server')
    account = pwd.getpwnam('accord')
    os.umask(0o077)
    private(STATE, account.pw_uid, directory=True)
    private(BACKUPS, account.pw_uid, directory=True)
    source = args.backup.absolute()
    if source.parent != BACKUPS or not NAME.fullmatch(source.name):
        raise RuntimeError('choose a managed snapshot from /var/backups/accord')
    verify(source, account.pw_uid)
    if args.check_only:
        print('Accord snapshot checksum and SQLite integrity verified.')
        return
    fd = os.open('/run/lock/accord-admin.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        if MARKER.exists() or MARKER.is_symlink():
            raise RuntimeError('previous restore requires resolution')
        # Write the boot guard before stopping anything; failures remain fail-closed.
        with MARKER.open('x', encoding='ascii') as marker:
            marker.write('Restore requires credential reset and lease expiry before restart.\n')
            marker.flush()
            os.fsync(marker.fileno())
        # Persist the boot guard's directory entry before stopping writers or
        # replacing state, so reboot cannot expose a restored active token.
        sync_directory(STATE)
        command('systemctl', 'stop', 'accord-backup.timer')
        command('systemctl', 'stop', 'accord-backup.service')
        command('systemctl', 'stop', 'accord.service')
        staging = STATE / ('.restore-' + uuid.uuid4().hex + '.db')
        with staging.open('xb') as output, source.open('rb') as stream:
            shutil.copyfileobj(stream, output)
            output.flush()
            os.fsync(output.fileno())
        os.chown(staging, account.pw_uid, account.pw_gid)
        if digest(staging) != digest(source):
            raise RuntimeError('copied snapshot mismatch; service remains stopped')
        # Copy before retention can remove an old selected snapshot.
        if (STATE / 'board.db').exists():
            command('runuser', '-u', 'accord', '--', '/usr/bin/python3', '/opt/accord/admin/backup.py')
        quarantine = STATE / ('pre-restore-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex)
        quarantine.mkdir(mode=0o700)
        os.chown(quarantine, account.pw_uid, account.pw_gid)
        # Preserve old DB and its journals as a set, only after both writers stopped.
        for suffix in ('', '-wal', '-shm', '-journal'):
            existing = STATE / ('board.db' + suffix)
            if existing.exists() or existing.is_symlink():
                private(existing, account.pw_uid)
                os.replace(existing, quarantine / existing.name)
        os.replace(staging, STATE / 'board.db')
        sync_directory(STATE)
        print('Accord restored and STOPPED. Boot guard remains active.')
        print('Reset restored credentials and expire leases using docs/deployment.md before removing the guard.')
        print('Preserved previous state directory:', quarantine.name)


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, KeyError, sqlite3.Error, subprocess.SubprocessError) as error:
        print('Accord restore failed:', type(error).__name__, file=sys.stderr)
        sys.exit(1)
