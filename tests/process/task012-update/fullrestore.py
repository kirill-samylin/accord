"""Actual offline all-project restore; independent recovery and immutable-data oracle."""
from contextlib import closing
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import sqlite3

from fixture import capture
from restore import restore
from validate import validate


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def files(root):
    return {str(path.relative_to(root)): sha(path) for path in root.rglob('*') if path.is_file()}


def immutable(state):
    values = capture(state)
    result = {}
    for name, tables in values.items():
        keep = ('projects', 'repositories', 'identities', 'audit') if name == 'registry' else (
            ('notes', 'idempotency', 'agent_profiles', 'agent_profile_revisions', 'coordination_metadata')
            if name.endswith('/board') else ())
        result[name] = {key: tables[key] for key in keep}
        if name.endswith('/board'):
            tasks = []
            for row in tables['tasks']:
                key, raw = json.loads(row)
                task = json.loads(raw)
                for field in ('lease', 'state', 'version', 'ready_at'):
                    task.pop(field, None)
                for attempt in task.get('attempts', []):
                    attempt.pop('ended_at', None)
                    attempt.pop('reason', None)
                tasks.append([key, task])
            result[name]['task_history'] = tasks
    return result


def check_restore(environment, snapshot):
    environment.stop('go')
    environment.stop('next')
    before = files(snapshot)
    live_before = files(environment.state)
    immutable_before = immutable(environment.state)
    marker = environment.state / 'projects-data/MULTIPROJECT_REQUIRES_CAPABILITY'
    marker_info = marker.stat()
    target = environment.root / 'restored-offline'
    binary = environment.go / 'releases' / ('c' * 40) / 'agent-board'
    restore(snapshot, target, binary)
    assert (target / 'RESTORE_REQUIRES_RESET').is_file()
    assert (target / 'RESTORE_AUTH_RESET_COMPLETE').is_file()
    assert immutable(target) == immutable_before, 'immutable business history changed during recovery'
    old_rows, new_rows = capture(environment.state), capture(target)
    assert len(new_rows) == 7
    for name, tables in old_rows.items():
        if name.endswith('/board'):
            assert set(tables['events']) <= set(new_rows[name]['events'])
            added = set(new_rows[name]['events']) - set(tables['events'])
            assert added
            for row in added:
                event = json.loads(json.loads(row)[1])
                assert event['type'] in ('tokens_revoked_all', 'lease_recovered', 'attempt_recovered', 'database_recovered')
                assert event['reason'] == 'restore'
    now = datetime.now(timezone.utc)
    with closing(sqlite3.connect(target / 'projects-data/registry.sqlite')) as db:
        assert db.execute('SELECT COUNT(*) FROM credentials WHERE revoked=0').fetchone()[0] == 0
        for table in ('sessions', 'actions', 'preauth', 'session_epochs'):
            assert db.execute('SELECT COUNT(*) FROM ' + table).fetchone()[0] == 0
        default, board, web = db.execute('SELECT default_id,default_db,default_web FROM registry_meta').fetchone()
        assert board == str(target / 'board.db') and web == str(target / 'web')
        entries = list(db.execute('SELECT id,state FROM projects'))
        memberships = list(db.execute('SELECT project_id,identity_id,access_epoch FROM memberships ORDER BY 1,2'))
    with closing(sqlite3.connect(environment.state / 'projects-data/registry.sqlite')) as db:
        old = list(db.execute('SELECT project_id,identity_id,access_epoch FROM memberships ORDER BY 1,2'))
    assert memberships == [(p, i, e + 1) for p, i, e in old]
    for identifier, _ in entries:
        folder = target if identifier == default else target / 'projects-data/projects' / identifier
        with closing(sqlite3.connect(folder / 'board.db')) as db:
            assert db.execute('PRAGMA user_version').fetchone()[0] == 4
            assert db.execute('SELECT COUNT(*) FROM tokens WHERE revoked=0').fetchone()[0] == 0
            for raw, in db.execute('SELECT data FROM sessions'):
                assert datetime.fromisoformat(json.loads(raw)['expires_at'].replace('Z', '+00:00')) <= now
            for raw, in db.execute('SELECT data FROM tasks'):
                assert not json.loads(raw).get('lease')
        with closing(sqlite3.connect(folder / 'web/web.db')) as db:
            for table in ('browser_sessions', 'web_actions', 'preauth', 'web_action_epochs'):
                assert db.execute('SELECT COUNT(*) FROM ' + table).fetchone()[0] == 0
    for path in target.rglob('*.migrations/*'):
        original = environment.state / path.relative_to(target)
        assert path.read_bytes() == original.read_bytes()
    assert files(snapshot) == before and files(environment.state) == live_before
    assert (marker.stat().st_uid, marker.stat().st_mode) == (marker_info.st_uid, marker_info.st_mode)
    assert validate(snapshot)['default']['board_schema'] == 4
    return {'databases': 7, 'credentials_revoked': True, 'sessions_cleared_or_expired': True,
            'membership_epochs_incremented': True, 'immutable_history_preserved': True,
            'source_snapshot_and_live_bytes_unchanged': True, 'boot_hold_retained': True}
