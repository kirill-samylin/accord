"""Equal catalog mix with separate setup warmup, HTTP timing and pacing context."""
import concurrent.futures
import threading
import time


def measure_people(people, keys, requests, waves=None, serial=None):
    barrier = threading.Barrier(len(people))
    if waves is not None:
        for index, person in enumerate(people):
            person.phase = 'warmup'
            person.request('get', {'task_key': keys[index % len(keys)]}, read=True)
            person.pacer = None
    if waves is not None and serial is not None:
        # Refill the wave budget after serial warmup, including size200.
        waves.pacer.previous = serial.previous
    from_date, to_date = '2000-01-01T00:00:00Z', '2100-01-01T00:00:00Z'

    def exercise(index):
        person = people[index]
        records, waits = [], []
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
                if waves is not None:
                    waits.append(waves.enter())
                name, payload, read = operations[number % len(operations)]
                _, latency_ms = person.request(name, payload, read)
                records.append((name, latency_ms))
                if waves is not None:
                    waits.append(waves.leave())
            return records, sum(waits)
        except BaseException:
            barrier.abort()
            if waves is not None:
                waves.abort()
            raise

    measured_started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(people)) as pool:
        measured = list(pool.map(exercise, range(len(people))))
    elapsed = time.perf_counter() - measured_started
    records = [record for participant, _ in measured for record in participant]
    pacing = waves.report([wait for _, wait in measured]) if waves else {"mode": "unpaced-legacy"}
    pacing["serial_setup_seed_warmup_wait_seconds"] = serial.wait_seconds if serial else 0
    return records, elapsed, pacing
