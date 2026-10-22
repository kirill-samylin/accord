#!/usr/bin/env python3
"""Real Go/SQLite update rehearsal. systemd/Next boundaries are explicit simulations."""
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import stat
import sqlite3
import subprocess
import sys
import time

from environment import Environment, NEW, OLD, host
from seed_extra import enrich
from fixture import command, rebind, seed

REPO = Path(__file__).resolve().parents[3]
BASE = REPO / '.local/task012-update-qa'
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()


def inventory():
    files = sorted([p for folder in ('task012-update', 'task011-update') for p in (REPO / 'tests/process' / folder).glob('*.py')] +
                   [p for p in (REPO / 'infra/accord').rglob('*') if p.suffix in ('.py', '.conf', '.in')])
    return {p.relative_to(REPO).as_posix(): sha(p) for p in files}


def worker(arguments, output):
    process = subprocess.Popen(arguments, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True)
    try:
        data, _ = process.communicate(timeout=100)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGKILL)
        data, _ = process.communicate(timeout=10)
        output.write_bytes(data)
        raise AssertionError('worker timed out; owned process group terminated')
    output.write_bytes(data)
    return process.returncode


def main():
    os.umask(0o077)
    source = inventory()
    old_infra = Path(os.environ['ACCORD_UPDATE_OLD_INFRA']).resolve()
    baseline_helpers = {str(p.relative_to(old_infra)): sha(p) for p in old_infra.rglob('*')
                        if p.is_file() and p.suffix in ('.py', '.conf', '.in')}
    old = Path(os.environ.get('ACCORD_UPDATE_OLD_BINARY', REPO / '.local/task011/candidate/agent-board')).resolve()
    new = Path(os.environ.get('ACCORD_UPDATE_BINARY', REPO / 'bin/agent-board')).resolve()
    assert old.is_file() and new.is_file() and not old.is_symlink() and not new.is_symlink()
    known_old = {'darwin': 'daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac',
                 'linux': 'a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d'}
    assert sha(old) == known_old[sys.platform], 'prior-schema4 executable differs from verified platform proof'
    old_caps = json.loads(command(old, ['capabilities', '--json']))
    new_caps = json.loads(command(new, ['capabilities', '--json']))
    assert old_caps == {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1], 'agent_profiles': 1, 'coordination': 1, 'board_schema': [4]}
    assert new_caps.get('agent_profiles') == 1 and new_caps.get('board_schema') == [4] and new_caps.get('coordination') == 1 and new_caps.get('deduplication') == 1
    assert sha(new) == os.environ['ACCORD_UPDATE_BINARY_SHA256'], 'candidate differs from pinned release proof'
    for parent in (BASE,) + tuple(BASE.parents):
        if parent.exists() or parent.is_symlink():
            info = parent.lstat()
            assert stat.S_ISDIR(info.st_mode) and not info.st_mode & 0o022, 'unsafe private fixture ancestry'
        if parent == REPO:
            break
    BASE.mkdir(parents=True, exist_ok=True, mode=0o700)
    assert BASE.stat().st_mode & 0o777 == 0o700
    root = BASE / ('run-' + str(time.time_ns()))
    root.mkdir(mode=0o700)
    report = {'schema': 1, 'result': 'FAIL', 'platform': sys.platform,
              'old_binary_sha256': sha(old), 'binary_sha256': sha(new), 'checks': [],
              'new_revision': os.environ.get('ACCORD_UPDATE_REVISION', 'local-build-provenance-separate'),
              'old_source_revision': os.environ.get('ACCORD_UPDATE_OLD_REVISION', 'f2edac9-product-identical-e53c18c' if sys.platform == 'darwin' else 'e53c18c9643b9ee086141d0f227ab485f4f88cac'),
              'tests_and_operator_files': source, 'old_operator_files': baseline_helpers,
              'limitations': ['systemd lifecycle/introspection simulated; real owned Go processes and SQLite',
                              'Next bundle/runtime simulated; Next hardened stage is a separate deployment check',
                              'fixture UID adapter uses invoking user; production root-owner checks separate',
                              'one host; no physical second PC', 'no production paths/services/ports used']}
    try:
        baseline = root / 'baseline'
        assert len(seed(old, baseline)) == 3
        enrich(old, baseline)
        report['checks'].append({'id': 'actual-old-schema4-three-projects-tasks-notes-profiles-assignments-links-ACL-epochs-synthetic-history', 'result': 'PASS'})
        for mode in ('success', 'start-failure', 'crash-after-go-pointer', 'manual-marker', 'helper-drift', 'expected-temp', 'unknown-temp'):
            case = root / mode
            case.mkdir(mode=0o700)
            shutil.copytree(baseline, case / 'state')
            rebind(case / 'state')
            environment = Environment(case, old, new)
            with environment.patches():
                environment.prepare()
            if mode == 'manual-marker':
                marker = case / 'state/START_REQUIRES_INSPECTION'
                marker.write_bytes(b'MANUAL INSPECTION\n')
                marker.chmod(0o600)
            if mode == 'helper-drift':
                (case / 'admin/projects/guard.py').write_bytes(b'unknown manual helper\n')
            if mode in ('expected-temp', 'unknown-temp'):
                temporary = case / 'admin/projects/.guard.py.next-new'
                temporary.write_bytes((REPO / 'infra/accord/projects/guard.py').read_bytes() if mode == 'expected-temp' else b'unknown partial bytes')
                temporary.chmod(0o644)
            arguments = [sys.executable, '-B', str(Path(__file__).with_name('worker.py')), str(case), mode]
            status = worker(arguments, case / 'worker.log')
            if mode == 'crash-after-go-pointer':
                assert status == 73, 'real abrupt-exit boundary was not reached'
                state = json.loads((case / 'journals' / NEW / 'state.json').read_text())
                assert state['phase'] == 'activation-started' and state.get('pre_snapshot')
                assert (case / 'state/START_REQUIRES_INSPECTION').is_file()
                assert os.readlink(case / 'go/current') == 'releases/' + NEW
                assert os.readlink(case / 'next/current') == 'releases/' + OLD
                with closing(sqlite3.connect(case / 'state/board.db')) as db:
                    assert db.execute('PRAGMA user_version').fetchone()[0] == 4
                arguments[-1] = 'resume'
                status = worker(arguments, case / 'resume.log')
            assert status == 0, mode + ' failed; private diagnostics retained'
            detail = json.loads((case / 'worker-result.json').read_text())
            if 'restore' in detail:
                report['fullrestore'] = detail['restore']
            report['checks'].append({'id': mode, 'result': 'PASS', 'observations': detail['checks']})
        assert inventory() == source, 'test/operator source changed during run'
        assert baseline_helpers == {str(p.relative_to(old_infra)): sha(p) for p in old_infra.rglob('*')
                                    if p.is_file() and p.suffix in ('.py', '.conf', '.in')}
        report.update(result='PASS', tests_and_operator_files=source, source_unchanged=True)
    except BaseException as error:
        report['failure'] = {'type': type(error).__name__, 'label': str(error)[:160]}
        raise
    finally:
        report['source_unchanged'] = inventory() == source
        if not report['source_unchanged']:
            report['result'] = 'FAIL'
        report['finished_at_ns'] = time.time_ns()
        (root / 'results.json').write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'result': report['result'], 'checks': len(report['checks']), 'report': str((root / 'results.json').relative_to(REPO))}))


if __name__ == '__main__':
    main()
