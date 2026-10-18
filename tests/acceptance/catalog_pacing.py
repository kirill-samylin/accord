"""Monotonic pacing, bounded wave synchronization and HTTP accounting.

A wave is a concurrency opportunity, not a guarantee of simultaneous OS/socket
execution. Waits are external to Participant's HTTP timer. No retries or
catch-up credit are accumulated; a broken worker aborts both rendezvous.
"""
import math
import threading
import time


class Ledger:
    """Count completed HTTP responses, including failures, by phase/operation."""
    def __init__(self):
        self.lock = threading.Lock()
        self.responses = {}
        self.attempts = 0

    def attempt(self):
        with self.lock:
            self.attempts += 1

    def record(self, phase, operation, status):
        with self.lock:
            key = (phase, operation, str(status))
            self.responses[key] = self.responses.get(key, 0) + 1

    def report(self):
        with self.lock:
            rows = [{'phase': phase, 'operation': operation, 'status': status, 'count': count}
                    for (phase, operation, status), count in sorted(self.responses.items())]
        return {'attempted_requests': self.attempts, 'completed_responses': sum(row['count'] for row in rows), 'by_operation': rows,
                'scope': 'All completed HTTP responses, including readiness; transport failures '
                         'have no HTTP status. Token CLI calls are not HTTP.'}


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
        if self.previous is not None:
            deadline = self.previous + self.interval
            while current < deadline:
                self.sleep(deadline - current)
                current = self.now()
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
