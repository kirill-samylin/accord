#!/usr/bin/python3
"""Install a reviewed, hash-pinned Linux binary; never modify another service."""
import argparse
import fcntl
import grp
import hashlib
import os
import pathlib
import pwd
import re
import shutil
import stat
import subprocess
import sys
from recovery import activate, check_guard, healthy, switch
from host_config import configure, validate

BASE = pathlib.Path('/opt/accord')
STATE = pathlib.Path('/var/lib/accord')
BACKUPS = pathlib.Path('/var/backups/accord')
ADMIN_FILES = ('backup.py', 'restore.py', 'token.py', 'publish.py', 'rollback.py', 'recovery.py', 'host_config.py')
UNIT_FILES = ('accord.service', 'accord-backup.service', 'accord-backup.timer')


def command(*args):
    return subprocess.run(args, check=True, stdout=subprocess.DEVNULL)


def regular(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode):
        raise RuntimeError('expected a regular non-symlink file')
    return info


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(block)
    return value.hexdigest()


def directory(path, mode, uid=0, gid=0):
    # Never follow a directory planted at any ancestor by another account.
    for ancestor in reversed(path.parents):
        info = ancestor.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
            raise RuntimeError('unsafe directory ancestor')
    if not path.exists():
        path.mkdir(mode=mode)
        os.chmod(path, mode)
        os.chown(path, uid, gid)
    info = path.lstat()
    if (not stat.S_ISDIR(info.st_mode) or info.st_uid != uid or info.st_gid != gid
            or stat.S_IMODE(info.st_mode) != mode):
        raise RuntimeError('existing directory ownership or permissions differ')


def install_file(source, target, mode=0o644):
    regular(source)
    if target.is_symlink():
        raise RuntimeError('refusing symlink destination')
    temporary = target.with_name('.' + target.name + '.new')
    with temporary.open('xb') as output, source.open('rb') as stream:
        shutil.copyfileobj(stream, output)
        output.flush()
        os.fsync(output.fileno())
    os.chmod(temporary, mode)
    os.replace(temporary, target)


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=pathlib.Path, required=True)
    parser.add_argument('--sha256', required=True)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--assets', type=pathlib.Path, required=True)
    parser.add_argument('--external-host', help='exact advertised lower-case DNS:port; omission preserves private config')
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on Linux with systemd')
    if args.external_host is not None:
        validate(args.external_host)
    if not re.fullmatch(r'[0-9a-f]{40}', args.revision):
        raise RuntimeError('revision must be a full Git commit SHA')
    if not re.fullmatch(r'[0-9a-f]{64}', args.sha256):
        raise RuntimeError('expected an independently checked SHA256')
    regular(args.binary)
    if digest(args.binary) != args.sha256:
        raise RuntimeError('binary SHA256 mismatch')
    for name in ADMIN_FILES + UNIT_FILES:
        regular(args.assets / name)
    os.umask(0o077)
    # Admin operations serialize with each other; snapshots have their own lock.
    fd = os.open('/run/lock/accord-admin.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        try:
            account = pwd.getpwnam('accord')
        except KeyError:
            command('useradd', '--system', '--user-group', '--home-dir', str(STATE),
                    '--shell', '/usr/sbin/nologin', 'accord')
            account = pwd.getpwnam('accord')
        if (account.pw_uid == 0 or account.pw_shell != '/usr/sbin/nologin'
                or account.pw_dir != str(STATE) or grp.getgrgid(account.pw_gid).gr_name != 'accord'):
            raise RuntimeError('existing accord account differs from service account contract')
        directory(BASE, 0o755)
        directory(BASE / 'releases', 0o755)
        directory(BASE / 'admin', 0o755)
        directory(STATE, 0o700, account.pw_uid, account.pw_gid)
        directory(BACKUPS, 0o700, account.pw_uid, account.pw_gid)
        check_guard(STATE)
        configure(args.external_host, directory)
        release = BASE / 'releases' / args.revision
        directory(release, 0o755)
        binary = release / 'agent-board'
        if binary.exists():
            info = regular(binary)
            if info.st_uid != 0 or stat.S_IMODE(info.st_mode) != 0o755 or digest(binary) != args.sha256:
                raise RuntimeError('immutable existing release differs')
        else:
            install_file(args.binary, binary, 0o755)
        if digest(binary) != args.sha256:
            raise RuntimeError('installed binary verification failed')
        checksum = release / 'SHA256'
        if checksum.exists():
            regular(checksum)
            if checksum.read_text(encoding='ascii').strip() != args.sha256:
                raise RuntimeError('release checksum metadata differs')
        else:
            with checksum.open('x', encoding='ascii') as stream:
                stream.write(args.sha256 + '\n')
                stream.flush()
                os.fsync(stream.fileno())
            checksum.chmod(0o644)
        previous = None
        current = BASE / 'current'
        if current.is_symlink():
            previous = os.readlink(current)
            if not re.fullmatch(r'releases/[0-9a-f]{40}', previous):
                raise RuntimeError('unexpected current target')
        elif current.exists():
            raise RuntimeError('current must be a managed release symlink')
        # Copy the reviewed operations code and units; they stay root-owned.
        for name in ADMIN_FILES:
            install_file(args.assets / name, BASE / 'admin' / name)
        for name in UNIT_FILES:
            install_file(args.assets / name, pathlib.Path('/etc/systemd/system') / name)
        command('systemctl', 'daemon-reload')
        activate(previous, args.revision, STATE, command,
                 lambda revision: switch(BASE, revision), healthy)
        command('systemctl', 'enable', 'accord.service')
        command('systemctl', 'enable', '--now', 'accord-backup.timer')
        command('systemctl', 'start', 'accord-backup.service')
        print('Accord installed and locally ready:', args.revision)
        print('Private HTTPS publication is a separate, reviewed step.')


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print('Accord installation failed:', str(error), file=sys.stderr)
        sys.exit(1)
