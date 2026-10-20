#!/usr/bin/env python3
"""One-way capable cutover. Failures preserve current data and require roll-forward."""
import argparse
import json
import os
from pathlib import Path
import re
import signal
import sys

from guard import require_capable
from host_ops import BACKUP_DROPIN, DROPIN, LEGACY_DROPIN, HERE, PROJECTS, STATE, WORK, host
from host_ops import helper_manifest, install_helpers, lock, preserved
from host_ops import owned_listener, port_free
from staging import stage
from launch import require_backup, require_launch, require_process
from startup_hold import hold_startup, owned_marker, release_startup


def owner_identity(value):
    if not re.fullmatch(r'[0-9a-f]{64}', value):
        raise argparse.ArgumentTypeError('existing public legacy token ID required; never pass a bearer')
    return value


def journal(revision):
    work = WORK / revision
    host.regular(work / 'state.json')
    state = json.loads((work / 'state.json').read_text())
    if state.get('revision') != revision:
        raise RuntimeError('release journal revision differs')
    return work, state


def validate(work, state, before=False):
    revision = state['revision']
    host.verify_bundle(host.BASE / 'releases' / revision, state['manifest_sha256'], revision)
    binary = host.GO / 'releases' / revision / 'agent-board'
    host.regular(binary)
    if host.sha(binary) != state['go_sha256'] or helper_manifest() != state['helpers']:
        raise RuntimeError('reviewed Go or operators changed')
    require_capable(binary, PROJECTS)
    host.regular(work / 'candidate-nginx')
    if host.sha(work / 'candidate-nginx') != state['candidate_sha256']:
        raise RuntimeError('reviewed ingress candidate changed')
    if preserved() != state['preserved']:
        raise RuntimeError('existing environment, timers, units or drop-ins changed')
    for marker in ('RESTORE_REQUIRES_RESET', 'START_REQUIRES_INSPECTION'):
        if (STATE / marker).exists() or (STATE / marker).is_symlink():
            if (marker == 'START_REQUIRES_INSPECTION' and
                    state['phase'] in ('blocked-roll-forward', 'activation-started') and
                    owned_marker(STATE, revision)):
                continue
            raise RuntimeError('unresolved existing recovery guard')
    if before:
        if (host.sha(host.NGINX) != state['nginx_sha256'] or
                host.pointer(host.GO) != state['old_go'] or host.pointer(host.BASE) != state['old_next']):
            raise RuntimeError('active pointers or nginx changed since staging')
        if DROPIN.exists() or LEGACY_DROPIN.exists() or BACKUP_DROPIN.exists() or PROJECTS.exists():
            raise RuntimeError('initial project cutover expects no prior registry activation')


def fail_closed(work, state):
    # Never select the previous Go: it cannot enforce the new registry ACL.
    # Do not replace DBs, remove guards or pretend an old backup is current data.
    failures = []
    try:
        hold_startup(STATE, state['revision'])
    except Exception:
        failures.append('startup-inspection-marker')
    for service in ('accord.service', 'accord-next.service'):
        try:
            host.run('systemctl', 'stop', service)
        except Exception:
            failures.append(service)
    state.update(phase='blocked-roll-forward', stop_failures=failures)
    host.save_json(work / 'state.json', state)


