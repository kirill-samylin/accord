"""Freeze installed operators before replacing only reviewed helper entries."""
import json
import re

from host_ops import ADMIN, BACKUP_DROPIN, DROPIN, HERE, LEGACY_DROPIN, PROJECTS, WORK, host
from launch import require_backup, require_launch


def reject_compiled_helpers():
    for folder in (ADMIN / 'projects', ADMIN / 'next', ADMIN / 'web'):
        if any(folder.rglob('__pycache__')) or any(folder.rglob('*.pyc')):
            raise RuntimeError('compiled operator caches require manual inspection')


def baseline_helpers(candidate):
    reject_compiled_helpers()
    result = {}
    for relative in candidate:
        path = ADMIN / relative
        if path.exists() or path.is_symlink():
            host.regular(path)
            result[relative] = host.sha(path)
    alias = ADMIN / 'web-backup.py'
    host.regular(alias)
    if host.sha(alias) != result.get('web/backup.py'):
        raise RuntimeError('legacy backup alias differs from installed reviewed helper')
    result['web-backup.py'] = host.sha(alias)
    previous = host.pointer(host.GO)
    revision = previous.rsplit('/', 1)[-1] if previous else ''
    if not re.fullmatch('[0-9a-f]{40}', revision):
        raise RuntimeError('known capable release pointer required')
    journal = WORK / revision / 'state.json'
    host.regular(journal)
    state = json.loads(journal.read_text())
    if state.get('revision') != revision or state.get('phase') != 'complete':
        raise RuntimeError('previous capable release must be complete')
    if any(result.get(relative) != digest for relative, digest in state['helpers'].items()):
        raise RuntimeError('installed helpers differ from completed previous release')
    for relative, digest in result.items():
        if relative == 'web-backup.py' or relative in state['helpers']:
            continue
        if candidate.get(relative) != digest:
            raise RuntimeError('unknown pre-existing new helper path requires inspection')
    return result


def require_runtime():
    require_launch()
    require_backup()
    if LEGACY_DROPIN.exists() or LEGACY_DROPIN.is_symlink():
        raise RuntimeError('unexpected superseded launch override')
    for installed, name in ((DROPIN, 'accord-projects.conf'),
                            (BACKUP_DROPIN, 'accord-projects-backup.conf')):
        host.regular(installed)
        if installed.read_bytes() != (HERE / name).read_bytes():
            raise RuntimeError('known project runtime and backup overrides required')
    host.regular(PROJECTS / 'MULTIPROJECT_REQUIRES_CAPABILITY')


def require_helpers(state, after=False):
    reject_compiled_helpers()
    for relative, digest in state['helpers'].items():
        path = ADMIN / relative
        baseline = state['installed_helpers'].get(relative)
        if not path.exists() and not path.is_symlink() and baseline is None and not after:
            continue
        host.regular(path)
        allowed = {digest} if after else {digest, baseline}
        if host.sha(path) not in allowed:
            raise RuntimeError('installed operator drift requires inspection')
    alias = ADMIN / 'web-backup.py'
    host.regular(alias)
    allowed = {state['helpers']['web/backup.py']}
    if not after:
        allowed.add(state['installed_helpers']['web-backup.py'])
    if host.sha(alias) not in allowed:
        raise RuntimeError('legacy backup alias drift requires inspection')
