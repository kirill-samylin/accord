"""Synthetic old-writer coordination plus byte-preservation history fixtures."""
from contextlib import closing
import hashlib
import json
from pathlib import Path
import sqlite3
import uuid

from fixture import Process, command, free_port


def boards(state):
    with closing(sqlite3.connect(state / 'projects-data/registry.sqlite')) as db:
        default = db.execute('SELECT default_id FROM registry_meta').fetchone()[0]
        return [(identifier, state if identifier == default else state / 'projects-data/projects' / identifier)
                for identifier, in db.execute('SELECT id FROM projects ORDER BY id')]


def enrich(binary, state):
    token = state.parent / 'synthetic-owner.token'
    port = free_port()
    origin = f'http://127.0.0.1:{port}'
    server = Process(binary, state, port, origin)
    try:
        for project, _ in boards(state):
            def call(op, body):
                return json.loads(command(binary, ['call', op, '--server', origin, '--token-file', token,
                                                   '--project', project], {'request_id': uuid.uuid4().hex, **body}))
            coordinator = call('describe', {'role': 'coordinator', 'machine': 'synthetic-update',
                                            'run_id': uuid.uuid4().hex})['session_id']
            profile = call('profile_create', {'session_id': coordinator, 'name': 'Pinned synthetic profile',
                           'description': 'Preserve immutable assignment', 'role': 'developer',
                           'instructions': 'No real execution', 'provider': 'custom', 'model': 'qa-no-worker'})
            epic = call('create', {'session_id': coordinator, 'type': 'epic', 'title': 'Pinned synthetic epic'})
            task = call('create', {'session_id': coordinator, 'type': 'task', 'epic_key': epic['key'],
                        'role': 'developer', 'title': 'Pinned synthetic task', 'description': 'Synthetic assignment preservation',
                        'profile_id': profile['id'], 'profile_version': 1})
            dependent = call('create', {'session_id': coordinator, 'type': 'task', 'epic_key': epic['key'],
                                       'role': 'developer', 'title': 'Dependent synthetic task', 'description': 'Synthetic dependency preservation'})
            call('link', {'session_id': coordinator, 'from': task['key'], 'to': dependent['key'], 'type': 'blocks'})
    finally:
        server.stop()
    for _, folder in boards(state):
        history = folder / 'board.db.migrations'
        history.mkdir(mode=0o700, exist_ok=True)
        # These are explicitly synthetic opaque history inputs, not evidence of
        # historical production migrations. Operator validators preserve bytes.
        for version in (1, 2, 3):
            temporary = history / ('fixture-' + str(version) + '.db')
            with closing(sqlite3.connect(temporary)) as db, db:
                db.execute('PRAGMA user_version=' + str(version))
                db.execute('CREATE TABLE synthetic_history(value TEXT)')
                db.execute('INSERT INTO synthetic_history VALUES(?)', ('preserve-schema-' + str(version),))
            digest = hashlib.sha256(temporary.read_bytes()).hexdigest()
            target = history / ('schema' + str(version) + '-' + digest + '.db')
            temporary.rename(target)
            target.chmod(0o600)
            checksum = target.with_name(target.name + '.sha256')
            checksum.write_text(digest + '  ' + target.name + '\n')
            checksum.chmod(0o600)
