#!/usr/bin/env python3
"""Reviewed root-only stage/apply/finalize/rollback for this existing Accord VPS.

The only active files changed are Go/Next current, accord-next.service and the
public nginx include. Existing env, native routes and all timers stay intact.
"""
import argparse
import fcntl
import json
import os
import pathlib
import re
import signal
import sys

from host import BASE, GO, NGINX, UNIT, BACKUPS, atomic, pointer, ready
from host import regular, run, save_json, sha, switch, verify_bundle
from recover import rollback
from render import port
from stage import preserved, stage


def lock(path):
    descriptor = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    handle = os.fdopen(descriptor, 'w')
    fcntl.flock(handle, fcntl.LOCK_EX)
    return handle


def state_for(revision):
    work = BACKUPS / revision
    regular(work / 'state.json')
    state = json.loads((work / 'state.json').read_text())
    if state.get('revision') != revision:
        raise RuntimeError('deployment journal source mismatch')
    return work, state


def validate(work, state, before=False):
    revision = state['revision']
    verify_bundle(BASE / 'releases' / revision, state['manifest_sha256'], revision)
    regular(GO / 'releases' / revision / 'agent-board')
    if sha(GO / 'releases' / revision / 'agent-board') != state['go_sha256']:
        raise RuntimeError('installed immutable Go binary changed')
    if preserved() != state['preserved']:
        raise RuntimeError('existing timers/environment/drop-in changed')
    for name, key in (('candidate-nginx', 'candidate_sha256'), ('accord-next.service', 'unit_sha256')):
        regular(work / name)
        if sha(work / name) != state[key]:
            raise RuntimeError('candidate config changed after stage')
    if before:
        regular(NGINX)
        if (sha(NGINX) != state['nginx_sha256'] or pointer(GO) != state['old_go']
                or pointer(BASE) != state['old_next']):
            raise RuntimeError('active deployment changed after stage')
        current_unit = UNIT.read_bytes() if UNIT.exists() else None
        old_unit = (work / 'previous-unit').read_bytes() if state['old_unit'] else None
        if current_unit != old_unit:
            raise RuntimeError('Next unit changed after stage')
    for guard in ('RESTORE_REQUIRES_RESET', 'START_REQUIRES_INSPECTION'):
        if (pathlib.Path('/var/lib/accord') / guard).exists():
            raise RuntimeError('Accord has an unresolved recovery guard')


def apply(work, state):
    if state['phase'] != 'staged':
        raise RuntimeError('Linux staging success required before apply')
    validate(work, state, before=True)
    # Existing backup unit holds its own maintenance lock and returns Go healthy.
    # Run before taking that same lock to avoid deadlocking the backup writer.
    run('systemctl', 'start', 'accord-web-backup.service')
    ready(7892)
    with lock('/var/lib/accord/.maintenance.lock'):
        validate(work, state, before=True)
        state['phase'] = 'applying'
        save_json(work / 'state.json', state)
        try:
            run('systemctl', 'stop', 'accord.service')
            switch(GO, 'releases/' + state['revision'])
            switch(BASE, 'releases/' + state['revision'])
            atomic(UNIT, (work / 'accord-next.service').read_bytes())
            run('systemd-analyze', 'verify', str(UNIT))
            run('systemctl', 'daemon-reload')
            run('systemctl', 'start', 'accord.service')
            ready(7892)
            run('systemctl', 'restart', 'accord-next.service')
            ready(state['port'], '/app/login', nonce=True)
            atomic(NGINX, (work / 'candidate-nginx').read_bytes())
            run('nginx', '-t')
            run('systemctl', 'reload', 'nginx')
            run('systemctl', 'enable', 'accord-next.service')
            validate(work, state)
            state['phase'] = 'pending-https-smoke'
            save_json(work / 'state.json', state)
        except BaseException:
            rollback(work, state)
            raise
    print('Candidate active; authenticated HTTPS smoke required before finalize.')


def finalize(work, state):
    if state['phase'] != 'pending-https-smoke':
        raise RuntimeError('active candidate awaiting HTTPS smoke required')
    try:
        validate(work, state)
        if (pointer(GO) != 'releases/' + state['revision'] or
                pointer(BASE) != 'releases/' + state['revision'] or
                sha(NGINX) != state['candidate_sha256'] or sha(UNIT) != state['unit_sha256']):
            raise RuntimeError('active candidate pointer/config drift')
        # Operator calls this only after separate authenticated external checks.
        run('systemctl', 'start', 'accord-web-backup.service')
        ready(7892)
        ready(state['port'], '/app/login', nonce=True)
        validate(work, state)
        state['phase'] = 'complete'
        save_json(work / 'state.json', state)
    except BaseException:
        with lock('/var/lib/accord/.maintenance.lock'):
            rollback(work, state)
        raise
    print('Release complete; consistent backup and both runtimes healthy.')


def interrupted(_signal, _frame):
    raise RuntimeError('release operation interrupted; recovery required')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('stage', 'apply', 'finalize', 'rollback'))
    parser.add_argument('--revision', required=True)
    parser.add_argument('--bundle', type=pathlib.Path)
    parser.add_argument('--manifest-sha256')
    parser.add_argument('--go-binary', type=pathlib.Path)
    parser.add_argument('--go-sha256')
    parser.add_argument('--expected-nginx-sha256')
    parser.add_argument('--port', type=port, default=7893)
    parser.add_argument('--stage-port', type=port, default=7894)
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on Linux with systemd')
    if not re.fullmatch(r'[0-9a-f]{40}', args.revision):
        raise RuntimeError('full source revision required')
    os.umask(0o077)
    for name in ('SIGTERM', 'SIGINT', 'SIGHUP'):
        if hasattr(signal, name):
            signal.signal(getattr(signal, name), interrupted)
    with lock('/run/lock/accord-admin.lock'):
        if args.mode == 'stage':
            if (args.port == args.stage_port or
                    any(getattr(args, name) is None for name in ('bundle', 'manifest_sha256',
                        'go_binary', 'go_sha256', 'expected_nginx_sha256'))):
                raise RuntimeError('all stage arguments and distinct ports required')
            stage(args)
        else:
            work, state = state_for(args.revision)
            if args.mode == 'apply':
                apply(work, state)
            elif args.mode == 'finalize':
                finalize(work, state)
            else:
                if state['phase'] not in ('applying', 'pending-https-smoke', 'complete', 'rollback-incomplete'):
                    raise RuntimeError('no activated candidate to roll back')
                with lock('/var/lib/accord/.maintenance.lock'):
                    rollback(work, state)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print('Next release operation failed: ' + str(error), file=sys.stderr)
        sys.exit(1)
