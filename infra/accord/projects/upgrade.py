#!/usr/bin/env python3
"""Capable project update: full snapshot, additive migration, guarded roll-forward."""
import argparse
import json
import os
from pathlib import Path
import re
import signal
import sys

from guard import require_capable
from host_ops import PROJECTS, STATE, WORK, host, install_helpers, lock
from host_ops import owned_listener, port_free
from launch import require_process
from operate import fail_closed, journal, validate
from snapshot import snapshot
from staging import stage
from startup_hold import hold_startup, release_startup
from upgrade_storage import migrate_storage
from upgrade_verify import require_helpers, require_runtime
from validate import validate as validate_snapshot


def check(work, state, before=False, after=False):
    if state.get('kind') != 'capable-update':
        raise RuntimeError('dedicated capable-update journal required')
    validate(work, state)
    require_runtime()
    require_helpers(state, after)
    if before and (host.pointer(host.GO) != state['old_go'] or
                   host.pointer(host.BASE) != state['old_next']):
        raise RuntimeError('active pointers changed since update staging')
    if host.sha(host.NGINX) != state['nginx_sha256']:
        raise RuntimeError('existing ingress changed since update staging')
    if after and (host.pointer(host.GO) != 'releases/' + state['revision'] or
                  host.pointer(host.BASE) != 'releases/' + state['revision']):
        raise RuntimeError('active release pointers differ')


def apply(work, state, resume=False):
    allowed = ('activation-started', 'blocked-roll-forward') if resume else ('staged',)
    if state['phase'] not in allowed:
        raise RuntimeError('staged update or explicit failed-update resume required')
    check(work, state, before=not resume)
    with lock(STATE / '.maintenance.lock'):
        check(work, state, before=not resume)
        state['phase'] = 'activation-started'
        host.save_json(work / 'state.json', state)
        try:
            hold_startup(STATE, state['revision'])
            host.run('systemctl', 'stop', 'accord-next.service')
            host.run('systemctl', 'stop', 'accord.service')
            port_free(7892)
            port_free(7893)
            if 'pre_snapshot' not in state:
                saved = snapshot(STATE / 'board.db', STATE / 'web', PROJECTS,
                                 Path('/var/backups/accord-web'))
                state.update(pre_snapshot=str(saved), pre_manifest_sha256=host.sha(saved / 'manifest.json'))
                host.save_json(work / 'state.json', state)
            saved = Path(state['pre_snapshot'])
            if saved.parent != Path('/var/backups/accord-web') or host.sha(saved / 'manifest.json') != state['pre_manifest_sha256']:
                raise RuntimeError('immutable pre-update snapshot differs')
            manifest = validate_snapshot(saved)
            # Install the independent coordination/schema4 floor while the owned cold-boot
            # marker still blocks every restart, before any board DDL occurs.
            install_helpers(state['helpers'])
            require_helpers(state, after=True)
            host.switch(host.GO, 'releases/' + state['revision'])
            host.switch(host.BASE, 'releases/' + state['revision'])
            host.run('systemctl', 'daemon-reload')
            require_runtime()
            binary = Path('/opt/accord/current/agent-board')
            require_capable(binary, PROJECTS)
            if not state.get('storage_migrated'):
                state['preservation'] = migrate_storage(binary, saved, manifest)
                state['storage_migrated'] = True
                host.save_json(work / 'state.json', state)
            release_startup(STATE, state['revision'])
            host.run('systemctl', 'start', 'accord.service')
            host.ready(7892)
            require_process(owned_listener(7892, 'accord.service'), state['go_sha256'])
            host.run('systemctl', 'start', 'accord-next.service')
            host.ready(7893, '/app/login', nonce=True)
            owned_listener(7893, 'accord-next.service')
            check(work, state, after=True)
            state['phase'] = 'pending-https-smoke'
            host.save_json(work / 'state.json', state)
        except BaseException:
            fail_closed(work, state)
            raise
    print('Capable update active; full pre-snapshot and preservation verified. Independent HTTPS checks required.')


def finalize(work, state):
    if state['phase'] != 'pending-https-smoke' or not state.get('storage_migrated'):
        raise RuntimeError('active migrated update awaiting independent smoke required')
    check(work, state, after=True)
    host.run('systemctl', 'start', 'accord-web-backup.service')
    host.ready(7892)
    host.ready(7893, '/app/login', nonce=True)
    require_process(owned_listener(7892, 'accord.service'), state['go_sha256'])
    owned_listener(7893, 'accord-next.service')
    check(work, state, after=True)
    state['phase'] = 'complete'
    host.save_json(work / 'state.json', state)
    print('Capable update complete; all-project backup and owned runtime verified.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('stage', 'apply', 'resume', 'finalize'))
    parser.add_argument('--revision', required=True)
    parser.add_argument('--bundle', type=Path)
    parser.add_argument('--manifest-sha256')
    parser.add_argument('--go-binary', type=Path)
    parser.add_argument('--go-sha256')
    parser.add_argument('--expected-nginx-sha256')
    parser.add_argument('--stage-port', type=int, default=7894)
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux' or not re.fullmatch('[0-9a-f]{40}', args.revision):
        raise RuntimeError('root/Linux and full revision required')
    os.umask(0o077)
    for name in ('SIGTERM', 'SIGINT', 'SIGHUP'):
        signal.signal(getattr(signal, name), lambda *_: (_ for _ in ()).throw(RuntimeError('update interrupted')))
    with lock('/run/lock/accord-admin.lock'):
        if args.mode == 'stage':
            if args.stage_port != 7894 or any(getattr(args, key) is None for key in (
                    'bundle', 'manifest_sha256', 'go_binary', 'go_sha256', 'expected_nginx_sha256')):
                raise RuntimeError('all immutable stage arguments and port7894 required')
            stage(args, upgrading=True)
        else:
            work, state = journal(args.revision)
            if args.mode == 'finalize':
                finalize(work, state)
            else:
                apply(work, state, resume=args.mode == 'resume')


if __name__ == '__main__':
    try:
        main()
    except Exception:
        raise SystemExit('Capable update failed. Inspect private journal and roll forward; data rollback prohibited.')
