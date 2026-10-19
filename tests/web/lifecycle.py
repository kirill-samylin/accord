"""Offline snapshot/restore on synthetic fixture; never invoke systemctl."""
import importlib.util
import json
import os
import pathlib
import shutil
import sqlite3
import subprocess
import sys
import uuid

sys.dont_write_bytecode = True
os.umask(0o077)
state, root, binary, module = sys.argv[1:]
state, root = pathlib.Path(state), pathlib.Path(root)
spec = importlib.util.spec_from_file_location('accord_backup', module)
backup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(backup)
backups = root / ('backups-' + uuid.uuid4().hex)
backups.mkdir(mode=0o700)
snapshot = backup.snapshot(state, backups)
manifest = backup.validate_snapshot(snapshot)
assert manifest['restore_requires_auth_reset']
restored = root / ('restored-' + uuid.uuid4().hex)
shutil.copytree(snapshot, restored)
backup.validate_snapshot(restored)
backup.prepare_restored_auth(restored)
result = subprocess.run([binary, 'maintenance', 'reset', '--db', str(restored / 'board.db'), '--reason', 'restore'], capture_output=True, timeout=15)
assert result.returncode == 0, 'native recovery failed'
for name, version in [('board.db', 2), ('web/web.db', 1)]:
    with sqlite3.connect(restored / name) as db:
        assert db.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
        assert db.execute('PRAGMA user_version').fetchone()[0] == version
with sqlite3.connect(restored / 'web/web.db') as db:
    for table in ['web_actions', 'browser_sessions', 'preauth']:
        assert db.execute('SELECT count(*) FROM ' + table).fetchone()[0] == 0
print(json.dumps({'restored': str(restored), 'manifest_files': len(manifest['files'])}))
