"""Schema6 authority validation units; SQLite in memory, no child or host operations."""
import json
from pathlib import Path
import sqlite3
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'infra/accord/projects'))
from automation_schema import INDEXES, TABLES, validate_automation
from restore_authority import verify_automation_reset


def connection(version=6):
    db = sqlite3.connect(':memory:')
    db.execute('PRAGMA user_version=' + str(version))
    for table in TABLES:
        db.execute('CREATE TABLE ' + table + '(key TEXT PRIMARY KEY,data TEXT NOT NULL)')
    db.executescript('CREATE TABLE automation_metadata(id INTEGER PRIMARY KEY, version); INSERT INTO automation_metadata VALUES(1,1)')
    for sql in INDEXES.values():
        db.execute(sql)
    return db


def put(db, table, data):
    db.execute('INSERT INTO ' + table + ' VALUES(?,?)', ('fixture', json.dumps(data)))


class AutomationSchema(unittest.TestCase):
    def test_declared6_requires_every_table_metadata_and_unique_execution_constraint(self):
        for damage in [*(('DROP TABLE ' + table) for table in TABLES),
                       'DELETE FROM automation_metadata',
                       'UPDATE automation_metadata SET version=2',
                       'UPDATE automation_metadata SET id=2',
                       'DROP INDEX automation_cap_attempt',
                       'DROP INDEX automation_step_execution; CREATE INDEX automation_step_execution ON workflow_steps(key)',
                       "DROP INDEX automation_step_execution; " + INDEXES['automation_step_execution'].replace('$.workflow_id', '$.WORKFLOW_ID'),
                       "DROP INDEX automation_step_execution; " + INDEXES['automation_step_execution'].replace('$.workflow_id', '$.workflow_ id')]:
            with self.subTest(damage=damage):
                db = connection()
                self.addCleanup(db.close)
                db.executescript(damage)
                with self.assertRaises((RuntimeError, sqlite3.Error)):
                    validate_automation(db)
        db = connection()
        self.addCleanup(db.close)
        validate_automation(db)

    def fixture(self):
        old, new = connection(), connection()
        self.addCleanup(old.close)
        self.addCleanup(new.close)
        records = {
            'automation_policies': {'state': 'active', 'version': 1, 'updated_at': 'before'},
            'automation_policy_revisions': {'state': 'active', 'version': 1},
            'workflows': {'state': 'running', 'version': 3, 'control_epoch': 2,
                          'grant': {'policy_id': 'fixture'}, 'updated_at': 'before'},
            'workflow_reservations': {'workflow_id': 'fixture', 'workspace': 'fixture'},
            'execution_capabilities': {'run_id': 'fixture', 'generation': 1, 'state': 'active', 'digest': 'history'},
            'workflow_steps': {'workflow_id': 'fixture', 'sequence': 1, 'attempt_no': 1},
            'workflow_plans': {'revision': 1}, 'workflow_questions': {'question': 'history'},
            'workflow_attestations': {'proof': 'history'},
        }
        for table, record in records.items():
            put(old, table, record)
            changed = dict(record)
            if table == 'automation_policies':
                changed.update(state='revoked', version=2, updated_at='after')
            if table == 'workflows':
                changed.update(state='needs_recovery', version=4, control_epoch=3,
                               reason_code='restore_authority_invalidated', updated_at='after')
            if table == 'execution_capabilities':
                changed['state'] = 'revoked'
            put(new, table, changed)
        return old, new

    def test_reset_preserves_history_unknown_reservation_and_invalidates_authority(self):
        old, new = self.fixture()
        verify_automation_reset(old, new)

    def test_boolean_reset_ack_does_not_substitute_authority_or_history_proof(self):
        changes = ["DELETE FROM workflow_reservations", "DELETE FROM workflow_steps",
                   "UPDATE automation_policies SET data=json_set(data,'$.state','active')",
                   "UPDATE workflows SET data=json_set(data,'$.state','queued')",
                   "UPDATE workflows SET data=json_set(data,'$.control_epoch',2)",
                   "UPDATE execution_capabilities SET data=json_set(data,'$.state','active')",
                   "UPDATE workflow_attestations SET data='{}'"]
        for sql in changes:
            with self.subTest(sql=sql):
                old, new = self.fixture()
                new.execute(sql)
                with self.assertRaises(RuntimeError):
                    verify_automation_reset(old, new)

    def test_historical_restore_cannot_synthesize_authority(self):
        old, new = connection(5), connection()
        self.addCleanup(old.close)
        self.addCleanup(new.close)
        verify_automation_reset(old, new)
        put(new, 'automation_policies', {'state': 'active'})
        with self.assertRaises(RuntimeError):
            verify_automation_reset(old, new)

