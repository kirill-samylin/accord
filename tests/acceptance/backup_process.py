#!/usr/bin/env python3
"""Run isolated backup acceptance with prebuilt CLI and reviewed wrapper; stdlib only."""
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import platform
import shutil
import sqlite3
import stat
import sys
import tempfile
import time
import uuid

from backup_process_support import DRIVER, Evidence, counts, database, private_write, require, sha, tree


class Acceptance:
    def __init__(self, args, work, evidence):
        self.args, self.work, self.evidence = args, work, evidence
        self.state, self.backups = work / 'state', work / 'backups'
        self.state.mkdir(mode=0o700)
        self.backups.mkdir(mode=0o700)
        self.db, self.driver = self.state / 'board.db', work / 'driver.py'
        private_write(self.driver, DRIVER.encode())
        self.latest = []

    def cli(self, label, *arguments, hidden=False):
        return self.evidence.command(label, [self.args.binary, *arguments], hidden=hidden)

    def wrapper(self, label, plan=False, expected=0):
        argv = [sys.executable, self.driver, self.args.backup_wrapper, self.state,
                self.backups, self.args.binary]
        if plan:
            argv.append('--plan')
        return self.evidence.command(label, argv, expected=expected)

    def snapshots(self):
        return sorted(self.backups.glob('board-*.db'))

    def verify(self, path, expected):
        require(stat.S_IMODE(path.stat().st_mode) == 0o600, 'snapshot_mode')
        require(path.stat().st_nlink == 1, 'snapshot_hardlink')
        checksum = path.with_name(path.name + '.sha256')
        require(stat.S_IMODE(checksum.stat().st_mode) == 0o600, 'checksum_mode')
        require(checksum.read_text() == sha(path) + '  ' + path.name + '\n', 'checksum_exact')
        require(database(path) == expected, 'snapshot_full_schema_rows')
        require(not any(Path(str(path) + suffix).exists() for suffix in ('-wal', '-shm', '-journal')), 'snapshot_sidecars')

    def bootstrap(self):
        self.cli('token_create_first', 'token', 'create', '--db', self.db,
                 '--owner', 'qa068', '--machine', 'synthetic', '--output', self.work / 'first.token', hidden=True)
        self.first_image = database(self.db)
        require(self.first_image[0] >= 2, 'schema_version')
        require(len(self.first_image[2]['tokens']) == 1, 'first_token_count')
        require(len(self.first_image[2]['events']) >= 1, 'first_audit_count')
        out, _ = self.wrapper('bootstrap_apply')
        require(b'redundancy: 1/2' in out, 'bootstrap_redundancy')
        require(len(self.snapshots()) == 1, 'first_snapshot_count')
        self.first = self.snapshots()[0]
        self.verify(self.first, self.first_image)
        return {'schema': self.first_image[0], 'table_counts': counts(self.first_image), 'redundancy': '1/2'}

    def second(self):
        self.cli('token_create_second', 'token', 'create', '--db', self.db,
                 '--owner', 'qa068', '--machine', 'synthetic', '--output', self.work / 'second.token', hidden=True)
        self.second_image = database(self.db)
        require(len(self.second_image[2]['tokens']) == 2, 'second_token_count')
        require(len(self.second_image[2]['events']) > len(self.first_image[2]['events']), 'second_audit_growth')
        out, _ = self.wrapper('second_apply')
        require(b'redundancy: 2/2' in out, 'second_redundancy')
        require(len(self.snapshots()) == 2, 'second_snapshot_count')
        self.second_copy = next(path for path in self.snapshots() if path != self.first)
        self.verify(self.first, self.first_image)
        self.verify(self.second_copy, self.second_image)
        self.latest = {self.first.name, self.second_copy.name}
        return {'table_counts': counts(self.second_image), 'redundancy': '2/2'}

    def history(self):
        now = dt.datetime.now(dt.timezone.utc)
        hour = now.replace(minute=0, second=0, microsecond=0)
        day = now.replace(hour=0, minute=0, second=0, microsecond=0)
        items = [(hour - dt.timedelta(days=40), False),
                 (hour - dt.timedelta(hours=2) + dt.timedelta(minutes=5), False),
                 (hour - dt.timedelta(hours=2) + dt.timedelta(minutes=10), True),
                 (day - dt.timedelta(days=3) + dt.timedelta(hours=1), False),
                 (day - dt.timedelta(days=3) + dt.timedelta(hours=2), True),
                 (day - dt.timedelta(days=6) + dt.timedelta(hours=12), True),
                 (day - dt.timedelta(days=7), False)]
        self.delete, self.keep = set(), set(self.latest)
        for created, retained in items:
            name = 'board-' + created.strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex + '.db'
            path = self.backups / name
            shutil.copyfile(self.second_copy, path)
            path.chmod(0o600)
            private_write(self.backups / (name + '.sha256'), (sha(path) + '  ' + name + '\n').encode())
            (self.keep if retained else self.delete).add(name)
        self.unsafe = []
        for suffix in ('corrupt', 'symlink'):
            name = 'board-' + (day - dt.timedelta(days=50)).strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex + '.db'
            path = self.backups / name
            if suffix == 'corrupt':
                shutil.copyfile(self.second_copy, path)
                path.chmod(0o600)
                private_write(self.backups / (name + '.sha256'), b'0' * 64 + b'  ' + name.encode() + b'\n')
            else:
                path.symlink_to(self.second_copy.name)
                private_write(self.backups / (name + '.sha256'), (sha(self.second_copy) + '  ' + name + '\n').encode())
            self.unsafe.extend([name, name + '.sha256'])
        private_write(self.backups / 'pre-upgrade-unknown.db', b'synthetic-preserve')
        self.unsafe.append('pre-upgrade-unknown.db')
        before, source = tree(self.backups), tree(self.state)
        out, _ = self.wrapper('retention_plan', plan=True)
        plan = json.loads(out)
        require(tree(self.backups) == before and tree(self.state) == source, 'plan_readonly_bytes_names_mtime')
        require(plan['mode'] == 'plan' and plan['verified'] == 9 and plan['ignored'] == 2, 'plan_inventory')
        require(set(plan['protected']) == self.latest, 'plan_latest_two')
        require(set(plan['delete']) == self.delete, 'plan_exact_retention_deletes')
        self.unsafe_before = {name: before[name] for name in self.unsafe}
        self.plan = plan
        return plan

    def prune(self):
        self.wrapper('retention_apply')
        for name in self.delete:
            require(not (self.backups / name).exists() and not (self.backups / (name + '.sha256')).exists(), 'obsolete_pair_deleted')
        # A third current-hour snapshot may retire the older of the original two;
        # latest2 protection before apply forbids pre-copy deletion, not future retention.
        out, _ = self.wrapper('post_apply_plan', plan=True)
        plan = json.loads(out)
        require(not plan['delete'] and plan['ignored'] == 2, 'post_apply_plan_clean')
        require(self.second_copy.exists(), 'prior_latest_preserved')
        require(all((self.backups / name).exists() for name in self.keep - self.latest), 'preferred_buckets_preserved')
        after = tree(self.backups)
        require({name: after.get(name) for name in self.unsafe} == self.unsafe_before, 'unsafe_unchanged')
        verified_names = {path.name for path in self.snapshots()} - {name for name in self.unsafe if name.endswith('.db')}
        require(len(verified_names) == 5 and plan['verified'] == 5, 'retained_pair_count')
        for name in verified_names:
            self.verify(self.backups / name, self.second_image)
        return {'planned_obsolete_deleted': len(self.delete), 'verified_retained': plan['verified'],
                'ignored': plan['ignored'], 'protected': plan['protected']}

    def unsafe_source(self):
        before = tree(self.backups)
        self.db.chmod(0o640)
        try:
            out, err = self.wrapper('unsafe_source_rejected', expected=1)
        finally:
            self.db.chmod(0o600)
        require(not out and err == b'Accord backup failed: unsafe_regular_file\n', 'unsafe_source_fixed_failure')
        require(tree(self.backups) == before, 'failure_preserves_all_backups')
        return {'reason': 'unsafe_regular_file', 'backups_unchanged': True}

    def restore(self):
        directory = self.work / 'restore'
        directory.mkdir(mode=0o700)
        restored = directory / 'board.db'
        shutil.copyfile(self.second_copy, restored)
        restored.chmod(0o600)
        require(database(restored) == self.second_image, 'restore_copy_exact')
        before = tree(self.backups)
        self.cli('restore_revoke_all', 'token', 'revoke-all', '--db', restored, '--reason', 'restore')
        self.cli('restore_maintenance_reset', 'maintenance', 'reset', '--db', restored, '--reason', 'restore')
        image = database(restored)
        require(len(image[2]['tokens']) == 2 and all(row[-1] == 1 for row in image[2]['tokens']), 'restore_tokens_revoked')
        events = [json.loads(row[1]) for row in image[2]['events']]
        types = [event['type'] for event in events]
        require('tokens_revoked_all' in types and 'database_recovered' in types, 'restore_audit_events')
        require(tree(self.backups) == before, 'restore_original_snapshots_unchanged')
        require(database(self.db) == self.second_image, 'restore_source_unchanged')
        return {'schema': image[0], 'tokens_revoked': 2, 'audit_types_checked': ['tokens_revoked_all', 'database_recovered']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--backup-wrapper', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True, help='new JSON in an existing owned0700 evidence directory')
    parser.add_argument('--timeout', type=float, default=60, help='per child seconds; independent of future cgroup limit')
    args = parser.parse_args()
    args.binary, args.backup_wrapper = args.binary.resolve(strict=True), args.backup_wrapper.resolve(strict=True)
    args.output = args.output.absolute()
    parent = args.output.parent.stat()
    require(parent.st_uid == os.getuid() and stat.S_IMODE(parent.st_mode) == 0o700, 'output_parent_private0700')
    require(0 < args.timeout <= 300, 'bounded_timeout')
    report = {'suite': 'backup-process-068-v1', 'status': 'RUNNING', 'cases': [], 'processes': [],
              'environment': {'os': platform.system(), 'release': platform.release(), 'arch': platform.machine(),
                              'python': platform.python_version(), 'sqlite': sqlite3.sqlite_version},
              'hashes_before': {'binary': sha(args.binary), 'wrapper': sha(args.backup_wrapper),
                                'harness': sha(Path(__file__)), 'support': sha(Path(__file__).with_name('backup_process_support.py'))},
              'limits': {'per_child_seconds': args.timeout, 'cgroup': 'NOT-RUN', 'annual_fixture': 'NOT-RUN', 'two_physical_pcs': 'NOT-RUN'}}
    evidence = Evidence(args.output, args.timeout, report)
    started, work = time.monotonic(), None
    try:
        work = Path(tempfile.mkdtemp(prefix='backup068-', dir=args.output.parent))
        acceptance = Acceptance(args, work, evidence)
        for case_id, action in [('B068-01', acceptance.bootstrap), ('B068-02', acceptance.second),
                                ('B068-03', acceptance.history), ('B068-04', acceptance.prune),
                                ('B068-05', acceptance.unsafe_source), ('B068-06', acceptance.restore)]:
            evidence.case(case_id, action)
        report['status'] = 'PASS'
    except BaseException as error:
        report.update(status='FAIL', error_type=type(error).__name__)
    finally:
        try:
            report['hashes_after'] = {'binary': sha(args.binary), 'wrapper': sha(args.backup_wrapper)}
            require(all(report['hashes_before'][key] == value for key, value in report['hashes_after'].items()), 'readonly_inputs_unchanged')
            if work is not None:
                shutil.rmtree(work)  # Only this run's fresh mkdtemp tree, after bounded children exit.
        except BaseException as error:
            report.update(status='FAIL', finalization_error_type=type(error).__name__)
        report['workspace_removed'] = work is not None and not work.exists()
        report['seconds'] = round(time.monotonic() - started, 6)
        evidence.save()
    print(json.dumps({'status': report['status'], 'counts': report['counts'], 'seconds': report['seconds']}))
    return 0 if report['status'] == 'PASS' else 1


if __name__ == '__main__':
    sys.exit(main())
