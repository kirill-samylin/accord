#!/usr/bin/python3
"""Fail-safe executable activation; database restoration is always operator-owned."""
import os
import sqlite3
import stat
import subprocess
import time
import urllib.request


FAILURES = (OSError, RuntimeError, sqlite3.Error, subprocess.SubprocessError)


def switch(base, revision):
    temporary = base / '.current.new'
    temporary.symlink_to('releases/' + revision)
    try:
        os.replace(temporary, base / 'current')
    finally:
        if temporary.is_symlink():
            temporary.unlink()


def healthy(timeout=15):
    # A schema4 migration measured 29.1s locally on the annual fixture. Install
    # uses a separate 300s cap only for older schemas; normal restart keeps15s.
    # A dead/restarting service fails immediately instead of consuming the cap.
    client = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            return False
        try:
            with client.open('http://127.0.0.1:7892/readyz', timeout=min(1, remaining)) as response:
                if response.status == 200:
                    return True
        except OSError:
            pass
        if not service_active():
            return False
        time.sleep(min(0.5, max(0, deadline - time.monotonic())))
    return False


def service_active():
    try:
        result = subprocess.run(['systemctl', 'is-active', '--quiet', 'accord.service'],
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                timeout=1, check=False)
        return result.returncode == 0
    except (OSError, subprocess.SubprocessError):
        return False


def check_guard(state):
    for name in ('RESTORE_REQUIRES_RESET', 'START_REQUIRES_INSPECTION'):
        marker = state / name
        if marker.exists() or marker.is_symlink():
            raise RuntimeError('recovery guard is present; refusing service start')


def inspection_required(state, message, cause):
    """Persist a separate boot guard; never remove the restore/reset guard."""
    owner = state.lstat()
    if not stat.S_ISDIR(owner.st_mode) or stat.S_IMODE(owner.st_mode) != 0o700:
        raise RuntimeError('service stopped; cannot persist guard in unsafe state directory') from cause
    marker = state / 'START_REQUIRES_INSPECTION'
    try:
        with marker.open('x', encoding='ascii') as stream:
            stream.write('Inspect schema and executable compatibility before explicitly restarting.\n')
            stream.flush()
            os.fchmod(stream.fileno(), 0o600)
            os.fsync(stream.fileno())
        descriptor = os.open(state, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
    except OSError as error:
        raise RuntimeError('activation failed; inspection guard could not be persisted; '
                           'operator must prevent service start immediately') from error
    raise RuntimeError(message + '; persistent inspection guard set; database not restored') from cause


def schema_snapshot(state):
    """Read the stopped database including WAL; never use immutable to ignore WAL."""
    owner = state.lstat()
    if not stat.S_ISDIR(owner.st_mode) or stat.S_IMODE(owner.st_mode) != 0o700:
        raise RuntimeError('database directory must be private and non-symlink')
    path = state / 'board.db'
    for suffix in ('', '-wal', '-shm', '-journal'):
        entry = state / ('board.db' + suffix)
        if not entry.exists() and not entry.is_symlink():
            continue
        info = entry.lstat()
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != owner.st_uid
                or stat.S_IMODE(info.st_mode) != 0o600):
            raise RuntimeError('database and journals must be private owned regular files')
    if not path.exists():
        return None
    database = sqlite3.connect(path.absolute().as_uri() + '?mode=ro', uri=True, timeout=5)
    try:
        version = database.execute('PRAGMA user_version').fetchone()[0]
        # Also reject an unversioned DDL change. Source/data-format migrations
        # must bump user_version; recovery cannot infer arbitrary compatibility.
        schema = tuple(database.execute(
            'SELECT type,name,tbl_name,sql FROM sqlite_schema ORDER BY type,name'))
        return version, schema
    finally:
        database.close()


def activate(previous, revision, state, command, switch, healthy, migration_healthy=None):
    """Activate once, restoring a previous executable only on known compatible schema."""
    check_guard(state)
    # If stop itself fails the executable and database have not been changed.
    command('systemctl', 'stop', 'accord.service')
    before = None
    try:
        before = schema_snapshot(state)
        if previous:
            if before is None:
                raise RuntimeError('previous release has no database; compatibility is unknown')
            command('runuser', '-u', 'accord', '--', '/usr/bin/python3',
                    '/opt/accord/admin/backup.py')
        switch(revision)
        check_guard(state)
        command('systemctl', 'start', 'accord.service')
        checker = healthy
        if before is not None and before[0] in (1, 2, 3) and migration_healthy is not None:
            checker = migration_healthy
        if not checker():
            raise RuntimeError('new executable failed readiness')
    except FAILURES as failure:
        recover(previous, before, state, command, switch, healthy, failure)


def recover(previous, before, state, command, switch, healthy, failure):
    # A failed systemctl start can leave a restart job scheduled. Explicit stop
    # cancels it before checking the DB or switching a release pointer.
    try:
        command('systemctl', 'stop', 'accord.service')
    except FAILURES as error:
        inspection_required(state, 'activation failed; service stop failed; operator repair required', error)
    if not previous:
        inspection_required(state, 'activation failed on first installation; service stopped; no previous release', failure)
    try:
        check_guard(state)
        after = schema_snapshot(state)
        if before is None or before[0] not in (1, 2, 3, 4) or before != after:
            raise RuntimeError('database schema changed or compatibility is unknown')
    except FAILURES as error:
        inspection_required(state, 'activation failed; service stopped; database compatibility unproven; '
                            'current executable retained; operator repair required', error)
    try:
        switch(previous.split('/')[-1])
        check_guard(state)
        command('systemctl', 'start', 'accord.service')
        if not healthy():
            raise RuntimeError('previous executable failed readiness')
    except FAILURES as error:
        try:
            command('systemctl', 'stop', 'accord.service')
        except FAILURES:
            inspection_required(state, 'activation and recovery failed; service stop failed; operator repair required', error)
        inspection_required(state, 'activation and recovery failed; previous executable selected; '
                            'service stopped; operator repair required', error)
    raise RuntimeError('activation failed; previous executable restored and ready; database not restored') from failure
