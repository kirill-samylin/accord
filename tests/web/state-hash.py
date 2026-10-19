"""Hash raw durable answer/event rows and schema, never expose their bodies."""
import hashlib
import json
import pathlib
import sqlite3
import sys

state = pathlib.Path(sys.argv[1])
with sqlite3.connect((state / 'board.db').as_uri() + '?mode=ro&immutable=1', uri=True) as db:
    version = db.execute('PRAGMA user_version').fetchone()[0]
    assert version == 2
    task = db.execute('SELECT data FROM tasks WHERE key=?', (sys.argv[2],)).fetchone()[0]
    events = db.execute("SELECT cursor,data FROM events WHERE json_extract(data,'$.type')='human_answered' AND json_extract(data,'$.task_key')=? ORDER BY cursor", (sys.argv[2],)).fetchall()
    assert len(events) == 1
    assert json.loads(task)['evidence']['answer'].startswith('Synthetic final answer')
    print(hashlib.sha256(json.dumps([version, task, events]).encode()).hexdigest())
