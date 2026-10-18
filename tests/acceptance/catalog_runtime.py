"""Owned-child HTTP and runtime observations for catalog load; no external URL input."""
import http.client
import json
import os
import pathlib
import selectors
import subprocess
import time
import urllib.parse


class HTTPFailure(RuntimeError):
    """Non-200 is terminal: retries would change the offered workload."""
    def __init__(self, operation, status):
        self.operation, self.status = operation, status
        super().__init__('Synthetic ' + operation + ' status=' + str(status))


def server_command(binary, database, persisted_defaults=False):
    command = [str(binary), 'serve', '--db', str(database), '--listen', '127.0.0.1:0']
    if not persisted_defaults:
        command += ['--session-ttl', '1h', '--task-ttl', '1h']
    return command


class Participant:
    def __init__(self, origin, token, ledger=None, pacer=None):
        authority = urllib.parse.urlsplit(origin)
        self.connection = http.client.HTTPConnection(authority.hostname, authority.port, timeout=30)
        self.headers = {'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json'}
        self.sequence = 0
        self.ledger, self.pacer, self.phase = ledger, pacer, "setup"

    def request(self, operation, payload, read=False):
        self.sequence += 1
        if read:
            query = urllib.parse.urlencode(payload)
            path, body = '/api/v1/' + operation + '?' + query, None
            if operation == 'get':
                path = '/api/v1/tasks/' + urllib.parse.quote(payload['task_key'], safe='')
        else:
            routes = {'describe': 'session', 'create': 'tasks', 'note': 'notes'}
            path = '/api/v1/' + routes.get(operation, operation)
            payload = dict(payload, request_id='load-' + str(self.sequence))
            body = json.dumps(payload).encode()
        if self.pacer is not None:
            self.pacer.wait()
        if self.ledger is not None:
            self.ledger.attempt()
        started = time.perf_counter()
        self.connection.request('GET' if read else 'POST', path, body, self.headers)
        response = self.connection.getresponse()
        data = response.read()
        elapsed = (time.perf_counter() - started) * 1000
        if self.ledger is not None:
            self.ledger.record(self.phase, operation, response.status)
        if response.status != 200:
            raise HTTPFailure(operation, response.status)
        return json.loads(data), elapsed

    def close(self):
        self.connection.close()


def rss_kib(pid):
    status = pathlib.Path('/proc') / str(pid) / 'status'
    if status.exists():
        for line in status.read_text().splitlines():
            if line.startswith('VmRSS:'):
                return int(line.split()[1])
    value = subprocess.check_output(['ps', '-o', 'rss=', '-p', str(pid)], text=True)
    return int(value.strip())


def start_server(binary, database, persisted_defaults=False, ledger=None):
    environment = {k: v for k, v in os.environ.items()
                   if k not in ['AGENT_BOARD_TOKEN', 'ACCORD_EXTERNAL_HOST']}
    started = time.perf_counter()
    command = server_command(binary, database, persisted_defaults)
    child = subprocess.Popen(command, stdout=subprocess.PIPE,
                             stderr=subprocess.DEVNULL, env=environment, text=True)
    selector = selectors.DefaultSelector()
    try:
        selector.register(child.stdout, selectors.EVENT_READ)
        if not selector.select(timeout=10):
            raise RuntimeError('Synthetic child startup deadline exceeded')
        line = child.stdout.readline().strip()
        prefix = 'agent-board listening on '
        if not line.startswith(prefix):
            raise RuntimeError('Synthetic child did not announce address')
        origin = line[len(prefix):]
        health = Participant(origin, 'synthetic-unused')
        try:
            for path in ['/healthz', '/readyz']:
                if ledger is not None:
                    ledger.attempt()
                health.connection.request('GET', path)
                response = health.connection.getresponse()
                response.read()
                if ledger is not None:
                    ledger.record('readiness', path, response.status)
                if response.status != 200:
                    raise RuntimeError('Synthetic child not ready')
        finally:
            health.close()
        return child, origin, (time.perf_counter() - started) * 1000
    except BaseException:
        stop_server(child)
        raise
    finally:
        selector.close()


def stop_server(child):
    child.terminate()
    try:
        child.wait(timeout=7)
    except subprocess.TimeoutExpired:
        child.kill()
        child.wait()
        raise RuntimeError('Owned child exceeded graceful shutdown deadline')
    finally:
        child.stdout.close()
