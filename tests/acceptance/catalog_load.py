#!/usr/bin/env python3
"""Isolated real-process load experiment; never connects to an existing service.

Only synthetic state is created in a private temporary directory. Defaults model
20 simultaneous sessions / 2000 tasks, not 200000 annual events. Measured mix is
get, tasks(limit=20), events(limit=20), stats(owner), heartbeat, repeated equally.
HTTP latency includes loopback request/socket/response-body overhead; excludes
JSON encode/decode, seed and pacing waits.
Results describe the observed machine; they cannot certify a different VPS.
"""
import argparse
import hashlib
import http.client
import json
import math
import os
import pathlib
import platform
import statistics
import subprocess
import tempfile
import threading
import time
import urllib.parse

from catalog_cgroup import cgroup_limits
from catalog_pacing import Ledger, SerialPacer, Waves
from catalog_runtime import Participant, rss_kib, start_server, stop_server
from catalog_workload import measure_people


def percentile(values, fraction):
    return sorted(values)[max(0, math.ceil(len(values) * fraction) - 1)]


def summary(values):
    return {'count': len(values), 'p50_ms': statistics.median(values),
            'p95_ms': percentile(values, .95), 'p99_ms': percentile(values, .99),
            'max_ms': max(values)}


def run(binary, sessions, tasks, requests, offered_rps=0, persisted_defaults=False, ledger=None):
    ledger = Ledger() if ledger is None else ledger
    serial = SerialPacer(offered_rps) if offered_rps else None
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
            child, origin, startup = start_server(binary, database, persisted_defaults, ledger)

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
                person = Participant(origin, secret, ledger, serial)
                people.append(person)
                view, _ = person.request('describe', {'role': 'coordinator' if number == 0 else 'developer',
                                                    'run_id': 'synthetic-' + str(number),
                                                    'machine': 'synthetic-' + str(number)})
                person.headers['X-Session'] = view['session_id']
            coordinator = people[0]
            epic, _ = coordinator.request('create', {'type': 'epic', 'title': 'Synthetic load',
                                                    'description': 'No real user data'})
            coordinator.phase = 'seed'
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
            child, restarted_origin, populated_startup = start_server(binary, database, persisted_defaults, ledger)
            for person in people:
                authority = urllib.parse.urlsplit(restarted_origin)
                person.connection = http.client.HTTPConnection(authority.hostname, authority.port, timeout=30)
            stop_sampling.clear()
            sampler = threading.Thread(target=sample)
            sampler.start()
            waves = Waves(sessions, offered_rps) if offered_rps else None
            records, elapsed, pacing = measure_people(people, keys, requests, waves, serial)
            latency = summary([value for _, value in records])
            per_operation = {name: summary([value for operation, value in records if operation == name])
                             for name in sorted({operation for operation, _ in records})}
            if not rss:
                raise RuntimeError('RSS measurement unavailable; result would be incomplete')
            return {'method_version': 'catalog-load-055-v1', 'experiment_status': 'COMPLETED',
                    'offered_rps': offered_rps, 'persisted_defaults': persisted_defaults,
                    'http_count': ledger.report(), 'pacing_context': pacing,
                    'platform': platform.platform(), 'machine': platform.machine(),
                    'logical_cpus': os.cpu_count(), 'cgroup_limits': cgroup_limits(child.pid),
                    'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
                    'sessions': sessions, 'tasks': tasks, 'requests_per_session': requests,
                    'seed_seconds': seed_seconds, 'startup_empty_ms': startup,
                    'startup_populated_ms': populated_startup, 'sampled_peak_rss_mib': max(rss) / 1024,
                    'rss_sampling_seconds': .05, 'elapsed_seconds': elapsed,
                    'actual_measured_rps': len(records) / elapsed,
                    'measured_http_count': len(records),
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


def parse_arguments(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=pathlib.Path, required=True)
    parser.add_argument('--output', type=pathlib.Path, required=True)
    parser.add_argument('--sessions', type=int, default=20)
    parser.add_argument('--tasks', type=int, default=2000)
    parser.add_argument('--requests-per-session', type=int, default=20)
    parser.add_argument('--offered-rps', type=float, default=0,
                        help='0=legacy unpaced; finite 1..100=all-session paced waves')
    parser.add_argument('--persisted-defaults', action='store_true',
                        help='omit serve TTL flags; use token-bootstrap persisted defaults')
    args = parser.parse_args(argv)
    if args.sessions < 2 or args.tasks < 1 or args.requests_per_session < 5:
        parser.error('sessions>=2, tasks>=1, requests-per-session>=5 required')
    if not math.isfinite(args.offered_rps) or not (args.offered_rps == 0 or 1 <= args.offered_rps <= 100):
        parser.error('offered-rps must be 0 (legacy) or finite 1..100')
    if args.offered_rps and args.sessions > 200:
        parser.error('paced method bounds burst to 200 sessions; not a product session limit')
    return args


def main():
    args = parse_arguments()
    os.umask(0o077)
    ledger = Ledger()
    # Reserve output before starting child: never overwrite an earlier experiment.
    with args.output.open('x') as stream:
        try:
            report = run(args.binary.resolve(), args.sessions, args.tasks, args.requests_per_session,
                         args.offered_rps, args.persisted_defaults, ledger)
        except Exception as error:
            report = {'method_version': 'catalog-load-055-v1', 'experiment_status': 'FAIL',
                      'offered_rps': args.offered_rps, 'persisted_defaults': args.persisted_defaults,
                      'http_count': ledger.report(), 'error_type': type(error).__name__}
            json.dump(report, stream, indent=2)
            stream.write('\n')
            raise
        json.dump(report, stream, indent=2)
        stream.write('\n')
    print('Synthetic load completed; report saved. Target-environment acceptance is separate.')


if __name__ == '__main__':
    main()
