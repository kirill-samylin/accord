#!/usr/bin/env python3
"""Restore format2 only into a NEW private offline staging; never replace live data."""
import argparse
import json
import os
from pathlib import Path
import sqlite3
import subprocess

from guard import require_capable
from layout import LEGACY, copy_file
from validate import validate


def copy_tree(source, target):
    LEGACY.private(source, True)
    target.mkdir(mode=0o700)
    LEGACY.sync_dir(target.parent)
    for path in sorted(source.iterdir()):
        if path.is_dir() and not path.is_symlink():
            copy_tree(path, target / path.name)
        else:
            copy_file(path, target / path.name)
    LEGACY.sync_dir(target)


def marker(path, message):
    descriptor = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, 'w') as output:
        output.write(message + '\n')
        output.flush()
        os.fsync(output.fileno())
    LEGACY.sync_dir(path.parent)


def restore(source, target, binary):
    source, target = source.absolute(), target.absolute()
    if (target == source or source in target.parents or target in source.parents or
            target.exists() or target.is_symlink()):
        raise RuntimeError('choose a new offline staging outside the immutable snapshot')
    # The same capability guard rejects writable/symlink ancestry before creating
    # anything. The snapshot is verified again after copying, before auth reset.
    require_capable(binary, target / 'projects-data')
    manifest = validate(source)
    LEGACY.private(target.parent, True)
    target.mkdir(mode=0o700)
    LEGACY.sync_dir(target.parent)
    marker(target / 'RESTORE_REQUIRES_RESET', 'Offline restored state; no listener until recovery checks and new credentials.')
    copy_file(source / 'default/board.db', target / 'board.db')
    copy_tree(source / 'default/web', target / 'web')
    projects = target / 'projects-data'
    copy_tree(source / 'registry', projects)
    copy_tree(source / 'projects', projects / 'projects')
    if manifest.get('quarantine'):
        copy_tree(source / 'quarantine', projects / 'quarantine')
    for relative, digest in manifest['files'].items():
        if relative.startswith('default/'):
            copied = target / relative.removeprefix('default/')
        elif relative.startswith('registry/'):
            copied = projects / relative.removeprefix('registry/')
        else:
            copied = projects / relative
        if LEGACY.sha(copied) != digest:
            raise RuntimeError('copied offline restore differs; staging remains blocked')
    result = subprocess.run([str(binary), 'projects', 'maintenance-reset',
                             '--projects-data', str(projects), '--db', str(target / 'board.db'),
                             '--web-data', str(target / 'web'), '--reason', 'restore'],
                            capture_output=True, timeout=120, check=True,
                            env={'PATH': '/usr/bin:/bin', 'LANG': 'C'})
    if len(result.stdout) > 4096 or result.stderr:
        raise RuntimeError('unexpected offline reset output; staging remains blocked')
    if json.loads(result.stdout).get('restored_state_reset') is not True:
        raise RuntimeError('offline reset was not confirmed; staging remains blocked')
    LEGACY.sync_dir(projects)
    LEGACY.sync_dir(target / 'web')
    marker(target / 'RESTORE_AUTH_RESET_COMPLETE', json.dumps({'registry_schema': 1, 'auth_reset': True}))
    # Keep the boot guard even after reset: replacement credentials and isolated
    # logical-data/runtime checks are distinct mandatory operator steps.
    return target


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--snapshot', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    restore(args.snapshot, args.output, args.binary)
    print('New offline staging copied and credentials reset; boot guard remains. Live data and source snapshot untouched.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, RuntimeError, ValueError, KeyError, sqlite3.Error, subprocess.SubprocessError):
        raise SystemExit('All-project offline restore failed; inspect blocked private staging. Live state was not replaced.')
