#!/usr/bin/env python3
"""Isolated real-process load experiment; never connects to an existing service.

Only synthetic state is created in a private temporary directory. Defaults model
20 simultaneous sessions / 2000 tasks, not 200000 annual events. Measured mix is
get, tasks(limit=20), events(limit=20), stats(owner), heartbeat, repeated equally.
HTTP latency includes loopback socket/JSON/client overhead, excludes seed time.
Results describe the observed machine; they cannot certify a different VPS.
"""
import argparse
import concurrent.futures
import hashlib
import http.client
import json
import math
import os
import pathlib
import platform
import selectors
import statistics
import subprocess
import tempfile
import threading
import time
import urllib.parse

from catalog_cgroup import cgroup_limits


class Participant:
    def __init__(self, origin, token):
        authority = urllib.parse.urlsplit(origin)
        self.connection = http.client.HTTPConnection(authority.hostname, authority.port, timeout=30)
        self.headers = {'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json'}
        self.sequence = 0

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
        started = time.perf_counter()
        self.connection.request('GET' if read else 'POST', path, body, self.headers)
        response = self.connection.getresponse()
        data = response.read()
        elapsed = (time.perf_counter() - started) * 1000
        if response.status != 200:
            raise RuntimeError('Synthetic ' + operation + ' status=' + str(response.status))
        return json.loads(data), elapsed

    def close(self):
        self.connection.close()


def percentile(values, fraction):
    return sorted(values)[max(0, math.ceil(len(values) * fraction) - 1)]


def summary(values):
    return {'count': len(values), 'p50_ms': statistics.median(values),
            'p95_ms': percentile(values, .95), 'p99_ms': percentile(values, .99),
            'max_ms': max(values)}


def rss_kib(pid):
    status = pathlib.Path('/proc') / str(pid) / 'status'
    if status.exists():
        for line in status.read_text().splitlines():
            if line.startswith('VmRSS:'):
                return int(line.split()[1])
    value = subprocess.check_output(['ps', '-o', 'rss=', '-p', str(pid)], text=True)
    return int(value.strip())


