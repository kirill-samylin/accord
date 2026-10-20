"""Synthetic private inspection/snapshot/restore. Only hashes and counts leave SQLite."""
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import shutil
import sqlite3
import sys

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO / 'infra/accord/projects'))
from snapshot import snapshot
from restore import restore
from guard import require_capable
from validate import validate


def digest(value):
    return hashlib.sha256(value).hexdigest()


def rows(database):
    result = {}
    for (name,) in database.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"):
        quoted = '"' + name.replace('"', '""') + '"'
        result[name] = sorted(digest(json.dumps(row, separators=(',', ':'), default=lambda v: {'bytes': v.hex()}).encode())
                              for row in database.execute('SELECT * FROM ' + quoted))
    return result


def projects(state):
    with closing(sqlite3.connect(state / 'projects-data/registry.sqlite')) as db:
        default = db.execute('SELECT default_id FROM registry_meta WHERE id=1').fetchone()[0]
        return [(identifier, state if identifier == default else state / 'projects-data/projects' / identifier)
                for identifier, lifecycle in db.execute('SELECT id,state FROM projects ORDER BY id') if lifecycle != 'provisioning']


def capture(state):
    result = {'databases': {}, 'blobs': {}, 'schemas': {}, 'profiles': {}}
    entries = [('registry', state / 'projects-data/registry.sqlite')]
    for identifier, folder in projects(state):
        entries.extend(((identifier + '/board', folder / 'board.db'), (identifier + '/web', folder / 'web/web.db')))
        for path in sorted((folder / 'web/blobs').iterdir()):
            result['blobs'][identifier + '/' + path.name] = digest(path.read_bytes())
    for name, path in entries:
        with closing(sqlite3.connect(path)) as db:
            result['schemas'][name] = db.execute('PRAGMA user_version').fetchone()[0]
            result['databases'][name] = rows(db)
            if name.endswith('/board') and result['schemas'][name] >= 3:
                result['profiles'][name] = {table: result['databases'][name][table]
                                           for table in ('agent_profiles', 'agent_profile_revisions')}
    return result


def immutable(state):
    result = {'boards': {}, 'blobs': capture_blobs(state)}
    with closing(sqlite3.connect(state / 'projects-data/registry.sqlite')) as db:
        result['repositories'] = sorted(digest(json.dumps(row).encode()) for row in db.execute('SELECT * FROM repositories'))
    for identifier, folder in projects(state):
        with closing(sqlite3.connect(folder / 'board.db')) as db:
            captured = rows(db)
            tasks = {}
            for key, raw in db.execute('SELECT key,data FROM tasks'):
                task = json.loads(raw)
                for name in ('lease', 'state', 'version', 'ready_at'):
                    task.pop(name, None)
                for attempt in task.get('attempts', []):
                    attempt.pop('ended_at', None)
                    attempt.pop('reason', None)
                tasks[key] = digest(json.dumps(task, sort_keys=True).encode())
            result['boards'][identifier] = {'task_history': tasks, 'events': captured['events'],
                                           'notes': captured['notes'], 'idempotency': captured['idempotency']}
    return result


def inspect(state):
    with closing(sqlite3.connect(state / 'projects-data/registry.sqlite')) as db:
        default = db.execute('SELECT default_id FROM registry_meta WHERE id=1').fetchone()[0]
        owner = db.execute("SELECT id FROM identities WHERE source='legacy' ORDER BY id LIMIT 1").fetchone()[0]
        credentials = db.execute('SELECT count(*) FROM credentials WHERE revoked=0').fetchone()[0]
        memberships = db.execute('SELECT count(*) FROM memberships WHERE active=1').fetchone()[0]
    result = {'default_id': default, 'owner_identity': owner, 'live_catalog_credentials': credentials,
              'active_memberships': memberships, 'boards': []}
    for identifier, folder in projects(state):
        with closing(sqlite3.connect(folder / 'board.db')) as db:
            token_count = db.execute('SELECT count(*) FROM tokens WHERE revoked=0').fetchone()[0]
            profile_count = db.execute('SELECT count(*) FROM agent_profiles').fetchone()[0] if db.execute('PRAGMA user_version').fetchone()[0] >= 3 else 0
        with closing(sqlite3.connect(folder / 'web/web.db')) as db:
            browsers = db.execute('SELECT count(*) FROM browser_sessions').fetchone()[0]
        result['boards'].append({'id': identifier, 'live_tokens': token_count, 'profiles': profile_count, 'browsers': browsers})
    return result


def main():
    mode, state = sys.argv[1], Path(sys.argv[2]).resolve()
    assert '/.local/task010-qa/run-' in str(state), 'only isolated synthetic run directories'
    opts = json.load(sys.stdin)
    if mode == 'capture':
        result = capture(state)
    elif mode == 'inspect':
        result = inspect(state)
    elif mode == 'immutable':
        result = immutable(state)
    elif mode == 'snapshot':
        destination = state.parent / opts['name']
        assert destination.parent == state.parent and not destination.exists()
        destination.mkdir(mode=0o700)
        saved = snapshot(state / 'board.db', state / 'web', state / 'projects-data', destination)
        manifest = validate(saved)
        result = {'snapshot': str(saved), 'manifest_sha256': digest((saved / 'manifest.json').read_bytes()),
                  'files': len(manifest['files']), 'board_schema': manifest['default']['board_schema']}
    elif mode == 'restore':
        destination = state.parent / opts['name']
        assert destination.parent == state.parent and not destination.exists()
        source = Path(opts['snapshot']).resolve()
        assert state.parent in source.parents
        restore(source, destination, Path(opts['binary']).resolve())
        result = {'state': str(destination), 'capture': capture(destination), 'inspect': inspect(destination), 'immutable': immutable(destination),
                  'boot_guard': (destination / 'RESTORE_REQUIRES_RESET').is_file(),
                  'reset_complete': (destination / 'RESTORE_AUTH_RESET_COMPLETE').is_file()}
    elif mode == 'guard':
        denials = {}
        for label in ('old', 'new'):
            try:
                require_capable(Path(opts[label]).resolve(), state / 'projects-data')
                denials[label] = 'allowed'
            except Exception:
                denials[label] = 'denied'
        result = denials
    elif mode == 'corrupt':
        destination = state.parent / opts['name']
        assert destination.parent == state.parent and not destination.exists()
        shutil.copytree(state, destination)
        for folder in [destination, *destination.rglob('*')]:
            folder.chmod(0o700 if folder.is_dir() else 0o600)
        with closing(sqlite3.connect(destination / 'projects-data/registry.sqlite')) as db, db:
            db.execute('UPDATE registry_meta SET default_db=?,default_web=? WHERE id=1', (str(destination / 'board.db'), str(destination / 'web')))
        with closing(sqlite3.connect(destination / 'board.db')) as db, db:
            db.execute('DROP TABLE agent_profile_revisions')
        result = {'state': str(destination), 'blobs': capture_blobs(destination)}
    elif mode == 'blobs':
        result = capture_blobs(state)
    elif mode == 'profile_tables':
        with closing(sqlite3.connect(state / 'board.db')) as db:
            result = [row[0] for row in db.execute("SELECT name FROM sqlite_schema WHERE name IN ('agent_profiles','agent_profile_revisions') ORDER BY name")]
    else:
        raise RuntimeError('unknown synthetic fixture mode')
    print(json.dumps(result, sort_keys=True))


def capture_blobs(state):
    return {str(path.relative_to(state)): digest(path.read_bytes()) for path in state.rglob('web/blobs/*') if path.is_file()}


if __name__ == '__main__':
    os.umask(0o077)
    main()
