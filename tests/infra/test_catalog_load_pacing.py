"""Behavioral method tests; synthetic clocks/clients, no product process or VPS."""
import collections
import concurrent.futures
import contextlib
import io
import pathlib
import json
import tempfile
import sys
import threading
import time
import unittest
from unittest import mock

ACCEPTANCE = pathlib.Path(__file__).resolve().parents[1] / 'acceptance'
sys.path.insert(0, str(ACCEPTANCE))
import catalog_load as load
from catalog_pacing import Ledger, SerialPacer, Waves
from catalog_runtime import HTTPFailure, Participant, server_command
from catalog_workload import measure_people


class Clock:
    def __init__(self):
        self.value = 0.0
        self.lock = threading.Lock()

    def now(self):
        with self.lock:
            return self.value

    def advance(self, duration):
        with self.lock:
            self.value += duration


class FakePerson:
    def __init__(self, gate=None, failure=False, latency=7):
        self.phase, self.pacer = 'setup', None
        self.gate, self.failure, self.latency = gate, failure, latency
        self.operations = []

    def request(self, operation, payload, read=False):
        self.operations.append((self.phase, operation))
        if self.phase == 'measured' and self.gate is not None:
            self.gate.wait(timeout=2)
        if self.failure and self.phase == 'measured':
            raise HTTPFailure(operation, 429)
        return {}, self.latency


