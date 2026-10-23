"""Strict offline registry inventory, never names, repository URLs or credentials."""
from contextlib import closing
import importlib.util
from pathlib import Path
import re
import sqlite3

from guard import MARKER, protected_root
from migration_files import PATTERN, copy_migrations, validate_migrations

PROJECT_ID = re.compile(r'p_[0-9a-f]{32}')


def legacy_module():
    # All-project tooling is installed beside the existing reviewed web snapshot
    # primitives. Sources and installed layout have the same relative structure.
    path = Path(__file__).resolve().parent.parent / 'web' / 'backup.py'
    spec = importlib.util.spec_from_file_location('accord_web_snapshot_primitives', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


LEGACY = legacy_module()


def readonly(path):
    LEGACY.private(path)
    for suffix in ('-wal', '-journal'):
        journal = path.with_name(path.name + suffix)
        if journal.exists() or journal.is_symlink():
            if LEGACY.private(journal):
                raise RuntimeError('offline database still has an unresolved transaction journal')
    return sqlite3.connect(path.absolute().as_uri() + '?mode=ro&immutable=1', uri=True)


def inventory(root, default_db, default_web):
    protected_root(root)
    LEGACY.private(root, True)
    LEGACY.private(root / MARKER)
    for entry in root.iterdir():
        if entry.name in ('projects', 'quarantine'):
            LEGACY.private(entry, True)
        elif entry.name in (MARKER, 'registry.sqlite', 'registry.sqlite-wal',
                            'registry.sqlite-shm', 'registry.sqlite-journal'):
            LEGACY.private(entry)
        else:
            raise RuntimeError('unaccounted registry entry requires recovery')
    with closing(readonly(root / 'registry.sqlite')) as database:
        if (database.execute('PRAGMA user_version').fetchone()[0] != 1 or
                database.execute('PRAGMA integrity_check').fetchone()[0] != 'ok'):
            raise RuntimeError('unsupported or corrupt registry')
        rows = database.execute('SELECT default_id,default_db,default_web FROM registry_meta WHERE id=1').fetchall()
        if len(rows) != 1:
            raise RuntimeError('registry default binding missing')
        default_id, stored_db, stored_web = rows[0]
        if (stored_db != str(default_db.absolute()) or stored_web != str(default_web.absolute()) or
                not PROJECT_ID.fullmatch(default_id)):
            raise RuntimeError('default data binding differs from operator inventory')
        projects = database.execute('SELECT id,state FROM projects ORDER BY id').fetchall()
    result = []
    found_default = False
    for identifier, state in projects:
        if not PROJECT_ID.fullmatch(identifier) or state not in ('active', 'archived', 'provisioning'):
            raise RuntimeError('unknown project identity or lifecycle')
        if identifier == default_id:
            found_default = True
            continue
        folder = root / 'projects' / identifier
        if state != 'provisioning' or folder.exists() or folder.is_symlink():
            LEGACY.private(folder, True)
        if state != 'provisioning':
            for path in folder.iterdir():
                if path.name == 'web':
                    LEGACY.private(path, True)
                elif path.name == 'board.db.migrations':
                    validate_migrations(path, LEGACY)
                elif path.name in ('board.db', 'board.db-wal', 'board.db-shm', 'board.db-journal'):
                    LEGACY.private(path)
                else:
                    raise RuntimeError('unaccounted active project entry requires recovery')
        result.append({'id': identifier, 'state': state, 'folder': folder})
    if not found_default:
        raise RuntimeError('default project is not registered')
    expected = {entry['id'] for entry in result}
    parent = root / 'projects'
    if parent.exists():
        LEGACY.private(parent, True)
        actual = {entry.name for entry in parent.iterdir()}
        required = {entry['id'] for entry in result if entry['state'] != 'provisioning'}
        if not required <= actual <= expected:
            raise RuntimeError('unaccounted project or orphan directory requires recovery')
    elif any(entry['state'] != 'provisioning' for entry in result):
        raise RuntimeError('registered project storage is missing')
    quarantined = []
    quarantine = root / 'quarantine'
    if quarantine.exists():
        for folder in sorted(quarantine.iterdir()):
            if not PROJECT_ID.fullmatch(folder.name) or folder.name == default_id or folder.name in expected:
                raise RuntimeError('unexpected quarantined project identity')
            LEGACY.private(folder, True)
            quarantined.append(folder)
        if len(quarantined) > 128:
            raise RuntimeError('quarantine inventory needs operator maintenance')
    return {'default_id': default_id, 'projects': result, 'quarantine': quarantined}


def copy_database(source, target, allowed):
    with closing(readonly(source)) as database:
        version = database.execute('PRAGMA user_version').fetchone()[0]
    if version not in allowed:
        raise RuntimeError('unsupported snapshot database schema')
    LEGACY.db_copy(source, target, version)
    return version


def copy_file(source, target):
    LEGACY.private(source)
    import os
    import shutil
    source_fd = os.open(source, os.O_RDONLY | os.O_NOFOLLOW)
    try:
        target_fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    except BaseException:
        os.close(source_fd)
        raise
    with os.fdopen(source_fd, 'rb') as input_stream, os.fdopen(target_fd, 'wb') as output:
        shutil.copyfileobj(input_stream, output, 1 << 20)
        output.flush()
        os.fsync(output.fileno())
    if LEGACY.sha(source) != LEGACY.sha(target):
        raise RuntimeError('offline copy changed during snapshot')


def board_copy(board, web, destination):
    LEGACY.private(web, True)
    for path in web.iterdir():
        if path.name in ('blobs', 'staging'):
            LEGACY.private(path, True)
        elif path.name in ('web.db', 'web.db-wal', 'web.db-shm', 'web.db-journal'):
            LEGACY.private(path)
        else:
            raise RuntimeError('unaccounted web storage entry requires recovery')
    destination.mkdir(mode=0o700)
    (destination / 'web').mkdir(mode=0o700)
    board_version = copy_database(board, destination / 'board.db', (2, 3, 4, 5, 6))
    copy_migrations(board.with_name(board.name + '.migrations'),
                    destination / 'board.db.migrations', LEGACY, copy_file)
    web_version = copy_database(web / 'web.db', destination / 'web/web.db', (1, 2))
    for name in ('blobs', 'staging'):
        source = web / name
        LEGACY.private(source, True)
        target = destination / 'web' / name
        target.mkdir(mode=0o700)
        for path in source.iterdir():
            if not LEGACY.BLOB.fullmatch(path.name):
                raise RuntimeError('unexpected private design entry')
            copy_file(path, target / path.name)
        LEGACY.sync_dir(target)
    LEGACY.sync_dir(destination / 'web')
    LEGACY.sync_dir(destination)
    return {'board_schema': board_version, 'web_schema': web_version}


def pending_copy(source, target):
    """Preserve a closed unpublished provisioning set, including raw journals.

    No public board request can reach this state. Registry recovery must finish
    or reject it before readiness; it is not presented as a usable board snapshot.
    """
    target.mkdir(mode=0o700)
    if not source.exists():
        return
    for path in sorted(source.rglob('*')):
        relative = path.relative_to(source)
        name = relative.as_posix()
        if path.is_dir() and not path.is_symlink():
            LEGACY.private(path, True)
            if name not in ('web', 'web/blobs', 'web/staging', 'board.db.migrations'):
                raise RuntimeError('unexpected provisioning directory')
            (target / relative).mkdir(mode=0o700)
        else:
            LEGACY.private(path)
            if not allowed_board_file(relative, pending=True):
                raise RuntimeError('unexpected provisioning file')
            copy_file(path, target / relative)
    for folder in sorted((entry for entry in target.rglob('*') if entry.is_dir()), reverse=True):
        LEGACY.sync_dir(folder)
    LEGACY.sync_dir(target)


def allowed_board_file(relative, pending=False):
    name = relative.as_posix()
    if name in ('board.db', 'web/web.db'):
        return True
    if len(relative.parts) == 2 and relative.parts[0] == 'board.db.migrations':
        return bool(PATTERN.fullmatch(relative.name))
    if pending and re.fullmatch(r'(?:board\.db|web/web\.db)-(?:wal|shm|journal)', name):
        return True
    return (len(relative.parts) == 3 and relative.parts[:2] in
            (('web', 'blobs'), ('web', 'staging')) and
            bool(LEGACY.BLOB.fullmatch(relative.name)))