def apply(work, state, resume=False):
    if (not resume and state['phase'] != 'staged' or
            resume and state['phase'] not in ('blocked-roll-forward', 'activation-started')):
        raise RuntimeError('staged release or explicit failed-cutover resume required')
    validate(work, state, before=not resume)
    if not resume:
        # Backup owns this lock itself; run it before the release takes it.
        host.run('systemctl', 'start', 'accord-web-backup.service')
        host.ready(7892)
    with lock(STATE / '.maintenance.lock'):
        validate(work, state, before=not resume)
        state['phase'] = 'activation-started'
        host.save_json(work / 'state.json', state)
        try:
            hold_startup(STATE, state['revision'])
            host.run('systemctl', 'stop', 'accord-next.service')
            host.run('systemctl', 'stop', 'accord.service')
            port_free(7892)
            port_free(7893)
            install_helpers(state['helpers'])
            host.switch(host.GO, 'releases/' + state['revision'])
            host.switch(host.BASE, 'releases/' + state['revision'])
            host.run('systemctl', 'daemon-reload')
            host.run('systemd-analyze', 'verify', '/etc/systemd/system/accord.service',
                     '/etc/systemd/system/accord-web-backup.service')
            require_launch()
            require_backup()
            require_capable(Path('/opt/accord/current/agent-board'), PROJECTS)
            # Explicit bootstrap is an offline local administrator operation.
            # Same-identity replay must be idempotent across power-loss resume.
            host.run('runuser', '-u', 'accord', '--', '/opt/accord/current/agent-board',
                     'projects', 'bootstrap-owner', '--projects-data', str(PROJECTS),
                     '--db', str(STATE / 'board.db'), '--web-data', str(STATE / 'web'),
                     '--identity', state['owner_identity'])
            release_startup(STATE, state['revision'])
            host.run('systemctl', 'start', 'accord.service')
            host.ready(7892)
            require_process(owned_listener(7892, 'accord.service'), state['go_sha256'])
            host.run('systemctl', 'start', 'accord-next.service')
            host.ready(7893, '/app/login', nonce=True)
            owned_listener(7893, 'accord-next.service')
            host.atomic(host.NGINX, (work / 'candidate-nginx').read_bytes())
            host.run('nginx', '-t')
            host.run('systemctl', 'reload', 'nginx')
            validate(work, state)
            state['phase'] = 'pending-https-smoke'
            host.save_json(work / 'state.json', state)
        except BaseException:
            fail_closed(work, state)
            raise
    print('Capable candidate and selected owner active. Independent HTTPS smoke required.')


def finalize(work, state):
    if state['phase'] != 'pending-https-smoke':
        raise RuntimeError('active candidate awaiting explicit operator and independent smoke required')
    validate(work, state)
    require_launch()
    require_backup()
    if LEGACY_DROPIN.exists() or LEGACY_DROPIN.is_symlink():
        raise RuntimeError('superseded startup override still present')
    if (host.pointer(host.GO) != 'releases/' + state['revision'] or
            host.pointer(host.BASE) != 'releases/' + state['revision'] or
            host.sha(host.NGINX) != state['candidate_sha256']):
        raise RuntimeError('active capable release drift')
    for path, source in ((DROPIN, HERE / 'accord-projects.conf'),
                         (BACKUP_DROPIN, HERE / 'accord-projects-backup.conf')):
        host.regular(path)
        if path.read_bytes() != source.read_bytes():
            raise RuntimeError('mandatory project runtime/backup guard differs')
    host.run('systemctl', 'start', 'accord-web-backup.service')
    host.ready(7892)
    host.ready(7893, '/app/login', nonce=True)
    require_process(owned_listener(7892, 'accord.service'), state['go_sha256'])
    owned_listener(7893, 'accord-next.service')
    validate(work, state)
    state['phase'] = 'complete'
    host.save_json(work / 'state.json', state)
    print('Capable release complete with verified all-project backup. Data retained.')


def interrupted(_signal, _frame):
    raise RuntimeError('project release interrupted; inspect journal and roll forward')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('stage', 'apply', 'resume', 'finalize'))
    parser.add_argument('--revision', required=True)
    parser.add_argument('--bundle', type=Path)
    parser.add_argument('--manifest-sha256')
    parser.add_argument('--go-binary', type=Path)
    parser.add_argument('--go-sha256')
    parser.add_argument('--expected-nginx-sha256')
    parser.add_argument('--owner-identity', type=owner_identity,
                        help='explicit existing legacy token ID, never a bearer')
    parser.add_argument('--stage-port', type=int, default=7894)
    args = parser.parse_args()
    if (os.geteuid() != 0 or sys.platform != 'linux' or
            not re.fullmatch(r'[0-9a-f]{40}', args.revision)):
        raise RuntimeError('root/Linux and full source revision required')
    os.umask(0o077)
    for name in ('SIGTERM', 'SIGINT', 'SIGHUP'):
        signal.signal(getattr(signal, name), interrupted)
    with lock('/run/lock/accord-admin.lock'):
        if args.mode == 'stage':
            if (args.stage_port != 7894 or any(getattr(args, name) is None for name in
                    ('bundle', 'manifest_sha256', 'go_binary', 'go_sha256', 'expected_nginx_sha256', 'owner_identity'))):
                raise RuntimeError('reviewed stage7894 and all manifest arguments required')
            stage(args)
        else:
            work, state = journal(args.revision)
            if args.mode in ('apply', 'resume'):
                apply(work, state, resume=args.mode == 'resume')
            else:
                finalize(work, state)


if __name__ == '__main__':
    try:
        main()
    except Exception:
        raise SystemExit('Project release failed. Inspect private journal; legacy rollback is prohibited.')
