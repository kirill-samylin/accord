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
from fixture import command, rebind, seed

REPO = Path(__file__).resolve().parents[3]
BASE = REPO / '.local/task011-update-qa'
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()


def inventory():
    files = sorted([p for p in Path(__file__).parent.glob('*.py')] +
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
    old = Path(os.environ.get('ACCORD_UPDATE_OLD_BINARY', REPO / '.local/task011/legacy/agent-board')).resolve()
    new = Path(os.environ.get('ACCORD_UPDATE_BINARY', REPO / 'bin/agent-board')).resolve()
    assert old.is_file() and new.is_file() and not old.is_symlink() and not new.is_symlink()
    known_old = {'darwin': '4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085',
                 'linux': '66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa'}
    assert sha(old) == known_old[sys.platform], 'prior-schema3 executable differs from verified platform proof'
    old_caps = json.loads(command(old, ['capabilities', '--json']))
    new_caps = json.loads(command(new, ['capabilities', '--json']))
    assert old_caps == {'project_registry': 1, 'project_acl': 1, 'registry_schema': [1], 'agent_profiles': 1, 'board_schema': [3]}
    assert new_caps.get('agent_profiles') == 1 and new_caps.get('board_schema') == [4] and new_caps.get('coordination') == 1
    if sys.platform == 'linux':
        assert sha(new) == os.environ['ACCORD_UPDATE_BINARY_SHA256'], 'Linux candidate differs from pinned release proof'
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
              'old_source_revision': os.environ.get('ACCORD_UPDATE_OLD_REVISION', '343b93c71297c58c972f73fed4d1d4ca3a349740' if sys.platform == 'darwin' else 'cbf202086431335d4657c722a0f75328a09dc4ea'),
              'tests_and_operator_files': source,
              'limitations': ['systemd lifecycle/introspection simulated; real owned Go processes and SQLite',
                              'Next bundle/runtime simulated; Next hardened stage is a separate deployment check',
                              'fixture UID adapter uses invoking user; production root-owner checks separate',
                              'one host; no physical second PC', 'no production paths/services/ports used']}
    try:
        baseline = root / 'baseline'
        assert len(seed(old, baseline)) == 3
        report['checks'].append({'id': 'actual-old-schema3-created-three-projects-tasks-notes-repository-authority', 'result': 'PASS'})
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
                    assert db.execute('PRAGMA user_version').fetchone()[0] == 3
                arguments[-1] = 'resume'
                status = worker(arguments, case / 'resume.log')
            assert status == 0, mode + ' failed; private diagnostics retained'
            detail = json.loads((case / 'worker-result.json').read_text())
            report['checks'].append({'id': mode, 'result': 'PASS', 'observations': detail['checks']})
        assert inventory() == source, 'test/operator source changed during run'
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
