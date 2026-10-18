"""Explicit private advertised authority; never infer trust from proxy headers."""
import os
import pathlib
import re
import stat

CONFIG = pathlib.Path('/etc/accord')
NAME = 'server.env'
PREFIX = 'ACCORD_EXTERNAL_HOST='


def validate(authority):
    # Match transport.ValidateExternalHost: lower ASCII DNS + canonical port.
    host, separator, port = authority.rpartition(':')
    if not separator or not re.fullmatch(r'[1-9][0-9]{0,4}', port) or int(port) > 65535:
        raise RuntimeError('external host requires canonical DNS:port')
    labels = host.split('.')
    if (len(host) > 253 or len(labels) < 2
            or not re.search(r'[a-z]', labels[-1])
            or any(not re.fullmatch(r'[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?', label) for label in labels)):
        raise RuntimeError('external host requires canonical DNS:port')


def existing(path):
    if not path.exists() and not path.is_symlink():
        return None
    info = path.lstat()
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_gid != 0
            or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > 512):
        raise RuntimeError('unsafe existing Accord server configuration')
    try:
        content = path.read_text(encoding='ascii')
    except UnicodeError:
        raise RuntimeError('unmanaged Accord server configuration; inspect manually') from None
    if not content.startswith(PREFIX) or not content.endswith('\n') or content.count('\n') != 1:
        raise RuntimeError('unmanaged Accord server configuration; inspect manually')
    validate(content[len(PREFIX):-1])
    return content


def configure(authority, directory):
    """Preserve omitted config; replace only the exact managed one-line format.

    Root-owned ancestors and directory are checked by install.directory. Both
    new file and parent directory are synced before activation may start.
    """
    if authority is not None:
        validate(authority)
    if authority is None and not CONFIG.exists() and not CONFIG.is_symlink():
        return
    directory(CONFIG, 0o700)
    path = CONFIG / NAME
    old = existing(path)
    if authority is None or old == PREFIX + authority + '\n':
        return
    staging = CONFIG / ('.' + NAME + '.new')
    fd = os.open(staging, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    try:
        with os.fdopen(fd, 'w', encoding='ascii') as output:
            output.write(PREFIX + authority + '\n')
            output.flush()
            os.fsync(output.fileno())
        os.replace(staging, path)
        descriptor = os.open(CONFIG, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
    finally:
        # A failed exclusive open never enters this block; do not remove a
        # foreign staging file. Installed configuration is never rolled back.
        if staging.exists():
            staging.unlink()
