"""Private synthetic fixture inspection; never print stored credentials or user data."""
from contextlib import closing
import hashlib
import hmac
import json
import os
from pathlib import Path
import sqlite3
import sys


def connection(path):
    return sqlite3.connect(path)


def digest(value):
    return hashlib.sha256(value).hexdigest()


def capture(state):
    result = {'board': {}, 'web': {}, 'blobs': {}}
    for name, path, tables in (
            ('board', state / 'board.db', ['tasks', 'sessions', 'events', 'idempotency', 'tokens']),
            ('web', state / 'web/web.db', ['browser_sessions', 'web_actions', 'design_assets'])):
        with closing(connection(path)) as database:
            result[name + '_schema'] = database.execute('PRAGMA user_version').fetchone()[0]
            for table in tables:
                rows = database.execute('SELECT * FROM ' + table).fetchall()
                result[name][table] = sorted(digest(json.dumps(row, separators=(',', ':')).encode()) for row in rows)
    for path in sorted((state / 'web/blobs').iterdir()):
        result['blobs'][path.name] = digest(path.read_bytes())
    return result


def inspect(state):
    root = state / 'projects-data'
    with closing(connection(root / 'registry.sqlite')) as database:
        default = database.execute('SELECT default_id,default_db,default_web FROM registry_meta WHERE id=1').fetchone()
        projects = database.execute('SELECT id,state FROM projects ORDER BY id').fetchall()
        counts = {table: database.execute('SELECT count(*) FROM ' + table).fetchone()[0]
                  for table in ('sessions', 'actions', 'preauth', 'session_epochs', 'rejected_provisions')}
        live_credentials = database.execute('SELECT count(*) FROM credentials WHERE revoked=0').fetchone()[0]
    boards = []
    for identifier, lifecycle in projects:
        if lifecycle == 'provisioning':
            continue
        folder = state if identifier == default[0] else root / 'projects' / identifier
        with closing(connection(folder / 'board.db')) as database:
            sessions = [json.loads(row[0]) for row in database.execute('SELECT data FROM sessions')]
            leases = sum(bool(json.loads(row[0]).get('lease')) for row in database.execute('SELECT data FROM tasks'))
            live_tokens = database.execute('SELECT count(*) FROM tokens WHERE revoked=0').fetchone()[0]
        with closing(connection(folder / 'web/web.db')) as database:
            web_counts = {table: database.execute('SELECT count(*) FROM ' + table).fetchone()[0]
                          for table in ('browser_sessions', 'web_actions', 'preauth', 'web_action_epochs')}
        boards.append({'id': identifier, 'live_tokens': live_tokens, 'leases': leases,
                       'session_expiries': [entry['expires_at'] for entry in sessions], 'web_counts': web_counts})
    return {'default_id': default[0], 'binding_matches': default[1:] == (str(state / 'board.db'), str(state / 'web')),
            'projects': projects, 'global_counts': counts, 'live_credentials': live_credentials, 'boards': boards,
            'quarantine': sorted(path.name for path in (root / 'quarantine').iterdir()) if (root / 'quarantine').exists() else []}


def pending(state, options):
    root = state / 'projects-data'
    with closing(connection(root / 'registry.sqlite')) as database, database:
        key = database.execute('SELECT signing_key FROM registry_meta WHERE id=1').fetchone()[0]
        value = b'describe\0catalog\0create\x000\0' + options['action'].encode()
        identifier = 'p_' + hmac.new(bytes.fromhex(key), value, hashlib.sha256).hexdigest()[:32]
        database.execute("INSERT INTO projects VALUES(?,?,'provisioning',1,?,1)",
                         (identifier, 'Synthetic unpublished', options['identity']))
    folder = root / 'projects' / identifier
    folder.mkdir(mode=0o700)
    path = folder / 'board.db-journal'
    path.write_bytes(b'synthetic incomplete provisioning journal')
    path.chmod(0o600)
    invalid = folder / 'board.db'
    invalid.write_bytes(b'synthetic incomplete unpublished SQLite header')
    invalid.chmod(0o600)
    if options.get('partial_quarantine'):
        quarantine = root / 'quarantine'
        quarantine.mkdir(mode=0o700, exist_ok=True)
        folder.rename(quarantine / identifier)
    return {'id': identifier}


def main():
    mode, state = sys.argv[1], Path(sys.argv[2]).resolve()
    options = json.load(sys.stdin)
    if mode == 'capture':
        result = capture(state)
    elif mode == 'inspect':
        result = inspect(state)
    elif mode == 'pending':
        result = pending(state, options)
    elif mode == 'catalog_owner':
        with closing(connection(state / 'projects-data/registry.sqlite')) as database:
            result = {'identity': database.execute("SELECT id FROM identities WHERE source='legacy' AND token_id=?", (options['token_id'],)).fetchone()[0]}
    elif mode == 'upload_inspect':
        folder = state / 'projects-data/projects' / options['project'] / 'web'
        with closing(connection(folder / 'web.db')) as database:
            rows = database.execute('SELECT status,reserved FROM design_assets WHERE title=?', (options['title'],)).fetchall()
        result = {'pending': sum(status == 'pending' for status, _ in rows),
                  'ready': sum(status == 'ready' for status, _ in rows),
                  'reserved': sum(size for _, size in rows), 'staging': len(list((folder / 'staging').iterdir()))}
    else:
        raise RuntimeError('unknown synthetic inspection')
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    os.umask(0o077)
    main()