from contextlib import closing
import hashlib
from test_projects_snapshot import ProjectsFixture
from layout import LEGACY
from migration_files import validate_migrations


def upgrade_fixture(path, version):
    with closing(sqlite3.connect(path)) as db, db:
        db.executescript('''DROP TABLE tasks;
            CREATE TABLE tasks(key,data); CREATE TABLE metadata(id,sequence);
            INSERT INTO metadata VALUES(1,1); CREATE TABLE sessions(key,data);
            CREATE TABLE role_generations(role,generation); CREATE TABLE events(cursor,data);
            CREATE TABLE idempotency(key,hash,response); CREATE TABLE tokens(id,hash,owner,machine,expires,revoked);
            CREATE TABLE notes(key,data); CREATE TABLE note_metadata(id,sequence);
            INSERT INTO note_metadata VALUES(1,1); CREATE TABLE agent_profiles(key,data);
            CREATE TABLE agent_profile_revisions(profile_id,version,data);''')
        if version >= 4:
            db.executescript('CREATE TABLE coordination_metadata(id,version); INSERT INTO coordination_metadata VALUES(1,1)')
        if version >= 5:
            db.executescript('CREATE TABLE managed_metadata(id,version); INSERT INTO managed_metadata VALUES(1,1)')
            for table in ('managed_hosts', 'managed_runs', 'managed_commands'):
                db.execute('CREATE TABLE ' + table + '(key,data)')
        if version >= 6:
            for table in TABLES:
                db.execute('CREATE TABLE ' + table + '(key TEXT PRIMARY KEY,data TEXT NOT NULL)')
            db.executescript('CREATE TABLE automation_metadata(id,version); INSERT INTO automation_metadata VALUES(1,1)')
            for sql in INDEXES.values():
                db.execute(sql)
            put(db, 'workflows', {'state': 'running', 'control_epoch': 1})
            put(db, 'workflow_reservations', {'workflow_id': 'fixture'})
        db.execute('PRAGMA user_version=' + str(version))


class AutomationSnapshot(ProjectsFixture, unittest.TestCase):
    def test_format2_accepts_historical_and6_with_all_authority_history_bytes(self):
        upgrade_fixture(self.state / 'board.db', 6)
        folder = self.state / 'board.db.migrations'
        folder.mkdir(mode=0o700)
        name = 'schema5-' + 'a' * 64 + '.db'
        old = folder / name
        old.write_bytes(b'historical immutable snapshot fixture')
        old.chmod(0o600)
        checksum = folder / (name + '.sha256')
        checksum.write_text(hashlib.sha256(old.read_bytes()).hexdigest() + '  ' + name + '\n')
        checksum.chmod(0o600)
        saved = self.snapshot()
        self.assertEqual((saved / 'default/board.db.migrations' / name).read_bytes(), old.read_bytes())
        with closing(sqlite3.connect(saved / 'default/board.db')) as db:
            self.assertEqual(db.execute('SELECT data FROM workflows').fetchone()[0], '{"state": "running", "control_epoch": 1}')
            self.assertEqual(db.execute('SELECT count(*) FROM workflow_reservations').fetchone()[0], 1)
        validate_migrations(saved / 'default/board.db.migrations', LEGACY)

    def test_incomplete6_is_never_published(self):
        upgrade_fixture(self.state / 'board.db', 6)
        with closing(sqlite3.connect(self.state / 'board.db')) as db, db:
            db.execute('DROP INDEX automation_cap_attempt')
        with self.assertRaises(RuntimeError):
            self.snapshot()
        self.assertEqual(list(self.copies.iterdir()), [])

    def test_historical3_4_5_continue_to_snapshot(self):
        for version in (3, 4, 5):
            with self.subTest(version=version):
                # Each subtest owns a fresh private fixture, not a downgrade of schema6.
                fixture = AutomationSnapshot('test_incomplete6_is_never_published')
                fixture.setUp()
                try:
                    upgrade_fixture(fixture.state / 'board.db', version)
                    saved = fixture.snapshot()
                    self.assertTrue((saved / 'default/board.db').is_file())
                finally:
                    fixture.doCleanups()

