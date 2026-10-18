"""Private evidence and bounded child-process support for backup_process.py."""
import contextlib
from dataclasses import dataclass
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import signal
import sqlite3
import subprocess
import time


DRIVER = '''import importlib.util, pathlib, sqlite3, subprocess, sys
spec = importlib.util.spec_from_file_location("checked_backup", sys.argv[1])
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)
module.STATE, module.BACKUPS = map(pathlib.Path, sys.argv[2:4])
module.BINARY = sys.argv[4]
try:
    module.run(len(sys.argv) > 5 and sys.argv[5] == "--plan")
except (OSError, RuntimeError, sqlite3.Error, subprocess.SubprocessError) as error:
    reason = str(error) if isinstance(error, module.BackupError) else type(error).__name__
    print("Accord backup failed:", reason, file=sys.stderr)
    sys.exit(1)
'''


def sha(path):
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def require(condition, reason):
    """Only static reasons enter evidence; never stringify database exceptions."""
    if not condition:
        raise AssertionError(reason)


def private_write(path, data):
    with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'wb') as stream:
        stream.write(data)


def tree(directory):
    """Compare names, bytes and mtime without following fixture symlinks."""
    result = {}
    for path in directory.iterdir():
        info = path.lstat()
        value = os.readlink(path) if path.is_symlink() else sha(path)
        result[path.name] = (info.st_mode, info.st_size, info.st_mtime_ns, value)
    return result


def database(path):
    """Closed copies only. Full rows stay in memory and never enter JSON/logs."""
    uri = path.resolve().as_uri() + '?mode=ro&immutable=1'
    with contextlib.closing(sqlite3.connect(uri, uri=True)) as db:
        require(db.execute('PRAGMA integrity_check').fetchall() == [('ok',)], 'sqlite_integrity')
        version = db.execute('PRAGMA user_version').fetchone()[0]
        schema = db.execute("SELECT type,name,sql FROM sqlite_master ORDER BY type,name").fetchall()
        tables = [row[0] for row in db.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")]
        rows = {name: sorted(db.execute('SELECT * FROM "' + name.replace('"', '""') + '"').fetchall(), key=repr) for name in tables}
    return version, schema, rows


def counts(image):
    return {name: len(rows) for name, rows in image[2].items()}


@dataclass(frozen=True)
class KnownSnapshot:
    """Declared fixture identity and content origin; never inferred from plan output."""
    name: str
    created: dt.datetime
    mtime_ns: int
    size: int
    digest: str
    origin: str

    @property
    def order(self):
        return self.created, self.mtime_ns, self.name


def known_snapshot(path, origin):
    created = dt.datetime.strptime(path.name.split('-')[1], '%Y%m%dT%H%M%SZ').replace(tzinfo=dt.timezone.utc)
    return KnownSnapshot(path.name, created, path.stat().st_mtime_ns,
                         path.stat().st_size + path.with_name(path.name + '.sha256').stat().st_size,
                         sha(path), origin)


def expected_retained(records, now, budget=5 * 1024 ** 3, estimate=0):
    """Contract oracle: enumerate bucket members, choose maxima, then fit logical budget.

    No production module/import/plan output is used. Free-space pressure is excluded
    separately by the small process fixture; this oracle models the logical budget.
    """
    ordered = sorted(records, key=lambda item: item.order, reverse=True)
    protected = {item.name for item in ordered[:2]}
    selected = set(protected)
    current_hour = now.replace(minute=0, second=0, microsecond=0)
    buckets = [current_hour - dt.timedelta(hours=offset) for offset in range(24)]
    for hour in buckets:
        members = [item for item in records if hour <= item.created < hour + dt.timedelta(hours=1)
                   and item.created <= now]
        if members:
            selected.add(max(members, key=lambda item: item.order).name)
    for offset in range(7):
        date = now.date() - dt.timedelta(days=offset)
        members = [item for item in records if item.created.date() == date and item.created <= now]
        if members:
            selected.add(max(members, key=lambda item: item.order).name)
    for item in reversed(ordered):
        if sum(record.size for record in records if record.name in selected) + estimate <= budget:
            return selected
        if item.name in selected and item.name not in protected:
            selected.remove(item.name)
    require(sum(item.size for item in records if item.name in selected) + estimate <= budget,
            'oracle_budget_preserving_two')
    return selected


def expected_at_interval(records, start, end, budget, estimate):
    """Exact possibilities when an unmodified child crosses UTC clock boundaries."""
    require(start <= end, 'fixture_clock_monotonic')
    instants = {start, end}
    boundary = start.replace(minute=0, second=0, microsecond=0) + dt.timedelta(hours=1)
    while boundary <= end:
        instants.add(boundary)
        boundary += dt.timedelta(hours=1)
    instants.update(item.created for item in records if start <= item.created <= end)
    return {frozenset(expected_retained(records, instant, budget, estimate)) for instant in instants}


class Evidence:
    """Persist partial results after every child/case, without raw child output."""

    def __init__(self, output, timeout, report):
        self.output, self.timeout, self.report = output, timeout, report
        private_write(output, b'{}\n')
        self.save()

    def save(self):
        self.report['counts'] = {status: sum(case['status'] == status for case in self.report['cases'])
                                 for status in ('PASS', 'FAIL', 'NOT-RUN')}
        with self.output.open('w', encoding='utf-8') as stream:
            json.dump(self.report, stream, indent=2, sort_keys=True)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())

    def command(self, label, argv, expected=0, hidden=False):
        started = time.monotonic()
        item = {'operation': label, 'status': 'RUNNING'}
        self.report['processes'].append(item)
        self.save()
        # No inherited credentials, stdin, shell, server or network are needed.
        env = {'PATH': os.defpath, 'LANG': 'C', 'PYTHONDONTWRITEBYTECODE': '1'}
        pipe = subprocess.DEVNULL if hidden else subprocess.PIPE
        with subprocess.Popen([str(arg) for arg in argv], stdin=subprocess.DEVNULL,
                              stdout=pipe, stderr=pipe, env=env, start_new_session=True) as child:
            item['pid'] = child.pid
            self.save()
            try:
                out, err = child.communicate(timeout=self.timeout)
            except BaseException:
                # This fresh process group includes the wrapper's Go child only.
                os.killpg(child.pid, signal.SIGTERM)
                try:
                    child.communicate(timeout=3)
                except subprocess.TimeoutExpired:
                    os.killpg(child.pid, signal.SIGKILL)
                    child.communicate()
                item.update(status='INTERRUPTED', exit_code=child.returncode,
                            seconds=round(time.monotonic() - started, 6))
                self.save()
                raise
        item.update(status='EXITED', exit_code=child.returncode,
                    seconds=round(time.monotonic() - started, 6),
                    stdout_bytes=len(out or b''), stderr_bytes=len(err or b''), hidden=hidden)
        self.save()
        require(child.returncode == expected, 'unexpected_exit:' + label)
        return out or b'', err or b''

    def case(self, case_id, action):
        item = {'id': case_id, 'status': 'RUNNING'}
        self.report['cases'].append(item)
        self.save()
        started = time.monotonic()
        try:
            details = action()
            item.update(status='PASS', details=details)
        except BaseException as error:
            item.update(status='FAIL', error_type=type(error).__name__)
            if isinstance(error, AssertionError):
                item['reason'] = str(error)  # All harness assertions use static reasons.
            raise
        finally:
            item['seconds'] = round(time.monotonic() - started, 6)
            self.save()
