#!/usr/bin/python3
"""Issue a distinct credential to a private file without emitting its secret."""
import argparse
import os
import pathlib
import pwd
import re
import stat
import subprocess
import sys
import uuid

STATE = pathlib.Path('/var/lib/accord')
ISSUED = STATE / 'issued'


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--owner', required=True)
    parser.add_argument('--machine', required=True)
    parser.add_argument('--days', type=int, default=30)
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on the Linux server')
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,63}', args.machine):
        raise RuntimeError('use a machine label of 1..64 letters, digits, dots, dashes or underscores')
    if not args.owner.strip() or len(args.owner) > 128 or any(ord(c) < 32 for c in args.owner):
        raise RuntimeError('owner must be a nonempty name without control characters')
    if not 1 <= args.days <= 365:
        raise RuntimeError('days must be 1..365')
    account = pwd.getpwnam('accord')
    os.umask(0o077)
    parent = STATE.lstat()
    if not stat.S_ISDIR(parent.st_mode) or parent.st_uid != account.pw_uid or stat.S_IMODE(parent.st_mode) != 0o700:
        raise RuntimeError('state directory ownership or permissions differ')
    if not ISSUED.exists():
        ISSUED.mkdir(mode=0o700)
        os.chown(ISSUED, account.pw_uid, account.pw_gid)
    parent = ISSUED.lstat()
    if not stat.S_ISDIR(parent.st_mode) or parent.st_uid != account.pw_uid or stat.S_IMODE(parent.st_mode) != 0o700:
        raise RuntimeError('issuance directory must be owned and private')
    token = ISSUED / (args.machine + '-' + uuid.uuid4().hex + '.token')
    subprocess.run(['runuser', '-u', 'accord', '--', '/opt/accord/current/agent-board',
                    'token', 'create', '--db', str(STATE / 'board.db'), '--owner', args.owner,
                    '--machine', args.machine, '--days', str(args.days), '--output', str(token)],
                   check=True, stdout=subprocess.DEVNULL)
    info = token.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != account.pw_uid or stat.S_IMODE(info.st_mode) != 0o600:
        raise RuntimeError('issued credential permissions differ')
    print('Private credential file:', token)
    print('Transfer only through an authenticated encrypted channel; never paste its content.')


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, KeyError, subprocess.SubprocessError) as error:
        print('Accord token issuance failed:', type(error).__name__, file=sys.stderr)
        sys.exit(1)
