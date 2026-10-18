"""Equal catalog mix; drain every future before freezing partial observations."""
import concurrent.futures
import threading
import time


def measure_people(people, keys, requests, waves=None, serial=None):
    barrier = threading.Barrier(len(people))
    cancelled = threading.Event()
    records = [[] for _ in people]
    waits = [0.0 for _ in people]
    errors = []
    measured_started = None
    from_date, to_date = '2000-01-01T00:00:00Z', '2100-01-01T00:00:00Z'

    def abort():
        cancelled.set()
        barrier.abort()
        if waves is not None:
            waves.abort()

    def rendezvous(index, action):
        started = time.monotonic()
        try:
            action()
        finally:
            # Includes failed/aborted waits, without entering HTTP latency.
            waits[index] += time.monotonic() - started

    def exercise(index):
        person = people[index]
        operations = [('get', {'task_key': keys[index % len(keys)]}, True),
                      ('tasks', {'limit': 20}, True),
                      ('events', {'since_cursor': 0, 'limit': 20, 'task_key': keys[index % len(keys)]}, True),
                      ('stats', {'from': from_date, 'to': to_date, 'group_by': 'owner'}, True),
                      ('heartbeat', {}, False)]
        try:
            if waves is None:
                person.phase = 'warmup'
                person.request('get', {'task_key': keys[index % len(keys)]}, read=True)
                barrier.wait(timeout=30)
            person.phase = 'measured'
            for number in range(requests):
                if cancelled.is_set():
                    break
                if waves is not None:
                    rendezvous(index, waves.enter)
                if cancelled.is_set():
                    break
                name, payload, read = operations[number % len(operations)]
                _, latency_ms = person.request(name, payload, read)
                records[index].append((name, latency_ms))
                if waves is not None:
                    rendezvous(index, waves.leave)
        except BaseException:
            abort()
            raise

    def context():
        elapsed = 0 if measured_started is None else time.perf_counter() - measured_started
        pacing = waves.report(waits) if waves else {'mode': 'unpaced-legacy'}
        pacing['serial_setup_seed_warmup_wait_seconds'] = serial.wait_seconds if serial else 0
        return {'elapsed_seconds': elapsed, 'measurement_started': measured_started is not None,
                'pacing_context': pacing,
                'workload_errors': [{'worker': index, 'error_type': type(error).__name__}
                                    for index, error in errors],
                'workload_complete': not errors and sum(map(len, records)) == len(people) * requests}

    try:
        if waves is not None:
            for index, person in enumerate(people):
                person.phase = 'warmup'
                person.request('get', {'task_key': keys[index % len(keys)]}, read=True)
                person.pacer = None
            if serial is not None:
                # Refill the wave budget after serial warmup, including size200.
                waves.pacer.previous = serial.previous
        measured_started = time.perf_counter()
        with concurrent.futures.ThreadPoolExecutor(max_workers=len(people)) as pool:
            try:
                futures = {pool.submit(exercise, index): index for index in range(len(people))}
                for future in concurrent.futures.as_completed(futures):
                    try:
                        future.result()
                    except BaseException as error:
                        errors.append((futures[future], error))
            except BaseException:
                abort()
                raise
        # All running calls (including late successful peers) have finished here.
        if errors:
            # Keep the causal HTTP/transport failure ahead of induced broken barriers.
            errors.sort(key=lambda row: (isinstance(row[1], threading.BrokenBarrierError), row[0]))
            raise errors[0][1]
        observed = context()
        return [row for participant in records for row in participant], observed['elapsed_seconds'], observed['pacing_context']
    except BaseException as error:
        abort()
        if not errors:
            errors.append((None, error))
        error.workload_context = context()
        raise
