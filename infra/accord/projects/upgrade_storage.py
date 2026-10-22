"""Offline additive migration with exact logical preservation of all old tables."""
from contextlib import closing
import hashlib
import json
from pathlib import Path
import sqlite3

from layout import inventory, readonly
from host_ops import PROJECTS, STATE, host
from validate import board


def facts(path):
    with closing(readonly(path)) as db:
        tables = [row[0] for row in db.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")]
        result = {}
        for name in tables:
            quoted = '"' + name.replace('"', '""') + '"'
            rows = sorted(json.dumps(row, ensure_ascii=True, separators=(',', ':'),
                                     default=lambda value: {'bytes': value.hex()})
                          for row in db.execute('SELECT * FROM ' + quoted))
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
    for path, web in paths:
        with closing(readonly(web / 'web.db')) as db:
            version = db.execute('PRAGMA user_version').fetchone()[0]
        board(path.parent, {'board_schema': 5, 'web_schema': version}, live=True)
    for before, after in sets(snapshot, manifest):
        old, current = facts(before), facts(after)
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
        else:
            continue
        if host.sha(target) != digest:
            raise RuntimeError('historical private content changed during migration')
    return {'databases_preserved': sum(1 for _ in sets(snapshot, manifest)),
            'published_boards_schema': 5, 'logical_rows_preserved': True}
