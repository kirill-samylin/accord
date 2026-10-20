"""Small host primitives shared by the guarded Next release operator."""
import hashlib
import json
import os
import pathlib
import re
import stat
import subprocess
import time
import urllib.request

BASE = pathlib.Path('/opt/accord/next')
GO = pathlib.Path('/opt/accord')
NGINX = pathlib.Path('/etc/nginx/conf.d/accord-public.conf')
UNIT = pathlib.Path('/etc/systemd/system/accord-next.service')
BACKUPS = pathlib.Path('/var/backups/accord-next')


def run(*args):
    return subprocess.run(args, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True).stdout


def sha(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            value.update(block)
    return value.hexdigest()


def regular(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
        raise RuntimeError('expected protected root-owned regular file')
    return info


def directory(path, mode=0o755):
    for ancestor in reversed((path,) + tuple(path.parents)):
        if not ancestor.exists():
            ancestor.mkdir(mode=mode if ancestor == path else 0o755)
        info = ancestor.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
            raise RuntimeError('unsafe root-owned directory or ancestor')
    path.chmod(mode)


def atomic(path, data, mode=0o644):
    if path.exists() or path.is_symlink():
        regular(path)
    temporary = path.with_name('.' + path.name + '.next-new')
    if temporary.exists() or temporary.is_symlink():
        info = regular(temporary)
        if stat.S_IMODE(info.st_mode) != mode or temporary.read_bytes() != data:
            raise RuntimeError('unowned or incomplete atomic temporary requires inspection')
        # A killed previous write may leave the fully fsynced expected bytes.
        # Never discard a different or partial operator artifact on resume.
        temporary.unlink()
        sync(path.parent)
    descriptor = os.open(temporary, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, mode)
    try:
        with os.fdopen(descriptor, 'wb') as output:
            output.write(data)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, path)
        sync(path.parent)
    finally:
        if temporary.exists():
            temporary.unlink()


def sync(path):
    descriptor = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def pointer(base):
    current = base / 'current'
    if not current.exists() and not current.is_symlink():
        return None
    if not current.is_symlink():
        raise RuntimeError('current is not a managed release pointer')
    target = os.readlink(current)
    if not valid_target(base, target):
        raise RuntimeError('unexpected current target')
    return target


def valid_target(base, target):
    relative = r'releases/[0-9a-f]{40}'
    return bool(re.fullmatch(relative, target) or
                re.fullmatch(re.escape(str(base)) + '/' + relative, target))


def switch(base, target):
    pointer(base)
    current = base / 'current'
    if target is None:
        current.unlink(missing_ok=True)
    else:
        if not valid_target(base, target):
            raise RuntimeError('invalid release pointer')
        temporary = base / '.current.next-new'
        if temporary.exists() or temporary.is_symlink():
            info = temporary.lstat()
            if not stat.S_ISLNK(info.st_mode) or info.st_uid != 0 or os.readlink(temporary) != target:
                raise RuntimeError('unowned release pointer temporary requires inspection')
            temporary.unlink()
            sync(base)
        temporary.symlink_to(target)
        os.replace(temporary, current)
    sync(base)


def verify_bundle(path, expected, revision):
    regular(path / 'manifest.json')
    if not re.fullmatch(r'[0-9a-f]{64}', expected) or sha(path / 'manifest.json') != expected:
        raise RuntimeError('bundle manifest hash mismatch')
    manifest = json.loads((path / 'manifest.json').read_text())
    if manifest.get('format') != 1 or manifest.get('revision') != revision:
        raise RuntimeError('bundle format/source mismatch')
    files = set()
    for entry in path.rglob('*'):
        info = entry.lstat()
        if stat.S_ISDIR(info.st_mode):
            if info.st_uid != 0 or info.st_mode & 0o022:
                raise RuntimeError('unsafe bundle directory')
        elif stat.S_ISREG(info.st_mode):
            regular(entry)
            files.add(str(entry.relative_to(path)))
        else:
            raise RuntimeError('non-regular member in immutable release')
    if files != set(manifest['files']) | {'manifest.json'}:
        raise RuntimeError('bundle file set differs')
    for name, digest in manifest['files'].items():
        relative = pathlib.PurePosixPath(name)
        if relative.is_absolute() or '..' in relative.parts or sha(path / name) != digest:
            raise RuntimeError('bundle member hash/path mismatch')
    source = json.loads((path / 'SOURCE.json').read_text())
    if (source.get('revision') != revision or source.get('node_version') != 'v24.19.0'
            or source.get('next_version') != '16.4.2' or source.get('go_origin') != 'http://127.0.0.1:7892'):
        raise RuntimeError('runtime source contract differs')
    return manifest


def ready(port, path='/readyz', attempts=45, nonce=False):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for _ in range(attempts):
        try:
            with opener.open('http://127.0.0.1:' + str(port) + path, timeout=3) as response:
                data = response.read(2 << 20)
                csp = response.headers.get('Content-Security-Policy', '')
                if response.status == 200 and (not nonce or (b'/_next/static/' in data and "'nonce-" in csp)):
                    return True
        except (OSError, TimeoutError):
            pass
        time.sleep(1)
    raise RuntimeError('loopback readiness/nonce/assets did not become healthy')


def save_json(path, value):
    atomic(path, (json.dumps(value, sort_keys=True, indent=2) + '\n').encode(), 0o600)
