"""Registry schema1->2 capable update units: synthetic migration, strict preservation."""
from contextlib import closing
import hashlib
import json
import sqlite3
import unittest
from unittest.mock import patch

from test_projects_automation import upgrade_fixture
from test_projects_snapshot import ProjectsFixture, OTHER, REGISTRY2_DDL, registry1_copy, registry2
import upgrade_storage
from layout import LEGACY, inventory
from migration_files import validate_registry_migrations
from validate import validate


class RegistryUpgrade(ProjectsFixture, unittest.TestCase):
    """migrate_storage against the fixture tree, with the Go step replaced by SQL."""

    def setUp(self):
        super().setUp()
        for folder in (self.state, self.projects / 'projects' / OTHER):
            upgrade_fixture(folder / 'board.db', 6)
        self.saved = self.snapshot()
        self.manifest = validate(self.saved)
        self.assertEqual(self.manifest['registry_schema'], 1)
        self.binary = self.root / 'capable-binary'

    def migrate(self, native):
        def run(*args):
            self.assertEqual(args[:4], ('runuser', '-u', 'accord', '--'))
            self.assertEqual(args[4:7], (str(self.binary), 'projects', 'migrate-storage'))
            self.assertEqual(args[args.index('--projects-data') + 1], str(self.projects))
            native()
            return json.dumps({'storage_migrated': True, 'boards': 2})

        with patch.object(upgrade_storage, 'PROJECTS', self.projects), \
                patch.object(upgrade_storage, 'STATE', self.state), \
                patch.object(upgrade_storage.host, 'run', side_effect=run):
            return upgrade_storage.migrate_storage(self.binary, self.saved, self.manifest)

    def test_registry1_to2_with_pre_ddl_copy_preserves_every_old_row(self):
        result = self.migrate(lambda: registry2(self.projects))
        self.assertEqual(result['registry_schema'], 2)
        self.assertTrue(result['logical_rows_preserved'])
        self.assertEqual(result['databases_preserved'], 5)
        catalog = inventory(self.projects, self.state / 'board.db', self.state / 'web')
        self.assertEqual(catalog['registry_schema'], 2)
        copies = validate_registry_migrations(self.projects / 'registry.sqlite.migrations', LEGACY)
        self.assertEqual(len(copies), 1)
        self.assertEqual(copies[0].stat().st_mode & 0o777, 0o600)
        with closing(sqlite3.connect(copies[0])) as historical:
            self.assertEqual(historical.execute('PRAGMA user_version').fetchone()[0], 1)
            self.assertEqual(historical.execute("SELECT count(*) FROM sqlite_master WHERE name='people'").fetchone()[0], 0)
        # The post-update snapshot sees registry2 beside the schema1 history.
        later = validate(self.snapshot())
        self.assertEqual(later['registry_schema'], 2)
        self.assertEqual(len([name for name in later['files'] if 'registry.sqlite.migrations/' in name]), 1)

    def test_sidecar_named_copy_is_also_accepted(self):
        self.migrate(lambda: registry2(self.projects, sidecar=True))
        validate(self.snapshot())

    def test_migration_without_copy_or_without_ddl_is_not_confirmed(self):
        with self.assertRaisesRegex(RuntimeError, 'pre-migration copy'):
            self.migrate(lambda: registry2(self.projects, copy=False))
        for native, message in ((lambda: registry1_copy(self.projects), 'schema/integrity'),
                                (lambda: None, 'schema/integrity')):
            with self.subTest(message=message):
                fixture = RegistryUpgrade('test_sidecar_named_copy_is_also_accepted')
                fixture.setUp()
                try:
                    with self.assertRaisesRegex(RuntimeError, message):
                        fixture.migrate(native)
                finally:
                    fixture.doCleanups()

    def test_copy_must_match_the_immutable_pre_update_snapshot(self):
        def native():
            with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as db, db:
                db.execute("INSERT INTO identities VALUES('i_late','token','t3','mallory','x')")
            registry2(self.projects)
        with self.assertRaisesRegex(RuntimeError, 'pre-migration copy'):
            self.migrate(native)

    def test_old_columns_changed_during_ddl_are_rejected(self):
        def native():
            registry2(self.projects)
            with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as db, db:
                db.execute("UPDATE identities SET owner='renamed' WHERE id='i_one'")
        with self.assertRaisesRegex(RuntimeError, 'changed existing logical rows'):
            self.migrate(native)

    def test_incomplete_registry2_ddl_is_rejected(self):
        def native():
            registry1_copy(self.projects)
            with closing(sqlite3.connect(self.projects / 'registry.sqlite')) as db, db:
                db.executescript(REGISTRY2_DDL.replace('CREATE INDEX IF NOT EXISTS registry_identity_person ON identities(person_id);', ''))
        with self.assertRaisesRegex(RuntimeError, 'people index'):
            self.migrate(native)
        with self.assertRaises(RuntimeError):
            self.snapshot()

    def test_unbound_identity_after_bootstrap_is_rejected(self):
        with self.assertRaisesRegex(RuntimeError, 'without a person'):
            self.migrate(lambda: registry2(self.projects, bootstrap=False))

    def test_historical_board_copies_are_still_checked_through_registry_prefix(self):
        history = self.state / 'board.db.migrations'
        history.mkdir(mode=0o700)
        old = history / ('schema5-' + 'a' * 64 + '.db')
        old.write_bytes(b'historical board evidence')
        old.chmod(0o600)
        checksum = old.with_name(old.name + '.sha256')
        checksum.write_text(hashlib.sha256(old.read_bytes()).hexdigest() + '  ' + old.name + '\n')
        checksum.chmod(0o600)
        copy = registry1_copy(self.projects)
        self.saved = self.snapshot()
        self.manifest = validate(self.saved)
        self.migrate(lambda: registry2(self.projects, copy=False))
        # A second update must notice a registry copy rewritten since the snapshot.
        fresh = self.snapshot()
        manifest = validate(fresh)
        self.assertIn('registry/registry.sqlite.migrations/' + copy.name, manifest['files'])

    def test_facts_projection_ignores_only_appended_columns(self):
        before = self.saved / 'registry/registry.sqlite'
        registry2(self.projects)
        after = self.projects / 'registry.sqlite'
        self.assertNotEqual(upgrade_storage.facts(before)['identities'], upgrade_storage.facts(after)['identities'])
        projected = upgrade_storage.facts(after, upgrade_storage.columns(before))
        self.assertEqual(projected['identities'], upgrade_storage.facts(before)['identities'])
        self.assertNotIn('people', projected)
