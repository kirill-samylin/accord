"""Archived automation history (key prefix archive/) through snapshot, validate and restore checks."""
from contextlib import closing
import json
from pathlib import Path
import shutil
import sqlite3
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'infra/accord/projects'))
from restore_authority import verify_automation_reset, verify_managed_reset
from test_projects_automation import upgrade_fixture
from test_projects_snapshot import ProjectsFixture
from validate import validate

ARCHIVE = 'archive/'
# What the server writes when it rotates history: the final row value under an
# archive/ key; capabilities of an archived workflow are already revoked.
ROWS = {
    'workflows': [('wf_live', {'state': 'running', 'version': 3, 'control_epoch': 2, 'updated_at': 'before'}),
                  (ARCHIVE + 'wf_old', {'state': 'completed', 'version': 9, 'control_epoch': 4, 'updated_at': 'old'})],
    'workflow_steps': [(ARCHIVE + 'st_old', {'workflow_id': 'wf_old', 'sequence': 1, 'attempt_no': 1})],
    'workflow_attestations': [(ARCHIVE + 'at_old', {'workflow_id': 'wf_old', 'kind': 'run_exit'})],
    'execution_capabilities': [('ec_live', {'run_id': 'mr_live', 'generation': 1, 'state': 'active'}),
                               (ARCHIVE + 'ec_old', {'run_id': 'mr_old', 'generation': 1, 'state': 'revoked'})],
    'managed_runs': [(ARCHIVE + 'mr_old', {'state': 'stopped'})],
    'managed_commands': [(ARCHIVE + 'mc_old', {'command': {'status': 'applied'}})],
}


def archive_board(path):
    upgrade_fixture(path, 6)
    with closing(sqlite3.connect(path)) as db, db:
        db.execute('DELETE FROM workflows')
        db.execute('DELETE FROM workflow_reservations')
        for table, rows in ROWS.items():
            db.executemany('INSERT INTO ' + table + ' VALUES(?,?)', [(k, json.dumps(v)) for k, v in rows])


def server_reset(path, revoke_archive=True):
    """Model of RecoverState: live authority invalidated, archive inventory kept and
    every capability row (archive rows too) revoked."""
    with closing(sqlite3.connect(path)) as db, db:
        db.execute("UPDATE workflows SET data=json_set(data,'$.state','needs_recovery',"
                   "'$.reason_code','restore_authority_invalidated',"
                   "'$.control_epoch',json_extract(data,'$.control_epoch')+1,"
                   "'$.version',json_extract(data,'$.version')+1,'$.updated_at','after') "
                   "WHERE json_extract(data,'$.state') NOT IN ('completed','cancelled')")
        where = '' if revoke_archive else " WHERE substr(key,1,8)<>'archive/'"
        db.execute("UPDATE execution_capabilities SET data=json_set(data,'$.state','revoked')" + where)


class ArchivedHistory(ProjectsFixture, unittest.TestCase):
    def restored(self, saved, **reset):
        before = saved / 'default/board.db'
        after = self.root / ('restored-%d.db' % len(list(self.root.glob('restored-*.db'))))
        shutil.copyfile(before, after)
        server_reset(after, **reset)
        return before, after

    def test_snapshot_validate_and_restore_accept_archive_rows(self):
        archive_board(self.state / 'board.db')
        saved = self.snapshot()
        validate(saved)
        with closing(sqlite3.connect(saved / 'default/board.db')) as db:
            for table, rows in ROWS.items():
                self.assertEqual(sorted(k for k, _ in db.execute('SELECT key,data FROM ' + table)),
                                 sorted(k for k, _ in rows), table)
        before, after = self.restored(saved)
        with closing(sqlite3.connect(before)) as old, closing(sqlite3.connect(after)) as new:
            verify_automation_reset(old, new)
            verify_managed_reset(old, new)
            self.assertEqual(new.execute("SELECT count(*) FROM execution_capabilities "
                                         "WHERE json_extract(data,'$.state')<>'revoked'").fetchone()[0], 0)

    def test_restore_rejects_retained_or_dropped_archive_authority(self):
        archive_board(self.state / 'board.db')
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            # A row archived by an older server may still say active.
            db.execute("UPDATE execution_capabilities SET data=json_set(data,'$.state','active') WHERE key='archive/ec_old'")
        saved = self.snapshot()
        validate(saved)
        before, after = self.restored(saved)
        with closing(sqlite3.connect(before)) as old, closing(sqlite3.connect(after)) as new:
            verify_automation_reset(old, new)
        for damage in ({'revoke_archive': False}, 'DELETE FROM workflows WHERE key=\'archive/wf_old\'',
                       "UPDATE execution_capabilities SET key='ec_old' WHERE key='archive/ec_old'"):
            with self.subTest(damage=damage):
                if isinstance(damage, dict):
                    before, after = self.restored(saved, **damage)
                else:
                    before, after = self.restored(saved)
                    with closing(sqlite3.connect(after)) as db, db:
                        db.execute(damage)
                with closing(sqlite3.connect(before)) as old, closing(sqlite3.connect(after)) as new:
                    with self.assertRaises(RuntimeError):
                        verify_automation_reset(old, new)


if __name__ == '__main__':
    unittest.main()
