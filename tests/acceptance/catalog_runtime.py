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


def roundtrip(connection, method, path, body=None, headers=None, ledger=None,
              phase='readiness', operation=None):
    """Time HTTP through full body/error; always release client in-flight slot.

    Status may exist on an incomplete body; only fully read responses count as
    completed. JSON processing and accounting are outside the latency timer.
    """
    status, completed, error_type, elapsed = None, False, None, None
    if ledger is not None:
        ledger.begin(phase)
    started = time.perf_counter()
    try:
        connection.request(method, path, body, headers or {})
        response = connection.getresponse()
        status = response.status
        data = response.read()
        elapsed = (time.perf_counter() - started) * 1000
        completed = True
        if status != 200:
            raise HTTPFailure(operation, status)
    except BaseException as error:
        error_type = type(error).__name__
        raise
    finally:
        if elapsed is None:
            elapsed = (time.perf_counter() - started) * 1000
        if ledger is not None:
            ledger.finish(phase, operation, status, completed, elapsed, error_type)
    return data, elapsed


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
        data, elapsed = roundtrip(self.connection, 'GET' if read else 'POST', path,
                                  body, self.headers, self.ledger, self.phase, operation)
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
                roundtrip(health.connection, 'GET', path, ledger=ledger, operation=path)
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
