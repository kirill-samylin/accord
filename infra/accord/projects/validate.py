"""Verify the complete self-contained format2 snapshot before publication/restore."""
from contextlib import closing
import json
from pathlib import PurePosixPath
import sqlite3

from guard import MARKER
from layout import LEGACY, PROJECT_ID, allowed_board_file
from migration_files import validate_migrations


def database(path, version):
    LEGACY.private(path)
    with path.open('rb') as stream:
        header = stream.read(20)
    if header[:16] != b'SQLite format 3\0' or header[18:20] != b'\x01\x01':
        raise RuntimeError('snapshot database is not self-contained')
    with closing(sqlite3.connect(path.absolute().as_uri() + '?mode=ro&immutable=1', uri=True)) as connection:
        if (connection.execute('PRAGMA user_version').fetchone()[0] != version or
                connection.execute('PRAGMA integrity_check').fetchall() != [('ok',)]):
            raise RuntimeError('snapshot schema/integrity differs')


def board(path, schemas):
    if schemas.get('board_schema') not in (2, 3) or schemas.get('web_schema') not in (1, 2):
        raise RuntimeError('unknown project snapshot schema')
    database(path / 'board.db', schemas['board_schema'])
    if schemas['board_schema'] == 3:
        with closing(sqlite3.connect((path / 'board.db').absolute().as_uri() + '?mode=ro&immutable=1', uri=True)) as connection:
            for query in ('SELECT id,sequence FROM metadata LIMIT 0',
                          'SELECT key,data FROM tasks LIMIT 0',
                          'SELECT key,data FROM sessions LIMIT 0',
                          'SELECT role,generation FROM role_generations LIMIT 0',
                          'SELECT cursor,data FROM events LIMIT 0',
                          'SELECT key,hash,response FROM idempotency LIMIT 0',
                          'SELECT id,hash,owner,machine,expires,revoked FROM tokens LIMIT 0',
                          'SELECT key,data FROM notes LIMIT 0',
                          'SELECT id,sequence FROM note_metadata LIMIT 0',
                          'SELECT key,data FROM agent_profiles LIMIT 0',
                          'SELECT profile_id,version,data FROM agent_profile_revisions LIMIT 0'):
                connection.execute(query)
            for table in ('metadata', 'note_metadata'):
                if len(connection.execute('SELECT sequence FROM ' + table + ' WHERE id=1').fetchall()) != 1:
                    raise RuntimeError('snapshot board metadata missing')
    if (path / 'board.db.migrations').exists():
        validate_migrations(path / 'board.db.migrations', LEGACY)
    database(path / 'web/web.db', schemas['web_schema'])
    with closing(sqlite3.connect((path / 'web/web.db').absolute().as_uri() + '?mode=ro&immutable=1', uri=True)) as connection:
        for blob, size, digest, preview, preview_size in connection.execute(
                "SELECT blob,size,sha256,preview,preview_size FROM design_assets WHERE status='ready'"):
            if not LEGACY.BLOB.fullmatch(blob) or preview and not LEGACY.BLOB.fullmatch(preview):
                raise RuntimeError('unsafe design reference')
            source = path / 'web/blobs' / blob
            if LEGACY.private(source) != size or LEGACY.sha(source) != digest:
                raise RuntimeError('design content mismatch')
            if preview and LEGACY.private(path / 'web/blobs' / preview) != preview_size:
                raise RuntimeError('preview content mismatch')


