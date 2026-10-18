"""HTTP overlap and retained failure evidence; stdlib fakes, no product process."""
import collections
import concurrent.futures
import json
import pathlib
import sys
import tempfile
import threading
import time
import unittest
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1] / 'acceptance'))
import catalog_load as load
from catalog_pacing import Ledger, Waves
from catalog_runtime import HTTPFailure, Participant
from catalog_workload import measure_people


class Connection:
    def __init__(self, owner, callback=None):
        self.owner, self.callback = owner, callback
        self.status, self.measured = 200, 0
        self.statuses = {}

    def request(self, *args):
        if self.owner.phase == 'measured':
            self.measured += 1

    def getresponse(self):
        self.status = self.statuses.get(self.measured, 200) if self.owner.phase == 'measured' else 200
        return self

    def read(self):
        if self.callback:
            self.callback(self)
        return b'{}'

    def close(self):
        pass


def participants(count, ledger, callbacks=()):
    people = []
    for index in range(count):
        person = Participant('http://127.0.0.1:1', 'synthetic', ledger)
        person.connection = Connection(person, callbacks[index] if callbacks else None)
        people.append(person)
    return people


class AccountingTests(unittest.TestCase):
    def fail_report(self, people_factory, waves=None, requests=5, legacy=False):
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory) / 'failure.json'
            args = load.parse_arguments(['--binary', '/synthetic', '--output', str(output),
                                         '--sessions', '2', '--offered-rps', '0' if legacy else '100'])

            def partial_run(binary, sessions, tasks, count, rate, defaults, ledger):
                people = people_factory(ledger)
                synchronization = None if legacy else waves or Waves(2, 100, timeout=.3)
                measure_people(people, ['T-1'], requests, synchronization)
                self.fail('workload must fail')

            with mock.patch.object(load, 'parse_arguments', return_value=args), \
                    mock.patch.object(load, 'run', side_effect=partial_run):
                with self.assertRaises(Exception):
                    load.main()
            return json.loads(output.read_text())

    def test_failure_after_successful_wave_preserves_late_peer_and_429(self):
        overlap = threading.Barrier(2)
        peer_finished = threading.Event()

        def failed(connection):
            if connection.measured == 2:
                overlap.wait(.5)
                time.sleep(.01)

        def slow(connection):
            if connection.measured == 2:
                overlap.wait(.5)
                time.sleep(.06)
                peer_finished.set()

        def factory(ledger):
            people = participants(2, ledger, [failed, slow])
            people[0].connection.statuses = {2: 429}
            return people

        report = self.fail_report(factory)
        self.assertTrue(peer_finished.is_set())
        self.assertEqual(report['experiment_status'], 'FAIL')
        self.assertNotIn('thresholds_observed', report)
        self.assertEqual(report['http_count']['completed_responses'], 6)
        measured_counts = [row for row in report['http_count']['by_operation'] if row['phase'] == 'measured']
        self.assertEqual(sum(row['count'] for row in measured_counts if row['status'] == '200'), 3)
        self.assertEqual(sum(row['count'] for row in measured_counts if row['status'] == '429'), 1)
        self.assertIn('elapsed_seconds', report)  # frozen055 loses this despite the six responses
        self.assertFalse(report['workload_complete'])
        self.assertEqual(report['measured_http_count'], 4)
        self.assertEqual(report['measured_attempt_count'], 4)
        self.assertEqual(report['measured_success_count'], 3)
        self.assertEqual(report['request_latency']['count'], 3)
        self.assertEqual(report['failed_request_latency']['count'], 1)
        self.assertGreaterEqual(report['request_latency']['max_ms'], 60)
        self.assertGreaterEqual(report['failed_request_latency']['max_ms'], 10)
        self.assertGreater(report['elapsed_seconds'], .06)
        self.assertEqual(report['pacing_context']['wave_count'], 2)
        self.assertEqual(report['http_count']['completed_responses'], 6)
        self.assertEqual(len(report['http_samples']), 6)
        self.assertTrue(any(row['status'] == 429 for row in report['http_samples']))
        self.assertTrue(any(row['error_type'] == 'HTTPFailure' for row in report['workload_errors']))
        self.assertEqual(report['client_http_concurrency']['active_in_flight'], 0)

    def test_legacy_failure_drains_in_progress_peer_and_retains_samples(self):
        overlap = threading.Barrier(2)

        def read(connection):
            overlap.wait(.5)
            if connection.measured == 2:
                time.sleep(.01 if connection.status == 429 else .04)

        def factory(ledger):
            people = participants(2, ledger, [read, read])
            people[0].connection.statuses = {2: 429}
            return people

        report = self.fail_report(factory, legacy=True)
        self.assertEqual(report['pacing_context']['mode'], 'unpaced-legacy')
        self.assertEqual(report['measured_http_count'], 4)
        self.assertEqual(report['measured_success_count'], 3)
        self.assertEqual(len(report['http_samples']), 6)
        self.assertGreaterEqual(report['request_latency']['max_ms'], 40)
        self.assertEqual(report['client_http_concurrency']['active_in_flight'], 0)
        self.assertFalse(report['workload_complete'])

    def test_failed_rate_wait_preserves_partial_pacing_without_new_http(self):
        waves = Waves(2, 100, timeout=.3)

        def failed_sleep(seconds):
            time.sleep(.01)
            raise RuntimeError('synthetic pacing failure')

        waves.pacer.sleep = failed_sleep
        report = self.fail_report(lambda ledger: participants(2, ledger), waves)
        self.assertEqual(report['measured_http_count'], 2)
        self.assertEqual(report['request_latency']['count'], 2)
        self.assertGreaterEqual(report['pacing_context']['rate_wait_seconds'], .01)
        self.assertGreaterEqual(report['elapsed_seconds'], .01)
        self.assertFalse(report['workload_complete'])
        self.assertNotIn('thresholds_observed', report)

    def test_transport_failure_retains_successes_without_fabricated_response(self):
        overlap = threading.Barrier(2)

        def callback(failure):
            def read(connection):
                if connection.measured == 2:
                    overlap.wait(.5)
                    time.sleep(.01)
                    if failure:
                        raise TimeoutError('synthetic transport')
            return read

        report = self.fail_report(lambda ledger: participants(2, ledger, [callback(True), callback(False)]))
        self.assertEqual(report['measured_attempt_count'], 4)
        self.assertEqual(report['measured_http_count'], 3)
        self.assertEqual(report['measured_success_count'], 3)
        self.assertEqual(report['failed_request_latency']['count'], 1)
        failure = [row for row in report['http_samples'] if row['error_type'] == 'TimeoutError']
        self.assertEqual(len(failure), 1)
        self.assertFalse(failure[0]['completed_response'])
        self.assertGreaterEqual(failure[0]['latency_ms'], 10)
        self.assertEqual(report['http_count']['attempted_requests'], 6)
        self.assertEqual(report['http_count']['completed_responses'], 5)
        self.assertEqual(report['client_http_concurrency']['active_in_flight'], 0)
        self.assertNotIn('thresholds_observed', report)

    def test_barrier_timeout_retains_successful_samples_duration_and_waits(self):
        def slow(connection):
            if connection.measured == 2:
                time.sleep(.06)

        report = self.fail_report(lambda ledger: participants(2, ledger, [slow, None]),
                                  Waves(2, 100, timeout=.02))
        self.assertEqual(report['measured_success_count'], 4)
        self.assertEqual(report['request_latency']['count'], 4)
        self.assertGreater(report['elapsed_seconds'], .06)
        self.assertGreater(report['pacing_context']['worker_synchronization_wait_seconds'], .02)
        self.assertFalse(report['workload_complete'])
        self.assertNotIn('thresholds_observed', report)

    def test_achieved_overlap_distinguishes_instant_calls_and_tracks_phases(self):
        ledger = Ledger()
        people = participants(2, ledger)
        for person in people:
            person.phase = 'setup'
            person.request('heartbeat', {})
        self.assertEqual(ledger.report()['client_http_concurrency']['peak_in_flight'], 1)
        gate = threading.Barrier(2)
        for person in people:
            person.phase = 'measured'
            person.connection.callback = lambda connection: gate.wait(.5)
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            futures = [pool.submit(person.request, 'heartbeat', {}) for person in people]
            for future in futures:
                future.result()
        report = ledger.report()
        self.assertEqual(report['client_http_concurrency']['peak_in_flight'], 2)
        self.assertEqual(report['client_http_concurrency']['active_in_flight'], 0)
        phases = {row['phase']: row for row in report['by_phase']}
        self.assertEqual(phases['setup']['peak_in_flight'], 1)
        self.assertEqual(phases['measured']['peak_in_flight'], 2)
        self.assertTrue(all(row['attempted_requests'] == row['completed_responses'] == 2
                            for row in phases.values()))

    def test_exception_cleanup_at_each_transport_boundary(self):
        for boundary in ('request', 'getresponse', 'read'):
            with self.subTest(boundary=boundary):
                ledger = Ledger()
                person = participants(1, ledger)[0]
                with mock.patch.object(person.connection, boundary, side_effect=OSError('synthetic')):
                    with self.assertRaises(OSError):
                        person.request('heartbeat', {})
                report = ledger.report()
                self.assertEqual(report['attempted_requests'], 1)
                self.assertEqual(report['completed_responses'], 0)
                self.assertEqual(report['client_http_concurrency']['active_in_flight'], 0)
                self.assertEqual(report['by_phase'][0]['active_in_flight'], 0)
                self.assertEqual(len(ledger.samples()), 1)

    def test_all_400_successful_http_samples_and_mix_preserved(self):
        ledger = Ledger()
        gate = threading.Barrier(20)

        def read(connection):
            if connection.owner.phase == 'measured':
                gate.wait(2)

        people = participants(20, ledger, [read] * 20)
        waves = Waves(20, 100, timeout=2)
        waves.pacer.sleep = lambda seconds: None
        waves.pacer.interval = 0  # concurrency fixture; cadence has dedicated units
        records, _, _ = measure_people(people, ['T-1'], 20, waves)
        samples = [row for row in ledger.samples() if row['phase'] == 'measured']
        self.assertEqual(len(records), 400)
        self.assertEqual(len(samples), 400)
        self.assertEqual(collections.Counter(row['operation'] for row in samples),
                         {name: 80 for name in ('get', 'tasks', 'events', 'stats', 'heartbeat')})
        self.assertTrue(all(row['successful_response'] for row in samples))
        self.assertEqual(ledger.report()['client_http_concurrency']['peak_in_flight'], 20)
        self.assertEqual(ledger.report()['client_http_concurrency']['active_in_flight'], 0)
        self.assertEqual(ledger.report()['completed_responses'], 420)

    def test_setup_failure_still_has_duration_pacing_and_raw_error(self):
        def factory(ledger):
            people = participants(2, ledger)
            people[0].connection.request = mock.Mock(side_effect=ConnectionError('synthetic'))
            return people

        report = self.fail_report(factory)
        self.assertEqual(report['measured_attempt_count'], 0)
        self.assertEqual(report['elapsed_seconds'], 0)
        self.assertGreater(report['experiment_elapsed_seconds'], 0)
        self.assertIn('pacing_context', report)
        self.assertEqual(len(report['http_samples']), 1)
        self.assertEqual(report['http_samples'][0]['phase'], 'warmup')
        self.assertFalse(report['workload_complete'])


if __name__ == '__main__':
    unittest.main()
