"""A subprocess boundary permits a real abrupt exit between active pointers."""
import json
import os
from pathlib import Path
import sys

from environment import Environment, NEW, OLD, guard, host, upgrade
from fixture import capture, command
from startup_hold import owned_marker
from upgrade_storage import facts
from validate import validate


def execute(root, mode):
    environment = Environment(root, existing=True)
    work = environment.work_release
    state = json.loads((work / 'state.json').read_text())
    before = facts(environment.state / 'board.db')
    authority_before = capture(environment.state)
    report = {'mode': mode, 'checks': []}
    with environment.patches():
        try:
            if mode != 'resume' and mode not in ('manual-marker', 'helper-drift'):
                environment.start_go()
            if mode == 'crash-after-go-pointer':
                original = host.switch
                def crash(base, target):
                    original(base, target)
                    if base == environment.go:
                        os._exit(73)
                host.switch = crash
            if mode == 'start-failure':
                environment.fail = 'start-go'
                try:
                    upgrade.apply(work, state)
                    raise AssertionError('expected startup failure did not occur')
                except RuntimeError as error:
                    assert 'injected' in str(error)
                assert state['phase'] == 'blocked-roll-forward'
                assert owned_marker(environment.state, NEW)
                assert not environment.processes
                assert state['storage_migrated'] is True
                report['checks'].append('startup-failure-retains-owned-marker-and-migrated-data')
                upgrade.apply(work, state, resume=True)
            elif mode in ('manual-marker', 'helper-drift', 'unknown-temp'):
                try:
                    upgrade.apply(work, state)
                    raise AssertionError('expected safety rejection did not occur')
                except RuntimeError:
                    pass
                if mode == 'manual-marker':
                    assert (environment.state / 'START_REQUIRES_INSPECTION').read_bytes() == b'MANUAL INSPECTION\n'
                    assert state['phase'] == 'staged'
                elif mode == 'helper-drift':
                    assert (environment.admin / 'projects/guard.py').read_bytes() == b'unknown manual helper\n'
                    assert state['phase'] == 'staged'
                else:
                    assert (environment.admin / 'projects/.guard.py.next-new').read_bytes() == b'unknown partial bytes'
                    assert state['phase'] == 'blocked-roll-forward'
                    assert owned_marker(environment.state, NEW)
                assert host.pointer(environment.go) == 'releases/' + OLD
                assert facts(environment.state / 'board.db') == before
                assert capture(environment.state) == authority_before
                report['checks'].append(mode + '-preserved-rejected-no-ddl-or-pointer-change')
                return report
            else:
                upgrade.apply(work, state, resume=mode == 'resume')
            assert state['phase'] == 'pending-https-smoke'
            assert state['preservation']['logical_rows_preserved'] is True
            assert state['preservation']['databases_preserved'] == 7
            assert not (environment.state / 'START_REQUIRES_INSPECTION').exists()
            assert host.pointer(environment.go) == host.pointer(environment.next) == 'releases/' + NEW
            assert all(facts(environment.state / 'board.db')[key] == value for key, value in before.items())
            assert not (environment.admin / 'projects/.guard.py.next-new').exists()
            current = capture(environment.state)
            assert len(authority_before) == len(current) == 7
            for database, tables in authority_before.items():
                for table, rows in tables.items():
                    assert current[database][table] == rows, 'old authority/history rows changed'
            report['checks'].append('actual-migrate-three-boards-registry-web-seven-db-preserved')
            environment.require_process(environment.processes['go'].child.pid, state['go_sha256'])
            old_binary = environment.go / 'releases' / OLD / 'agent-board'
            try:
                guard.require_capable(old_binary, environment.projects)
                raise AssertionError('old binary accepted schema4 floor')
            except RuntimeError:
                pass
            report['checks'].append('actual-old-capability-denied-and-new-owned-listener-ready')
            upgrade.finalize(work, state)
            assert state['phase'] == 'complete'
            saved = sorted(environment.backups.glob('projects-*'))
            assert len(saved) == 2
            assert validate(saved[-1])['default']['board_schema'] in (3, 4)
            assert sorted(validate(folder)['default']['board_schema'] for folder in saved) == [3, 4]
            report['checks'].append('finalize-new-full-schema4-backup-with-original-schema3-snapshot-retained')
            report['state'] = state['phase']
            report['actual_linux_proc_check'] = sys.platform == 'linux'
            return report
        finally:
            environment.cleanup()


if __name__ == '__main__':
    root = Path(sys.argv[1]).absolute()
    allowed = Path(__file__).resolve().parents[3] / '.local/task011-update-qa'
    assert allowed in root.parents and root.is_dir() and not root.is_symlink()
    result = execute(root, sys.argv[2])
    (root / 'worker-result.json').write_text(json.dumps(result, indent=2) + '\n')
    (root / 'worker-result.json').chmod(0o600)
