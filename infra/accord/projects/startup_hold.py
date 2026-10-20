"""An owned durable marker blocks the existing cold-boot guard during cutover."""
import json
import os
from pathlib import Path
import re
import stat


def marker_bytes(revision):
    if not re.fullmatch('[0-9a-f]{40}', revision):
        raise RuntimeError('full release revision required for recovery marker')
    return (json.dumps({'schema': 1, 'reason': 'TASK009 effective-launch recovery',
                        'revision': revision}, sort_keys=True) + '\n').encode()


def sync(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def owned_marker(state, revision):
    path = Path(state) / 'START_REQUIRES_INSPECTION'
    try:
        info = path.lstat()
    except FileNotFoundError:
        return False
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or
            stat.S_IMODE(info.st_mode) != 0o600 or path.read_bytes() != marker_bytes(revision)):
        raise RuntimeError('unowned startup inspection marker; preserve for manual inspection')
    return True


def hold_startup(state, revision):
    if owned_marker(state, revision):
        return
    path = Path(state) / 'START_REQUIRES_INSPECTION'
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as output:
        output.write(marker_bytes(revision))
        output.flush()
        os.fsync(output.fileno())
    sync(path.parent)


def release_startup(state, revision):
    if not owned_marker(state, revision):
        raise RuntimeError('owned startup guard required before resuming')
    path = Path(state) / 'START_REQUIRES_INSPECTION'
    path.unlink()
    sync(path.parent)
