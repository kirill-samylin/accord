"""Monotonic pacing, bounded wave synchronization and HTTP accounting.

A wave is a concurrency opportunity, not a guarantee of simultaneous OS/socket
execution. Waits are external to Participant's HTTP timer. No retries or
catch-up credit are accumulated; a broken worker aborts both rendezvous.
"""
import math
import threading
import time


class Ledger:
    """One locked ledger for HTTP attempts, completed bodies and raw timings.

    begin/finish bracket actual client roundtrips, not barrier participants.
    Snapshots are copied under the same lock; callers freeze after worker join.
    """
    def __init__(self):
        self.lock = threading.Lock()
        self.responses = {}
        self.attempts = 0
        self.active = self.peak = 0
        self.phases, self.timings = {}, []

    def begin(self, phase):
        with self.lock:
            row = self.phases.setdefault(phase, {'phase': phase, 'attempted_requests': 0,
                                                'completed_responses': 0,
                                                'active_in_flight': 0, 'peak_in_flight': 0})
            self.attempts += 1
            self.active += 1
            self.peak = max(self.peak, self.active)
            row['attempted_requests'] += 1
            row['active_in_flight'] += 1
            row['peak_in_flight'] = max(row['peak_in_flight'], row['active_in_flight'])

    def finish(self, phase, operation, status, completed, latency_ms, error_type):
        with self.lock:
            self.active -= 1
            row = self.phases[phase]
            row['active_in_flight'] -= 1
            if completed:
                self._record(phase, operation, status)
                row['completed_responses'] += 1
            self.timings.append({'phase': phase, 'operation': operation, 'status': status,
                                 'completed_response': completed, 'latency_ms': latency_ms,
                                 'successful_response': completed and status == 200,
                                 'error_type': error_type})

    def samples(self):
        with self.lock:
            return [dict(row) for row in self.timings]

    def _record(self, phase, operation, status):
        key = (phase, operation, str(status))
        self.responses[key] = self.responses.get(key, 0) + 1

    def attempt(self):
        with self.lock:
            self.attempts += 1

    def record(self, phase, operation, status):
        with self.lock:
            self._record(phase, operation, status)

    def report(self):
        with self.lock:
            rows = [{'phase': phase, 'operation': operation, 'status': status, 'count': count}
                    for (phase, operation, status), count in sorted(self.responses.items())]
            return {'attempted_requests': self.attempts,
                    'completed_responses': sum(row['count'] for row in rows), 'by_operation': rows,
                    'by_phase': [dict(self.phases[key]) for key in sorted(self.phases)],
                    'client_http_concurrency': {'peak_in_flight': self.peak,
                                                'active_in_flight': self.active,
                                                'scope': 'Client request through full body/error; '
                                                         'all phases, not server in-flight.'},
                    'scope': 'Completed HTTP bodies including readiness and non200; transport '
                             'failures are attempts without completed response. Token CLI excluded.'}


class SerialPacer:
    """One release per interval; late wakeups reset cadence, never build credit."""
    def __init__(self, rate, clock=time.monotonic, wait=time.sleep):
        if not math.isfinite(rate) or not 0 < rate <= 100:
            raise ValueError('offered rate must be finite in (0, 100]')
        self.interval, self.clock, self.sleep = 1 / rate, clock, wait
        self.previous, self.observed = None, None
        self.wait_seconds = 0.0

    def now(self):
        current = self.clock()
        if self.observed is not None and current < self.observed:
            raise RuntimeError('Monotonic clock moved backwards')
        self.observed = current
        return current

    def wait(self):
        started = self.now()
        current = started
        wall_started = time.monotonic()
        try:
            if self.previous is not None:
                deadline = self.previous + self.interval
                while current < deadline:
                    self.sleep(deadline - current)
                    current = self.now()
        except BaseException:
            # Aborted sleep has no release/clock sample, but its wait still happened.
            self.wait_seconds += time.monotonic() - wall_started
            raise
        self.previous = current
        elapsed = current - started
        self.wait_seconds += elapsed
        return elapsed


class Waves:
    """All sessions enter/leave each request wave; both barriers have deadlines.

    Interval is sessions/rate, anchored to the actual previous release. The
    interruptible pacing sleep wakes on abort before either barrier is aborted.
    The 200-session bound is a method burst bound, never a product session limit.
    """
    def __init__(self, sessions, rate, clock=time.monotonic, wait=None, timeout=None):
        if not 2 <= sessions <= 200:
            raise ValueError('paced method requires 2..200 sessions (burst bound)')
        self.cancelled = threading.Event()
        sleeper = self.pause if wait is None else wait
        self.pacer = SerialPacer(rate, clock, sleeper)
        self.pacer.interval = sessions / rate
        self.timeout = 35 + sessions / rate if timeout is None else timeout
        self.before = threading.Barrier(sessions, action=self.release, timeout=self.timeout)
        self.after = threading.Barrier(sessions, timeout=self.timeout)
        self.launches = []

    def pause(self, delay):
        if self.cancelled.wait(delay):
            raise RuntimeError("Wave cancelled")

    def release(self):
        self.pacer.wait()
        if self.cancelled.is_set():
            raise RuntimeError('Wave cancelled')
        self.launches.append(self.pacer.previous)

    def enter(self):
        started = time.monotonic()
        self.before.wait()
        return time.monotonic() - started

    def leave(self):
        started = time.monotonic()
        self.after.wait()
        return time.monotonic() - started

    def abort(self):
        self.cancelled.set()
        self.before.abort()
        self.after.abort()

    def report(self, worker_waits):
        return {'mode': 'barrier-waves', 'wave_count': len(self.launches),
                'wave_size': self.before.parties, 'minimum_interval_seconds': self.pacer.interval,
                'actual_release_offsets_seconds': [value - self.launches[0] for value in self.launches],
                'rate_wait_seconds': self.pacer.wait_seconds,
                'worker_synchronization_wait_seconds': sum(worker_waits),
                'scope': 'Worker waits include rate, peers and scheduling, summed across workers; '
                         'not wall time and never included in HTTP latency. Slow requests retained.'}