class PacingTests(unittest.TestCase):
    def parse(self, *extra):
        return load.parse_arguments(['--binary', '/synthetic', '--output', '/synthetic.json', *extra])

    def test_legacy_defaults_and_explicit_zero(self):
        for extra in ((), ('--offered-rps', '0')):
            args = self.parse(*extra)
            self.assertEqual((args.sessions, args.tasks, args.requests_per_session), (20, 2000, 20))
            self.assertEqual(args.offered_rps, 0)
            self.assertFalse(args.persisted_defaults)
        self.assertEqual(self.parse('--sessions', '201').sessions, 201)

    def test_paced_arguments_finite_bounded_and_method_only(self):
        with contextlib.redirect_stderr(io.StringIO()):
            for value in ('nan', 'inf', '-1', '0.5', '100.1'):
                with self.subTest(value=value), self.assertRaises(SystemExit):
                    self.parse('--offered-rps', value)
            with self.assertRaises(SystemExit):
                self.parse('--offered-rps', '100', '--sessions', '201')
        for sessions in (2, 20, 200):
            self.assertEqual(self.parse('--offered-rps', '100', '--sessions', str(sessions)).sessions, sessions)
        self.assertEqual(self.parse('--offered-rps', '1').offered_rps, 1)

    def test_persisted_defaults_only_omits_explicit_ttls(self):
        legacy = server_command('binary', 'db')
        persisted = server_command('binary', 'db', True)
        self.assertEqual(legacy, persisted + ['--session-ttl', '1h', '--task-ttl', '1h'])
        self.assertTrue(self.parse('--persisted-defaults').persisted_defaults)

    def test_serial_rate_no_catchup_after_delay(self):
        clock = Clock()
        pacer = SerialPacer(10, clock.now, clock.advance)
        releases = []
        for delay in (0, 0, 5, 0):
            clock.advance(delay)
            pacer.wait()
            releases.append(clock.now())
        for actual, expected in zip(releases, [0, .1, 5.1, 5.2]):
            self.assertAlmostEqual(actual, expected)
        self.assertAlmostEqual(pacer.wait_seconds, .2)

    def test_serial_early_wakeup_cannot_offer_early(self):
        clock = Clock()
        calls = []

        def early_sleep(duration):
            calls.append(duration)
            clock.advance(duration if len(calls) > 1 else duration / 2)

        pacer = SerialPacer(10, clock.now, early_sleep)
        pacer.wait()
        pacer.wait()
        self.assertEqual(clock.now(), .1)
        self.assertEqual(len(calls), 2)

    def test_clock_rollback_fails_closed(self):
        clock = Clock()
        pacer = SerialPacer(100, clock.now, clock.advance)
        pacer.wait()
        clock.advance(-1)
        with self.assertRaisesRegex(RuntimeError, 'backwards'):
            pacer.wait()

    def test_twenty_concurrent_every_wave_preserves_400_and_slow_samples(self):
        clock = Clock()
        waves = Waves(20, 100, clock.now, clock.advance, timeout=3)
        gate = threading.Barrier(20)
        people = [FakePerson(gate, latency=9000 if index == 0 else 7) for index in range(20)]
        records, elapsed, context = measure_people(people, ['T-1'], 20, waves)
        self.assertEqual(len(records), 400)
        self.assertEqual(collections.Counter(name for name, _ in records),
                         {name: 80 for name in ('get', 'tasks', 'events', 'stats', 'heartbeat')})
        self.assertEqual(sum(value == 9000 for _, value in records), 20)
        self.assertEqual(context['wave_count'], 20)
        self.assertEqual(context['wave_size'], 20)
        for difference in (right - left for left, right in zip(waves.launches, waves.launches[1:])):
            self.assertAlmostEqual(difference, .2)
        self.assertTrue(all(len(person.operations) == 21 for person in people))
        self.assertGreater(elapsed, 0)

    def test_slow_wave_resets_cadence_and_finish_blocks_fast_worker(self):
        clock = Clock()
        waves = Waves(2, 100, clock.now, clock.advance, timeout=2)
        fast_waiting = threading.Event()
        allow_slow = threading.Event()
        reached_next = threading.Event()

        def worker(index):
            for number in range(3):
                waves.enter()
                if number == 0:
                    if index == 0:
                        fast_waiting.set()
                    else:
                        self.assertTrue(allow_slow.wait(1))
                        clock.advance(5)
                else:
                    reached_next.set()
                waves.leave()

        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            futures = [pool.submit(worker, index) for index in range(2)]
            self.assertTrue(fast_waiting.wait(1))
            self.assertFalse(reached_next.is_set())
            allow_slow.set()
            for future in futures:
                future.result(timeout=2)
        self.assertEqual(waves.launches[:2], [0, 5])
        self.assertAlmostEqual(waves.launches[2], 5.02)

    def test_failed_worker_aborts_other_workers_without_retry(self):
        people = [FakePerson(failure=True), FakePerson()]
        waves = Waves(2, 100, timeout=.5)
        with self.assertRaises((HTTPFailure, threading.BrokenBarrierError)):
            measure_people(people, ['T-1'], 20, waves)
        self.assertTrue(waves.cancelled.is_set())
        self.assertTrue(all(len(person.operations) <= 2 for person in people))

    def test_barrier_deadlines_break_missing_worker(self):
        for side in ('enter', 'leave'):
            with self.subTest(side=side):
                waves = Waves(2, 100, timeout=.03)
                with self.assertRaises(threading.BrokenBarrierError):
                    getattr(waves, side)()
                waves.abort()

    def test_abort_interrupts_long_rate_sleep(self):
        waves = Waves(2, 1, timeout=3)
        sleeping = threading.Event()
        original = waves.pacer.sleep

        def signal_sleep(delay):
            sleeping.set()
            original(delay)

        waves.pacer.sleep = signal_sleep
        waves.pacer.previous = time.monotonic()
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            futures = [pool.submit(waves.enter) for _ in range(2)]
            self.assertTrue(sleeping.wait(1))
            waves.abort()
            for future in futures:
                with self.assertRaises((RuntimeError, threading.BrokenBarrierError)):
                    future.result(timeout=.5)

    def test_legacy_has_no_wave_waits_and_retains_mix(self):
        people = [FakePerson(), FakePerson()]
        records, _, context = measure_people(people, ['T-1'], 10)
        self.assertEqual(len(records), 20)
        self.assertEqual(context['mode'], 'unpaced-legacy')
        self.assertEqual(context['serial_setup_seed_warmup_wait_seconds'], 0)
        self.assertTrue(all(person.operations[0] == ('warmup', 'get') for person in people))

    def test_warmup_to_wave_transition_refills_entire_burst(self):
        clock = Clock()
        serial = SerialPacer(100, clock.now, clock.advance)
        waves = Waves(2, 100, clock.now, clock.advance, timeout=1)
        people = [FakePerson(), FakePerson()]
        for person in people:
            original = person.request

            def request(operation, payload, read=False, person=person, original=original):
                if person.phase == 'warmup':
                    serial.wait()
                return original(operation, payload, read)

            person.request = request
        measure_people(people, ['T-1'], 5, waves, serial)
        self.assertAlmostEqual(waves.launches[0], .03)
        self.assertAlmostEqual(waves.launches[1], .05)
        self.assertAlmostEqual(serial.wait_seconds, .01)
        for sessions in (1, 201):
            with self.assertRaises(ValueError):
                Waves(sessions, 100)

    def test_transport_failure_remains_an_attempt_and_has_no_retry(self):
        ledger = Ledger()
        person = Participant('http://127.0.0.1:1', 'synthetic', ledger)
        self.addCleanup(person.close)
        person.connection = mock.Mock()
        person.connection.request.side_effect = TimeoutError('synthetic timeout')
        with self.assertRaises(TimeoutError):
            person.request('heartbeat', {})
        self.assertEqual(ledger.report()['attempted_requests'], 1)
        self.assertEqual(ledger.report()['completed_responses'], 0)
        person.connection.request.assert_called_once()

    def test_legacy_warmup_failure_wakes_other_worker(self):
        people = [FakePerson(), FakePerson()]
        people[0].request = mock.Mock(side_effect=HTTPFailure('get', 429))
        started = time.monotonic()
        with self.assertRaises((HTTPFailure, threading.BrokenBarrierError)):
            measure_people(people, ['T-1'], 20)
        self.assertLess(time.monotonic() - started, 1)

    def test_failure_report_keeps_429_and_never_reports_threshold_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory) / 'failure.json'
            args = self.parse('--offered-rps', '100', '--output', str(output))

            def reject(binary, sessions, tasks, requests, rate, defaults, ledger):
                ledger.attempt()
                ledger.record('measured', 'get', 429)
                raise HTTPFailure('get', 429)

            with mock.patch.object(load, 'parse_arguments', return_value=args), \
                    mock.patch.object(load, 'run', side_effect=reject):
                with self.assertRaises(HTTPFailure):
                    load.main()
            report = json.loads(output.read_text())
            self.assertEqual(report['experiment_status'], 'FAIL')
            self.assertNotIn('thresholds_observed', report)
            self.assertEqual(report['http_count']['by_operation'][0]['status'], '429')

    def test_http_latency_excludes_pacer_and_429_is_counted_once(self):
        ledger = Ledger()
        person = Participant('http://127.0.0.1:1', 'synthetic', ledger, mock.Mock())
        self.addCleanup(person.close)
        person.connection = mock.Mock()
        response = person.connection.getresponse.return_value
        response.read.return_value = b'{}'
        response.status = 200
        with mock.patch('catalog_runtime.time.perf_counter', side_effect=[10, 10.007]):
            _, elapsed = person.request('heartbeat', {})
        self.assertAlmostEqual(elapsed, 7)
        self.assertEqual(person.pacer.wait.call_count, 1)
        response.status = 429
        with self.assertRaises(HTTPFailure):
            person.request('heartbeat', {})
        self.assertEqual(person.connection.request.call_count, 2)
        self.assertEqual(ledger.report()['attempted_requests'], 2)
        self.assertEqual(ledger.report()['completed_responses'], 2)
        self.assertEqual([row['count'] for row in ledger.report()['by_operation']], [1, 1])


if __name__ == '__main__':
    unittest.main()
