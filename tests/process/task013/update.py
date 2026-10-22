#!/usr/bin/env python3
"""Actual all-project schema4→5 updater and offline full restore process QA."""
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import shutil
import sys
import time
import uuid

from update_environment import Environment, OLD, NEW, REPO, host, upgrade, installed_guard
from fixture import capture, command, seed
from restore import restore
from snapshot import snapshot
from validate import validate

sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()

def files(folder):
    return {str(p.relative_to(folder)): sha(p) for p in folder.rglob('*') if p.is_file()}

def inventory():
    folders = ('tests/process/task013', 'tests/process/task011-update', 'infra/accord')
    return {str(p.relative_to(REPO)): sha(p) for folder in folders
            for p in sorted((REPO / folder).rglob('*')) if p.is_file() and p.suffix in ('.py', '.cjs', '.conf', '.in')}

def main():
    os.umask(0o077)
    sources = inventory()
    base = REPO / '.local/task013-qa'
    base.mkdir(mode=0o700, exist_ok=True)
    root = base / ('update-' + str(time.time_ns()))
    root.mkdir(mode=0o700)
    old = Path(os.environ.get('ACCORD_TASK013_OLD_BINARY', REPO / '.local/task012/candidate/agent-board')).resolve()
    new = Path(os.environ.get('ACCORD_TASK013_BINARY', REPO / 'bin/agent-board')).resolve()
    old_input, new_input = old, new
    old, new = root / 'old-agent-board', root / 'new-agent-board'
    shutil.copyfile(old_input, old)
    shutil.copyfile(new_input, new)
    old.chmod(0o700)
    new.chmod(0o700)
    old_proof = json.loads((REPO / '.local/task012/candidate/proof.json').read_text())
    assert sha(old) == old_proof['binary_sha256']
    assert json.loads(command(old, ['capabilities', '--json']))['board_schema'] == [4]
    assert json.loads(command(new, ['capabilities', '--json']))['managed_agents'] == 1
    report = {'result': 'FAIL', 'old_binary_sha256': sha(old), 'binary_sha256': sha(new),
              'source_revision': os.environ.get('ACCORD_TASK013_REVISION', 'local-diff'), 'sources': sources,
              'checks': [], 'limitations': ['real Go/SQLite/backup/restore/auth-reset; systemd/Next lifecycle simulated',
                                           'one synthetic Mac; production and Windows NOT-RUN']}
    env = None
    try:
        identifiers = seed(old, root / 'state')
        before = capture(root / 'state')
        env = Environment(root, old, new)
        with env.patches():
            state = env.prepare()
            env.start_go()
            upgrade.apply(env.work_release, state)
            assert state['phase'] == 'pending-https-smoke'
            assert state['preservation']['logical_rows_preserved']
            current = capture(env.state)
            assert len(current) == len(before) == 7
            for name, tables in before.items():
                for table, rows in tables.items():
                    assert current[name][table] == rows, 'old table rows changed'
            for name, tables in current.items():
                if name.endswith('/board'):
                    assert 'managed_hosts' in tables and 'managed_metadata' in tables
            installed = installed_guard(env.admin / 'projects/guard.py')
            installed.require_capable(env.go / 'releases' / NEW / 'agent-board', env.projects)
            try:
                installed.require_capable(env.go / 'releases' / OLD / 'agent-board', env.projects)
            except RuntimeError:
                pass
            else:
                raise AssertionError('old schema4 writer accepted')
            report['checks'].append('actual-capable-update-seven-DB-oldrows-ACL-profiles-preserved-floor-before-DDL')
            token = root / 'synthetic-owner.token'
            project = identifiers[0]
            def call(op, body):
                return json.loads(command(new, ['call', op, '--server', env.processes['go'].url,
                                               '--token-file', token, '--project', project], body))
            session = call('describe', {'request_id': uuid.uuid4().hex, 'role': 'human',
                                       'machine': 'synthetic-update', 'run_id': uuid.uuid4().hex})['session_id']
            def managed(op, body):
                return call(op, {'request_id': uuid.uuid4().hex, 'session_id': session, **body})
            profile = managed('profile_create', {'name': 'QA restored managed profile', 'role': 'developer',
                                                 'description': 'Synthetic only', 'instructions': 'Preserve pinned',
                                                 'provider': 'unconfigured', 'model': ''})
            proof = uuid.uuid4().hex + uuid.uuid4().hex
            enrolled = managed('managed_host_enroll', {'name': 'qa-restored-host', 'host_secret': proof,
                                                      'controllers': [], 'workspaces': ['qa'], 'providers': ['codex']})
            poll = managed('managed_poll', {'host_id': enrolled['id'], 'host_secret': proof, 'worker_id': 'worker_restore'})
            queued = managed('managed_start', {'host_id': enrolled['id'], 'workspace': 'qa', 'provider': 'codex',
                                              'profile_id': profile['id'], 'profile_version': 1, 'prompt': 'Preserve queued'})
            # Leave a durable dispatch unacknowledged; restore must never auto-launch it.
            dispatched = managed('managed_poll', {'host_id': enrolled['id'], 'host_secret': proof,
                                                  'worker_id': 'worker_restore', 'worker_epoch': poll['worker_epoch']})
            assert dispatched['command']['dispatch_id']
            upgrade.finalize(env.work_release, state)
            assert state['phase'] == 'complete'
            env.stop('go')
            final = next(p for p in env.backups.glob('projects-*') if str(p) != state['pre_snapshot'])
            assert validate(final)['default']['board_schema'] == 5
            assert validate(Path(state['pre_snapshot']))['default']['board_schema'] == 4
            old_bytes, live_bytes = files(final), files(env.state)
            pre_restore = capture(env.state)
            target = root / 'restored-offline'
            restore(final, target, new)
            assert (target / 'RESTORE_REQUIRES_RESET').is_file()
            assert (target / 'RESTORE_AUTH_RESET_COMPLETE').is_file()
            post_restore = capture(target)
            for name, tables in pre_restore.items():
                if name.endswith('/board'):
                    for table in ('notes', 'idempotency', 'agent_profiles', 'agent_profile_revisions', 'coordination_metadata'):
                        assert post_restore[name][table] == tables[table]
                    folder = target if name == 'default/board' else target / 'projects-data/projects' / name.split('/')[0]
                    with closing(sqlite3.connect(folder / 'board.db')) as db:
                        assert db.execute('PRAGMA user_version').fetchone()[0] == 5
                        for raw, in db.execute('SELECT data FROM managed_hosts'):
                            value = json.loads(raw)
                            assert value['host']['state'] == 'disabled'
                            assert not value.get('secret_hash') and not value.get('worker_id')
                        for raw, in db.execute('SELECT data FROM managed_runs'):
                            assert json.loads(raw)['state'] == 'needs_recovery'
                        for raw, in db.execute('SELECT data FROM managed_commands'):
                            assert json.loads(raw)['command']['status'] == 'cancelled'
            with closing(sqlite3.connect(target / 'projects-data/registry.sqlite')) as db:
                assert db.execute('SELECT COUNT(*) FROM credentials WHERE revoked=0').fetchone()[0] == 0
                for table in ('sessions', 'actions', 'preauth', 'session_epochs'):
                    assert db.execute('SELECT COUNT(*) FROM '+table).fetchone()[0] == 0
                epochs = list(db.execute('SELECT project_id,identity_id,access_epoch FROM memberships ORDER BY 1,2'))
            with closing(sqlite3.connect(env.projects / 'registry.sqlite')) as db:
                original = list(db.execute('SELECT project_id,identity_id,access_epoch FROM memberships ORDER BY 1,2'))
            assert epochs == [(p, i, e+1) for p, i, e in original]
            assert files(final) == old_bytes and files(env.state) == live_bytes
            report['checks'].append('actual-fullrestore-seven-DB-authreset-disabled-hosts-erased-proofs-cancelled-dispatch-noauto-launch')
            report['checks'].append('immutable-snapshot-livebytes-preserved-businesshistory-ACL-epochs-advanced-bootguard-retained')
            assert sources == inventory(), 'test/operator source drift during run'
            report['source_unchanged'] = True
            report['result'] = 'PASS'
    except BaseException as error:
        report['failure'] = {'type': type(error).__name__, 'label': str(error)[:240]}
        raise
    finally:
        try:
            if env:
                env.cleanup()
        except BaseException as error:
            report['cleanup_failure'] = type(error).__name__
            report['result'] = 'FAIL'
            raise
        finally:
            report['binaries_unchanged'] = sha(new) == report['binary_sha256'] and sha(old) == report['old_binary_sha256']
            if not report['binaries_unchanged']:
                report['result'] = 'FAIL'
            (root / 'results.json').write_text(json.dumps(report, indent=2)+'\n')
            print(json.dumps({'result': report['result'], 'checks': len(report['checks']), 'report': str((root / 'results.json').relative_to(REPO))}))

if __name__ == '__main__':
    main()
