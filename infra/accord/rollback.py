#!/usr/bin/python3
"""Switch to an installed verified executable; never silently roll back database data."""
import argparse
import fcntl
import hashlib
import os
import pathlib
import re
import stat
import subprocess
import sys
from recovery import activate, check_guard, healthy, switch

BASE = pathlib.Path('/opt/accord')
STATE = pathlib.Path('/var/lib/accord')


def command(*args):
    subprocess.run(args, check=True, stdout=subprocess.DEVNULL)


def regular_root(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
        raise RuntimeError('release must contain root-owned regular files')


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on the Linux server')
    if not re.fullmatch(r'[0-9a-f]{40}', args.revision):
        raise RuntimeError('revision must be a full installed Git commit SHA')
    release = BASE / 'releases' / args.revision
    info = release.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
        raise RuntimeError('release directory must be root-owned and immutable to service')
    binary, checksum = release / 'agent-board', release / 'SHA256'
    regular_root(binary)
    regular_root(checksum)
    value = hashlib.sha256()
    with binary.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(block)
    if value.hexdigest() != checksum.read_text(encoding='ascii').strip():
        raise RuntimeError('release SHA256 mismatch')
    fd = os.open('/run/lock/accord-admin.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        previous = os.readlink(BASE / 'current')
        if not re.fullmatch(r'releases/[0-9a-f]{40}', previous):
            raise RuntimeError('current must be a managed release symlink')
        if previous == 'releases/' + args.revision:
            raise RuntimeError('requested executable is already current')
        check_guard(STATE)
        activate(previous, args.revision, STATE, command,
                 lambda revision: switch(BASE, revision), healthy)
        print('Accord executable rolled back; database history preserved:', args.revision)


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print('Accord rollback failed:', str(error), file=sys.stderr)
        sys.exit(1)