from restore_authority import verify_managed_reset, verify_registry_reset


class ResetAuthority(unittest.TestCase):
    def registry(self, target, reset):
        db = sqlite3.connect(':memory:')
        self.addCleanup(db.close)
        db.executescript('''CREATE TABLE registry_meta(id,default_id,signing_key,default_db,default_web);
            CREATE TABLE credentials(revoked); CREATE TABLE projects(id,state);
            CREATE TABLE memberships(project_id,identity_id,role,local_id,active,version,access_epoch);
            CREATE TABLE sessions(id); CREATE TABLE preauth(id); CREATE TABLE actions(id);
            CREATE TABLE session_epochs(id); INSERT INTO projects VALUES('fixture','active');''')
        db.execute('INSERT INTO registry_meta VALUES(1,?,?,?,?)',
                   ('fixture', 'new' if reset else 'old', str(target / 'board.db'), str(target / 'web')))
        db.execute('INSERT INTO credentials VALUES(?)', (int(reset),))
        db.execute('INSERT INTO memberships VALUES(?,?,?,?,?,?,?)', ('fixture', 'who', 'editor', 'local', 1, 1, 2 if reset else 1))
        return db

    def test_registry_requires_new_signing_key_epoch_and_no_live_credentials(self):
        target = Path('/fixture/restored')
        for change in (None, "UPDATE registry_meta SET signing_key='old'",
                       'UPDATE memberships SET access_epoch=1',
                       'UPDATE credentials SET revoked=0', 'INSERT INTO session_epochs VALUES(1)'):
            with self.subTest(change=change):
                before, after = self.registry(target, False), self.registry(target, True)
                if change:
                    after.execute(change)
                    with self.assertRaises(RuntimeError):
                        verify_registry_reset(before, after, target)
                else:
                    verify_registry_reset(before, after, target)

    def test_managed_unknown_exit_keeps_recovery_and_advances_owner_epoch(self):
        for change in (None, "UPDATE managed_hosts SET data=json_set(data,'$.host.worker_epoch',1)",
                       "UPDATE managed_runs SET data=json_set(data,'$.state','completed')",
                       "UPDATE managed_commands SET data=json_set(data,'$.command.status','queued')"):
            with self.subTest(change=change):
                before, after = sqlite3.connect(':memory:'), sqlite3.connect(':memory:')
                self.addCleanup(before.close)
                self.addCleanup(after.close)
                for db in (before, after):
                    db.execute('PRAGMA user_version=6')
                    for table in ('managed_hosts', 'managed_runs', 'managed_commands'):
                        db.execute('CREATE TABLE ' + table + '(key,data)')
                put(before, 'managed_hosts', {'host': {'state': 'active', 'worker_epoch': 1, 'version': 1}, 'secret_hash': 'old', 'worker_id': 'old'})
                put(after, 'managed_hosts', {'host': {'state': 'disabled', 'worker_epoch': 2, 'version': 2}, 'secret_hash': '', 'worker_id': ''})
                put(before, 'managed_runs', {'state': 'running'})
                put(after, 'managed_runs', {'state': 'needs_recovery'})
                put(before, 'managed_commands', {'command': {'status': 'accepted'}})
                put(after, 'managed_commands', {'command': {'status': 'cancelled'}})
                if change:
                    after.execute(change)
                    with self.assertRaises(RuntimeError):
                        verify_managed_reset(before, after)
                else:
                    verify_managed_reset(before, after)