def validate(path):
    LEGACY.private(path, True)
    LEGACY.private(path / 'manifest.json')
    manifest = json.loads((path / 'manifest.json').read_text())
    if manifest.get('format') != 2 or manifest.get('registry_schema') != 1:
        raise RuntimeError('all-project snapshot format required')
    default_id = manifest.get('default_id', '')
    if not PROJECT_ID.fullmatch(default_id):
        raise RuntimeError('invalid default project snapshot identity')
    projects = manifest.get('projects')
    if not isinstance(projects, list) or len(projects) > 31:
        raise RuntimeError('unexpected project snapshot inventory')
    identities = set()
    states = {}
    directories = {'default', 'default/web', 'default/web/blobs', 'default/web/staging',
                   'default/board.db.migrations', 'registry', 'projects'}
    for entry in projects:
        identifier = entry['id']
        if (not PROJECT_ID.fullmatch(identifier) or identifier == default_id or
                identifier in identities or entry['state'] not in ('active', 'archived', 'provisioning')):
            raise RuntimeError('invalid project snapshot member')
        identities.add(identifier)
        states[identifier] = entry['state']
        prefix = 'projects/' + identifier
        directories.update((prefix, prefix + '/web', prefix + '/web/blobs', prefix + '/web/staging'))
        directories.add(prefix + '/board.db.migrations')
    quarantined = manifest.get('quarantine', [])
    if not isinstance(quarantined, list) or len(quarantined) > 128 or len(set(quarantined)) != len(quarantined):
        raise RuntimeError('invalid quarantined inventory')
    if quarantined:
        directories.add('quarantine')
    for identifier in quarantined:
        if not PROJECT_ID.fullmatch(identifier) or identifier == default_id or identifier in identities:
            raise RuntimeError('invalid quarantined project identity')
        prefix = 'quarantine/' + identifier
        directories.update((prefix, prefix + '/web', prefix + '/web/blobs', prefix + '/web/staging'))
        directories.add(prefix + '/board.db.migrations')
    files = set()
    for entry in path.rglob('*'):
        relative = entry.relative_to(path)
        name = relative.as_posix()
        if entry.is_dir() and not entry.is_symlink():
            LEGACY.private(entry, True)
            if name not in directories:
                raise RuntimeError('unexpected snapshot directory')
            continue
        LEGACY.private(entry)
        if name == 'manifest.json':
            continue
        if name.startswith('default/'):
            allowed = allowed_board_file(PurePosixPath(*relative.parts[1:]))
        elif name in ('registry/registry.sqlite', 'registry/' + MARKER):
            allowed = True
        elif len(relative.parts) >= 3 and relative.parts[0] == 'projects' and relative.parts[1] in identities:
            allowed = allowed_board_file(PurePosixPath(*relative.parts[2:]),
                                         pending=states[relative.parts[1]] == 'provisioning')
        elif len(relative.parts) >= 3 and relative.parts[0] == 'quarantine' and relative.parts[1] in quarantined:
            allowed = allowed_board_file(PurePosixPath(*relative.parts[2:]), pending=True)
        else:
            allowed = False
        if not allowed:
            raise RuntimeError('unexpected snapshot file')
        files.add(name)
    if set(manifest['files']) != files:
        raise RuntimeError('snapshot manifest file set differs')
    for name, digest in manifest['files'].items():
        relative = PurePosixPath(name)
        if relative.is_absolute() or '..' in relative.parts or LEGACY.sha(path / name) != digest:
            raise RuntimeError('snapshot manifest path/checksum differs')
    database(path / 'registry/registry.sqlite', 1)
    for identifier in quarantined:
        LEGACY.private(path / 'quarantine' / identifier, True)
    LEGACY.private(path / ('registry/' + MARKER))
    board(path / 'default', manifest['default'])
    for entry in projects:
        if entry['state'] != 'provisioning':
            board(path / 'projects' / entry['id'], entry)
    with closing(sqlite3.connect((path / 'registry/registry.sqlite').absolute().as_uri() + '?mode=ro&immutable=1', uri=True)) as connection:
        metadata = connection.execute('SELECT default_id FROM registry_meta WHERE id=1').fetchall()
        actual = dict(connection.execute('SELECT id,state FROM projects'))
    expected = {entry['id']: entry['state'] for entry in projects}
    expected[default_id] = 'active'
    if metadata != [(default_id,)] or actual != expected:
        raise RuntimeError('snapshot registry/project inventory differs')
    return manifest
