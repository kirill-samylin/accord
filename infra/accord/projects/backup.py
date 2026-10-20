#!/usr/bin/env python3
"""Consistent registry and all-board backup, with the established maintenance lock."""
import fcntl
import os
from pathlib import Path
import signal
import sqlite3
import subprocess

from guard import require_capable
from layout import LEGACY
from snapshot import snapshot
from launch import require_backup, require_launch


def interrupted(_signal, _frame):
    raise RuntimeError('all-project snapshot interrupted')


def run():
    os.umask(0o077)
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    state = Path('/var/lib/accord')
    projects = state / 'projects-data'
    backups = Path('/var/backups/accord-web')
    LEGACY.private(state, True)
    LEGACY.private(backups, True)
    require_capable(Path('/opt/accord/current/agent-board'), projects)
    require_launch()
    require_backup()
    descriptor = os.open(state / '.maintenance.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        active = subprocess.run(['systemctl', 'is-active', '--quiet', 'accord.service'],
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0
        if not active:
            raise RuntimeError('all-project snapshot requires healthy running writer')
        try:
            subprocess.run(['systemctl', 'stop', 'accord.service'], check=True)
            snapshot(state / 'board.db', state / 'web', projects, backups)
        finally:
            subprocess.run(['systemctl', 'start', 'accord.service'], check=True)
    print('Registry and all project snapshots verified; private format2 copy published.')


if __name__ == '__main__':
    try:
        run()
    except (OSError, RuntimeError, ValueError, sqlite3.Error, subprocess.SubprocessError):
        raise SystemExit('All-project backup failed; inspect private state. Writer restart was attempted.')
