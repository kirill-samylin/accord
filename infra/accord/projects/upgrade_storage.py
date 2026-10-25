"""Offline additive migration with exact logical preservation of all old tables."""
from contextlib import closing
import hashlib
import json

from layout import REGISTRY_SCHEMA, inventory, readonly
from host_ops import PROJECTS, STATE, host
from migration_files import REGISTRY_MIGRATIONS
from validate import board, registry


def quote(name):
    return '"' + name.replace('"', '""') + '"'


def columns(path):
    with closing(readonly(path)) as db:
        tables = [row[0] for row in db.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")]
        return {name: [row[1] for row in db.execute('PRAGMA table_info(' + quote(name) + ')')]
                for name in tables}


def facts(path, projection=None):
    """Per-table row digests; a projection restricts each table to its old columns.

    Additive DDL may append defaulted columns (registry2 identities.person_id);
    the rows seen through the previous column set must still be identical.
    """
    with closing(readonly(path)) as db:
        selected = projection if projection is not None else columns(path)
        result = {}
        for name, names in selected.items():
            select = ','.join(quote(column) for column in names) or '*'
            rows = sorted(json.dumps(row, ensure_ascii=True, separators=(',', ':'),
                                     default=lambda value: {'bytes': value.hex()})
                          for row in db.execute('SELECT ' + select + ' FROM ' + quote(name)))
            result[name] = hashlib.sha256(('\n'.join(rows)).encode()).hexdigest()
        return result


def sets(snapshot, manifest):
    yield snapshot / 'registry/registry.sqlite', PROJECTS / 'registry.sqlite'
    yield snapshot / 'default/board.db', STATE / 'board.db'
    yield snapshot / 'default/web/web.db', STATE / 'web/web.db'
    for item in manifest['projects']:
        if item['state'] != 'provisioning':
            for relative in ('board.db', 'web/web.db'):
                yield (snapshot / 'projects' / item['id'] / relative,
                       PROJECTS / 'projects' / item['id'] / relative)


def verify_registry_migration(snapshot, manifest):
    """Registry1 must have become registry2 behind a byte-proven pre-DDL copy."""
    registry(PROJECTS / 'registry.sqlite', REGISTRY_SCHEMA, live=True)
    if manifest['registry_schema'] >= REGISTRY_SCHEMA:
        return
    old = facts(snapshot / 'registry/registry.sqlite')
    copies = sorted((PROJECTS / REGISTRY_MIGRATIONS).glob('registry1-*.db'))
    if not any(facts(copy) == old for copy in copies):
        raise RuntimeError('registry pre-migration copy missing or differs from the snapshot')


def migrate_storage(binary, snapshot, manifest):
    catalog = inventory(PROJECTS, STATE / 'board.db', STATE / 'web')
    paths = [(STATE / 'board.db', STATE / 'web')]
    paths += [(item['folder'] / 'board.db', item['folder'] / 'web')
              for item in catalog['projects'] if item['state'] != 'provisioning']
    result = json.loads(host.run('runuser', '-u', 'accord', '--', str(binary),
                                'projects', 'migrate-storage', '--projects-data', str(PROJECTS),
                                '--db', str(STATE / 'board.db'), '--web-data', str(STATE / 'web')))
    if result.get('storage_migrated') is not True or result.get('boards') != len(paths):
        raise RuntimeError('offline existing-storage migration was not confirmed')
    # The migrated registry reads back through the same strict inventory,
    # including the new registry1 copy directory and its checksums.
    inventory(PROJECTS, STATE / 'board.db', STATE / 'web')
    verify_registry_migration(snapshot, manifest)
    for path, web in paths:
        with closing(readonly(web / 'web.db')) as db:
            version = db.execute('PRAGMA user_version').fetchone()[0]
        board(path.parent, {'board_schema': 6, 'web_schema': version}, live=True)
    for before, after in sets(snapshot, manifest):
        old, current = facts(before), facts(after, columns(before))
        if any(current.get(table) != digest for table, digest in old.items()):
            raise RuntimeError('additive storage migration changed existing logical rows')
    # Hashes include old immutable design bytes and historical migration copies.
    for relative, digest in manifest['files'].items():
        if '/web/blobs/' not in relative and '/web/staging/' not in relative and '.migrations/' not in relative:
            continue
        if relative.startswith('default/'):
            target = STATE / relative.removeprefix('default/')
        elif relative.startswith('projects/'):
            target = PROJECTS / relative
        elif relative.startswith('registry/'):
            target = PROJECTS / relative.removeprefix('registry/')
        else:
            continue
        if host.sha(target) != digest:
            raise RuntimeError('historical private content changed during migration')
    return {'databases_preserved': sum(1 for _ in sets(snapshot, manifest)),
            'published_boards_schema': 6, 'registry_schema': REGISTRY_SCHEMA,
            'logical_rows_preserved': True}
