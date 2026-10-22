#!/usr/bin/env python3
"""Fail closed before a selected executable can ignore an activated registry."""
import argparse
import json
import os
from pathlib import Path
import stat
import subprocess

MARKER = 'MULTIPROJECT_REQUIRES_CAPABILITY'


def protected_root(root):
    if not root.is_absolute():
        raise RuntimeError('absolute projects directory required')
    for entry in reversed((root,) + tuple(root.parents)):
        try:
            info = entry.lstat()
        except FileNotFoundError:
            continue
        if (not stat.S_ISDIR(info.st_mode) or
                (info.st_mode & 0o022 and str(entry) not in ('/tmp', '/private/tmp'))):
            raise RuntimeError('unsafe projects directory ancestor')
    if root.exists() and stat.S_IMODE(root.stat().st_mode) != 0o700:
        raise RuntimeError('projects directory must be private')
    marker = root / MARKER
    if marker.exists() or marker.is_symlink():
        info = marker.lstat()
        if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600:
            raise RuntimeError('unsafe project capability marker')


def require_capable(binary, root):
    protected_root(root)
    info = binary.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o022:
        raise RuntimeError('protected regular executable required')
    result = subprocess.run([str(binary), 'capabilities', '--json'], capture_output=True,
                            timeout=5, check=True, env={'PATH': '/usr/bin:/bin', 'LANG': 'C'})
    if len(result.stdout) > 4096 or result.stderr:
        raise RuntimeError('unexpected capability response')
    capabilities = json.loads(result.stdout)
    if (type(capabilities.get('project_registry')) is not int or
            capabilities['project_registry'] != 1 or
            type(capabilities.get('project_acl')) is not int or
            capabilities['project_acl'] != 1 or
            capabilities.get('registry_schema') != [1] or
            type(capabilities['registry_schema'][0]) is not int or
            type(capabilities.get('agent_profiles')) is not int or
            capabilities['agent_profiles'] != 1 or
            type(capabilities.get('coordination')) is not int or
            capabilities['coordination'] != 1 or
            type(capabilities.get('deduplication')) is not int or
            capabilities['deduplication'] != 1 or
            capabilities.get('board_schema') != [4] or
            type(capabilities['board_schema'][0]) is not int):
        raise RuntimeError('selected executable cannot enforce project isolation')
    return capabilities


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--projects-data', required=True, type=Path)
    parser.add_argument('--unit', choices=('accord.service',))
    args = parser.parse_args()
    require_capable(args.binary, args.projects_data)
    if args.unit:
        from launch import BINARY, PROJECTS, require_backup, require_launch
        if str(args.binary) != BINARY or str(args.projects_data) != PROJECTS:
            raise RuntimeError('systemd guard requires canonical service paths')
        require_launch()
        require_backup()
    print('Project registry and ACL executable capability verified.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, RuntimeError, ValueError, subprocess.SubprocessError):
        raise SystemExit('Project isolation startup guard rejected the executable or state.')
