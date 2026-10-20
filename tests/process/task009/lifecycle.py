"""Run reviewed all-project snapshot/restore against an offline synthetic fixture."""
from contextlib import closing
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import sys

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO / 'infra/accord/projects'))
from snapshot import snapshot
from validate import validate
spec = importlib.util.spec_from_file_location('task009_offline_restore', REPO / 'infra/accord/projects/restore.py')
restoration = importlib.util.module_from_spec(spec)
spec.loader.exec_module(restoration)


def main():
    state, root, binary = map(lambda value: Path(value).resolve(), sys.argv[1:4])
    assert str(root).startswith(str(REPO / '.local/task009-qa') + os.sep)
    copies = root / 'snapshots'
    copies.mkdir(mode=0o700)
    result = snapshot(state / 'board.db', state / 'web', state / 'projects-data', copies)
    manifest = validate(result)
    before = hashlib.sha256((result / 'manifest.json').read_bytes()).hexdigest()
    assert len(manifest['projects']) >= 3
    assert any(entry['state'] == 'provisioning' for entry in manifest['projects'])
    restored = root / 'restored'
    restoration.restore(result, restored, binary)
    assert (restored / 'RESTORE_REQUIRES_RESET').is_file()
    assert (restored / 'RESTORE_AUTH_RESET_COMPLETE').is_file()
    assert before == hashlib.sha256((result / 'manifest.json').read_bytes()).hexdigest()
    validate(result)
    # Independent inspection uses the actual new registry schema and every
    # published board. Quarantined bytes are private recovery evidence only.
    with closing(sqlite3.connect(restored / 'projects-data/registry.sqlite')) as database:
        assert database.execute('SELECT COUNT(*) FROM credentials WHERE revoked=0').fetchone()[0] == 0
        for table in ('sessions', 'actions', 'preauth', 'session_epochs'):
            assert database.execute('SELECT COUNT(*) FROM ' + table).fetchone()[0] == 0
        default, db, web = database.execute('SELECT default_id,default_db,default_web FROM registry_meta').fetchone()
        assert db == str(restored / 'board.db') and web == str(restored / 'web')
        projects = database.execute('SELECT id,state FROM projects').fetchall()
        assert all(state != 'provisioning' for _, state in projects)
    now = datetime.now(timezone.utc)
    for identifier, state in projects:
        folder = restored if identifier == default else restored / 'projects-data/projects' / identifier
        with closing(sqlite3.connect(folder / 'board.db')) as database:
            assert database.execute('SELECT COUNT(*) FROM tokens WHERE revoked=0').fetchone()[0] == 0
            for raw, in database.execute('SELECT data FROM sessions'):
                assert datetime.fromisoformat(json.loads(raw)['expires_at'].replace('Z', '+00:00')) <= now
            for raw, in database.execute('SELECT data FROM tasks'):
                assert not json.loads(raw).get('lease')
        with closing(sqlite3.connect(folder / 'web/web.db')) as database:
            for table in ('browser_sessions', 'web_actions', 'preauth', 'web_action_epochs'):
                assert database.execute('SELECT COUNT(*) FROM ' + table).fetchone()[0] == 0
    quarantine = restored / 'projects-data/quarantine'
    assert quarantine.is_dir()
    retained = sorted(path.name for path in quarantine.iterdir())
    assert retained
    print(json.dumps({'restored': str(restored), 'projects': len(projects),
                      'quarantined': len(retained), 'snapshot_manifest_sha256': before}))


if __name__ == '__main__':
    os.umask(0o077)
    main()
