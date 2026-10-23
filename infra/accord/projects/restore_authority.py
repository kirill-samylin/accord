"""Verify reset effects, not a command's exit code, before marking offline restore."""
from contextlib import closing
import json

from automation_schema import TABLES, validate_automation
from layout import readonly
from validate import board


def entities(connection, table):
    return {key: json.loads(data) for key, data in connection.execute('SELECT key,data FROM ' + table)}


def verify_automation_reset(before, after):
    validate_automation(after)
    source_version = before.execute('PRAGMA user_version').fetchone()[0]
    if source_version < 6:
        if any(after.execute('SELECT count(*) FROM ' + table).fetchone()[0] for table in TABLES):
            raise RuntimeError('restore synthesized automation authority or history')
        return
    for table in TABLES:
        old, new = entities(before, table), entities(after, table)
        if old.keys() != new.keys():
            raise RuntimeError('restore changed automation history inventory')
        for key, previous in old.items():
            current = new[key]
            expected = dict(previous)
            if table == 'automation_policies':
                expected.update(state='revoked', version=previous['version'] + 1,
                                updated_at=current.get('updated_at'))
            elif table == 'workflows' and previous['state'] not in ('completed', 'cancelled'):
                expected.update(state='needs_recovery', reason_code='restore_authority_invalidated',
                                control_epoch=previous['control_epoch'] + 1,
                                version=previous['version'] + 1, updated_at=current.get('updated_at'))
            elif table == 'execution_capabilities':
                expected['state'] = 'revoked'
            if current != expected:
                raise RuntimeError('restore lost automation history or retained execution authority')
    # Reservations survive even without a lease: elapsed time is no exit proof.


def verify_registry_reset(before, after, target):
    old_meta = before.execute('SELECT default_id,signing_key FROM registry_meta WHERE id=1').fetchone()
    new_meta = after.execute('SELECT default_id,signing_key,default_db,default_web FROM registry_meta WHERE id=1').fetchone()
    if (not old_meta or not new_meta or old_meta[0] != new_meta[0] or
            old_meta[1] == new_meta[1] or not new_meta[1] or
            new_meta[2:] != (str(target / 'board.db'), str(target / 'web'))):
        raise RuntimeError('restore retained signing authority or changed default identity')
    old_members = before.execute("SELECT project_id,identity_id,role,local_id,active,version,access_epoch FROM memberships WHERE project_id IN (SELECT id FROM projects WHERE state!='provisioning') ORDER BY project_id,identity_id").fetchall()
    new_members = after.execute('SELECT project_id,identity_id,role,local_id,active,version,access_epoch FROM memberships ORDER BY project_id,identity_id').fetchall()
    if new_members != [row[:-1] + (row[-1] + 1,) for row in old_members]:
        raise RuntimeError('restore changed membership history or retained ACL epochs')
    if after.execute('SELECT count(*) FROM credentials WHERE revoked=0').fetchone()[0]:
        raise RuntimeError('restore retained registry credentials')
    for table in ('sessions', 'preauth', 'actions', 'session_epochs'):
        if after.execute('SELECT count(*) FROM ' + table).fetchone()[0]:
            raise RuntimeError('restore retained registry session authority')


def verify_managed_reset(before, after):
    source_version = before.execute('PRAGMA user_version').fetchone()[0]
    for table in ('managed_hosts', 'managed_runs', 'managed_commands'):
        old = entities(before, table) if source_version >= 5 else {}
        new = entities(after, table)
        if old.keys() != new.keys():
            raise RuntimeError('restore changed managed history inventory')
        for key, record in new.items():
            previous = old[key]
            if table == 'managed_hosts':
                host, old_host = record['host'], previous['host']
                if (host['state'] != 'disabled' or record.get('secret_hash') or record.get('worker_id') or
                        host['worker_epoch'] != old_host['worker_epoch'] + 1 or
                        host['version'] != old_host['version'] + 1):
                    raise RuntimeError('restore retained managed host authority')
            elif table == 'managed_commands':
                expected = previous['command']['status']
                if expected in ('queued', 'accepted'):
                    expected = 'cancelled'
                if record['command']['status'] != expected:
                    raise RuntimeError('restore retained executable managed command')
            elif previous['state'] in ('queued', 'starting', 'running', 'stopping', 'needs_recovery'):
                if record['state'] != 'needs_recovery':
                    raise RuntimeError('restore forged managed process exit')


def verify_restored(source, target, manifest):
    pairs = [(source / 'default', target)]
    pairs += [(source / 'projects' / item['id'],
               target / 'projects-data/projects' / item['id'])
              for item in manifest['projects'] if item['state'] != 'provisioning']
    for old, new in pairs:
        with closing(readonly(new / 'web/web.db')) as web:
            web_version = web.execute('PRAGMA user_version').fetchone()[0]
        board(new, {'board_schema': 6, 'web_schema': web_version}, live=True)
        with closing(readonly(old / 'board.db')) as before, closing(readonly(new / 'board.db')) as after:
            verify_automation_reset(before, after)
            if after.execute('SELECT count(*) FROM tokens WHERE revoked=0').fetchone()[0]:
                raise RuntimeError('restore retained board credentials')
            verify_managed_reset(before, after)
    with closing(readonly(source / 'registry/registry.sqlite')) as before, closing(readonly(target / 'projects-data/registry.sqlite')) as after:
        verify_registry_reset(before, after, target)
