"""Create private authority with the actual prior executable, never live state."""
from contextlib import closing
import http.cookiejar
import json
import os
from pathlib import Path
import re
import select
import socket
import sqlite3
import subprocess
import time
import urllib.parse
import urllib.request
import uuid


def free_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    assert port not in (7892, 7893, 7894, 3100, 3101)
    return port


def command(binary, args, body=None):
    result = subprocess.run([str(binary), *map(str, args)], input=json.dumps(body) if body else None,
                            capture_output=True, text=True, timeout=30, env=clean_env())
    assert result.returncode == 0, 'synthetic executable command rejected; body suppressed'
    return result.stdout


class Process:
    def __init__(self, binary, state, port, origin, registry=True):
        self.args = [str(binary), 'serve', '--db', str(state / 'board.db'), '--listen', f'127.0.0.1:{port}',
                     '--web-origin', origin, '--web-data', str(state / 'web'), '--session-ttl', '30m', '--task-ttl', '30m']
        if registry:
            self.args += ['--projects-data', str(state / 'projects-data')]
        self.child = subprocess.Popen(self.args, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                                      env=clean_env(), text=True)
        self.url = f'http://127.0.0.1:{port}'
        try:
            assert select.select([self.child.stdout], [], [], 10)[0], 'owned startup output deadline'
            assert self.child.stdout.readline().startswith('agent-board listening on '), 'owned readiness line absent'
        except BaseException:
            self.child.terminate()
            self.child.wait(timeout=10)
            raise
        finally:
            self.child.stdout.close()
        for _ in range(100):
            assert self.child.poll() is None, 'owned server exited before readiness'
            try:
                with urllib.request.urlopen(self.url + '/readyz', timeout=1) as response:
                    if response.status == 200:
                        return
            except OSError:
                time.sleep(.03)
        self.stop()
        raise AssertionError('owned readiness timed out')

    def stop(self):
        if self.child.poll() is None:
            self.child.terminate()
            try:
                self.child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.child.kill()
                self.child.wait(timeout=5)
                raise AssertionError('owned process required forced shutdown')
        assert self.child.returncode == 0, 'owned server failed graceful shutdown'


def clean_env():
    return {k: v for k, v in os.environ.items() if not k.startswith(('ACCORD_', 'AGENT_BOARD_'))} | {'PYTHONDONTWRITEBYTECODE': '1'}


def admin(binary, state, op, extra=()):
    return command(binary, ['projects', op, '--projects-data', state / 'projects-data', '--db', state / 'board.db',
                            '--web-data', state / 'web', *extra])


def seed(binary, folder):
    folder.mkdir(mode=0o700)
    token = folder.parent / 'synthetic-owner.token'
    output = command(binary, ['token', 'create', '--db', folder / 'board.db', '--owner', 'Synthetic QA',
                              '--machine', 'synthetic-update', '--output', token])
    identity = re.search(r'Token (\S+) saved', output).group(1)
    port = free_port()
    origin = f'http://127.0.0.1:{port}'
    server = Process(binary, folder, port, origin, registry=False)
    server.stop()
    admin(binary, folder, 'bootstrap-owner', ['--identity', identity])
    server = Process(binary, folder, port, origin)
    try:
        jar = http.cookiejar.CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

        def web(route, fields=None):
            headers = {'Accept': 'application/json'}
            data = None
            if fields is not None:
                headers['Origin'] = origin
                data = urllib.parse.urlencode(fields).encode()
            with opener.open(urllib.request.Request(origin + route, data=data, headers=headers), timeout=10) as response:
                return json.load(response)

        csrf = web('/web/login')['page']['csrf']
        assert web('/web/login', {'csrf': csrf, 'token': token.read_text().strip()})['ok']
        catalog = web('/web/projects')['page']['registry']
        identifiers = [catalog['projects'][0]['id']]
        for number in range(2):
            result = web('/web/catalog/projects', {'csrf': catalog['csrf'], 'action_id': catalog['actions']['create'],
                         'name': 'Synthetic project ' + str(number), 'repository_url': 'https://example.com/qa.git', 'branch': 'main'})
            assert result['ok']
            catalog = web('/web/projects')['page']['registry']
        identifiers = [p['id'] for p in catalog['projects']]
        assert len(identifiers) == 3
        for project in identifiers:
            def call(op, body):
                return json.loads(command(binary, ['call', op, '--server', origin, '--token-file', token, '--project', project], body))
            session = call('describe', {'request_id': uuid.uuid4().hex, 'role': 'coordinator', 'machine': 'synthetic-update', 'run_id': uuid.uuid4().hex})['session_id']
            task = call('create', {'request_id': uuid.uuid4().hex, 'session_id': session, 'type': 'epic', 'title': 'Immutable synthetic history', 'description': 'Preserve me'})
            child = call('create', {'request_id': uuid.uuid4().hex, 'session_id': session, 'type': 'task', 'epic_key': task['key'], 'role': 'developer', 'title': 'Synthetic child task', 'description': 'Preserve child'})
            call('note', {'request_id': uuid.uuid4().hex, 'session_id': session, 'action': 'create', 'scope_type': 'epic',
                          'scope_id': task['key'], 'source_task_key': child['key'], 'text': 'Historical synthetic note'})
    finally:
        server.stop()
    return identifiers


def rebind(state):
    with closing(sqlite3.connect(state / 'projects-data/registry.sqlite')) as db, db:
        db.execute('UPDATE registry_meta SET default_db=?,default_web=?', (str(state / 'board.db'), str(state / 'web')))


def capture(state):
    """Independent SQL-row oracle; do not trust the operator's preservation flag."""
    registry = state / 'projects-data/registry.sqlite'
    paths = {'registry': registry, 'default/board': state / 'board.db', 'default/web': state / 'web/web.db'}
    with closing(sqlite3.connect(registry)) as db:
        default = db.execute('SELECT default_id FROM registry_meta WHERE id=1').fetchone()[0]
        for identifier, in db.execute("SELECT id FROM projects WHERE state IN ('active','archived')"):
            if identifier != default:
                for kind, relative in (('board', 'board.db'), ('web', 'web/web.db')):
                    paths[identifier + '/' + kind] = state / 'projects-data/projects' / identifier / relative
    result = {}
    for label, database in paths.items():
        with closing(sqlite3.connect(database.absolute().as_uri() + '?mode=ro&immutable=1', uri=True)) as db:
            tables = [row[0] for row in db.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")]
            result[label] = {table: sorted(json.dumps(tuple(row), default=lambda value: value.hex())
                             for row in db.execute('SELECT * FROM "' + table.replace('"', '""') + '"')) for table in tables}
    return result