def start_server(binary, database):
    environment = {k: v for k, v in os.environ.items()
                   if k not in ['AGENT_BOARD_TOKEN', 'ACCORD_EXTERNAL_HOST']}
    started = time.perf_counter()
    child = subprocess.Popen([str(binary), 'serve', '--db', str(database),
                              '--listen', '127.0.0.1:0', '--session-ttl', '1h',
                              '--task-ttl', '1h'], stdout=subprocess.PIPE,
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
                health.connection.request('GET', path)
                response = health.connection.getresponse()
                response.read()
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


def run(binary, sessions, tasks, requests):
    with tempfile.TemporaryDirectory(prefix='accord-synthetic-load-') as temporary:
        directory = pathlib.Path(temporary)
        directory.chmod(0o700)
        database = directory / 'board.db'
        people, rss, child = [], [], None
        stop_sampling = threading.Event()
        sampler = None
        try:
            # Tokens bootstrap storage before the measured restart, as on deployment.
            for number in range(sessions):
                token = directory / ('participant-' + str(number) + '.token')
                environment = {k: v for k, v in os.environ.items() if k != 'AGENT_BOARD_TOKEN'}
                subprocess.run([str(binary), 'token', 'create', '--db', str(database),
                                '--owner', 'synthetic-' + str(number), '--machine',
                                'synthetic-' + str(number), '--output', str(token)],
                               check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                               env=environment, timeout=30)
            child, origin, startup = start_server(binary, database)

            def sample():
                while not stop_sampling.is_set():
                    try:
                        rss.append(rss_kib(child.pid))
                    except (OSError, ValueError, subprocess.SubprocessError):
                        pass
                    stop_sampling.wait(.05)

            sampler = threading.Thread(target=sample)
            sampler.start()
            for number in range(sessions):
                secret = (directory / ('participant-' + str(number) + '.token')).read_text().strip()
                person = Participant(origin, secret)
                people.append(person)
                view, _ = person.request('describe', {'role': 'coordinator' if number == 0 else 'developer',
                                                    'run_id': 'synthetic-' + str(number),
                                                    'machine': 'synthetic-' + str(number)})
                person.headers['X-Session'] = view['session_id']
            coordinator = people[0]
            epic, _ = coordinator.request('create', {'type': 'epic', 'title': 'Synthetic load',
                                                    'description': 'No real user data'})
            seed_started = time.perf_counter()
            keys = []
            for number in range(tasks):
                task, _ = coordinator.request('create', {'type': 'task', 'epic_key': epic['key'],
                                                        'role': 'developer', 'title': 'Synthetic ' + str(number),
                                                        'description': 'Isolated load fixture'})
                keys.append(task['key'])
            seed_seconds = time.perf_counter() - seed_started
            # Include actual startup with the populated database; original clients reconnect.
            stop_sampling.set()
            sampler.join()
            for person in people:
                person.close()
            stop_server(child)
            child = None
            child, restarted_origin, populated_startup = start_server(binary, database)
            for person in people:
                authority = urllib.parse.urlsplit(restarted_origin)
                person.connection = http.client.HTTPConnection(authority.hostname, authority.port, timeout=30)
            stop_sampling.clear()
            sampler = threading.Thread(target=sample)
            sampler.start()
            barrier = threading.Barrier(sessions)
            from_date, to_date = '2000-01-01T00:00:00Z', '2100-01-01T00:00:00Z'

            def exercise(index):
                person = people[index]
                person.request('get', {'task_key': keys[index % len(keys)]}, read=True)
                barrier.wait(timeout=30)
                records = []
                operations = [('get', {'task_key': keys[index % len(keys)]}, True),
                              ('tasks', {'limit': 20}, True),
                              ('events', {'since_cursor': 0, 'limit': 20, 'task_key': keys[index % len(keys)]}, True),
                              ('stats', {'from': from_date, 'to': to_date, 'group_by': 'owner'}, True),
                              ('heartbeat', {}, False)]
                for number in range(requests):
                    name, payload, read = operations[number % len(operations)]
                    _, elapsed = person.request(name, payload, read)
                    records.append((name, elapsed))
                return records

            measured_started = time.perf_counter()
            with concurrent.futures.ThreadPoolExecutor(max_workers=sessions) as pool:
                measured = list(pool.map(exercise, range(sessions)))
            elapsed = time.perf_counter() - measured_started
            records = [record for participant in measured for record in participant]
            latency = summary([value for _, value in records])
            per_operation = {name: summary([value for operation, value in records if operation == name])
                             for name in sorted({operation for operation, _ in records})}
            if not rss:
                raise RuntimeError('RSS measurement unavailable; result would be incomplete')
            return {'platform': platform.platform(), 'machine': platform.machine(),
                    'logical_cpus': os.cpu_count(), 'cgroup_limits': cgroup_limits(child.pid),
                    'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
                    'sessions': sessions, 'tasks': tasks, 'requests_per_session': requests,
                    'seed_seconds': seed_seconds, 'startup_empty_ms': startup,
                    'startup_populated_ms': populated_startup, 'sampled_peak_rss_mib': max(rss) / 1024,
                    'rss_sampling_seconds': .05, 'elapsed_seconds': elapsed,
                    'request_latency': latency, 'per_operation': per_operation,
                    'thresholds_observed': {'p95_le_50ms': latency['p95_ms'] <= 50,
                                            'rss_le_128mib': max(rss) <= 128 * 1024,
                                            'populated_start_le_1s': populated_startup <= 1000},
                    'scope': 'Real isolated child; loopback HTTP; no TLS; synthetic workload; '
                             'sampled server RSS only; client excluded; no annual-event-volume proof; '
                             'no certification of another host or CPU/memory limits.'}
        finally:
            stop_sampling.set()
            if sampler is not None:
                sampler.join()
            for person in people:
                person.close()
            if child is not None:
                stop_server(child)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=pathlib.Path, required=True)
    parser.add_argument('--output', type=pathlib.Path, required=True)
    parser.add_argument('--sessions', type=int, default=20)
    parser.add_argument('--tasks', type=int, default=2000)
    parser.add_argument('--requests-per-session', type=int, default=20)
    args = parser.parse_args()
    if args.sessions < 2 or args.tasks < 1 or args.requests_per_session < 5:
        parser.error('sessions>=2, tasks>=1, requests-per-session>=5 required')
    os.umask(0o077)
    report = run(args.binary.resolve(), args.sessions, args.tasks, args.requests_per_session)
    with args.output.open('x') as stream:
        json.dump(report, stream, indent=2)
        stream.write('\n')
    print('Synthetic load completed; report saved. Target-environment acceptance is separate.')


if __name__ == '__main__':
    main()
