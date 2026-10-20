"""Reuse reviewed host primitives without ambiguous sibling module imports."""
import importlib.util
from pathlib import Path
import fcntl
import os

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('accord_next_host_primitives', HERE.parent / 'next/host.py')
host = importlib.util.module_from_spec(spec)
spec.loader.exec_module(host)
WORK = Path('/var/backups/accord-projects')
STATE = Path('/var/lib/accord')
PROJECTS = STATE / 'projects-data'
ADMIN = Path('/opt/accord/admin')
DROPIN = Path('/etc/systemd/system/accord.service.d/90-projects.conf')
BACKUP_DROPIN = Path('/etc/systemd/system/accord-web-backup.service.d/90-projects.conf')


def lock(path):
    descriptor = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    handle = os.fdopen(descriptor, 'w')
    fcntl.flock(handle, fcntl.LOCK_EX)
    return handle


def preserved():
    paths = [Path('/etc/accord/server.env'), Path('/etc/accord/web.env'),
             Path('/etc/systemd/system/accord.service'), host.UNIT,
             Path('/etc/nginx/nginx.conf')]
    paths.extend(sorted(Path('/etc/systemd/system/accord.service.d').glob('*.conf')))
    paths.extend(sorted(Path('/etc/systemd/system').glob('accord*.timer')))
    paths.extend(sorted(Path('/etc/systemd/system').glob('accord*.service')))
    return {str(path): host.sha(path) for path in set(paths) if path.is_file() and path != DROPIN}


def nginx_check(candidate, work):
    source = Path('/etc/nginx/nginx.conf').read_text()
    include = 'include /etc/nginx/conf.d/accord-public.conf;'
    if source.count(include) != 1:
        raise RuntimeError('unexpected nginx include topology')
    target = work / 'nginx-check.conf'
    host.atomic(target, source.replace(include, 'include ' + str(candidate) + ';').encode(), 0o600)
    host.run('nginx', '-t', '-c', str(target))


def install_helpers(expected):
    for relative, digest in expected.items():
        source = HERE.parent / relative
        host.regular(source)
        if host.sha(source) != digest:
            raise RuntimeError('reviewed operator helper changed')
        destination = ADMIN / relative
        host.directory(destination.parent)
        host.atomic(destination, source.read_bytes())
        if relative == 'web/backup.py':
            host.atomic(ADMIN / 'web-backup.py', source.read_bytes())
    host.directory(DROPIN.parent)
    host.directory(BACKUP_DROPIN.parent)
    host.atomic(DROPIN, (HERE / 'accord-projects.conf').read_bytes())
    host.atomic(BACKUP_DROPIN, (HERE / 'accord-projects-backup.conf').read_bytes())


def helper_manifest():
    files = [path for path in HERE.glob('*') if path.suffix in ('.py', '.conf')]
    files.extend((HERE.parent / 'web/backup.py', HERE.parent / 'next/host.py',
                  HERE.parent / 'restore.py'))
    result = {}
    for path in files:
        host.regular(path)
        result[path.relative_to(HERE.parent).as_posix()] = host.sha(path)
    return result
