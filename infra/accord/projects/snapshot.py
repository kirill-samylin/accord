"""Sole-writer offline snapshots; pending recovery state is preserved, never served."""
import datetime
import json
import os
import re
import shutil
import uuid

from layout import LEGACY, board_copy, inventory, pending_copy, registry_copy
from validate import validate

NAME = re.compile(r'projects-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}')


def snapshot(default_db, default_web, projects_root, backups):
    LEGACY.private(backups, True)
    catalog = inventory(projects_root, default_db, default_web)
    required = LEGACY.private(default_db) + sum(
        LEGACY.private(path) for source in (default_web, projects_root)
        for path in source.rglob('*') if not path.is_dir())
    if shutil.disk_usage(backups).free < required + (16 << 20) + LEGACY.FLOOR:
        raise RuntimeError('insufficient all-project snapshot disk reserve')
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    name = 'projects-' + stamp + '-' + uuid.uuid4().hex
    pending, published = backups / ('.pending-' + name), backups / name
    pending.mkdir(mode=0o700)
    try:
        schemas = board_copy(default_db, default_web, pending / 'default')
        (pending / 'registry').mkdir(mode=0o700)
        registry_schema = registry_copy(projects_root, pending / 'registry')
        (pending / 'projects').mkdir(mode=0o700)
        projects = []
        for entry in catalog['projects']:
            target = pending / 'projects' / entry['id']
            record = {'id': entry['id'], 'state': entry['state']}
            if entry['state'] == 'provisioning':
                pending_copy(entry['folder'], target)
            else:
                record.update(board_copy(entry['folder'] / 'board.db', entry['folder'] / 'web', target))
            projects.append(record)
        quarantined = []
        if catalog['quarantine']:
            (pending / 'quarantine').mkdir(mode=0o700)
            for source in catalog['quarantine']:
                pending_copy(source, pending / 'quarantine' / source.name)
                quarantined.append(source.name)
            LEGACY.sync_dir(pending / 'quarantine')
        manifest = {'format': 2, 'registry_schema': registry_schema, 'default_id': catalog['default_id'],
                    'default': schemas, 'projects': projects, 'restore_requires_auth_reset': True,
                    'quarantine': quarantined,
                    'files': {path.relative_to(pending).as_posix(): LEGACY.sha(path)
                              for path in pending.rglob('*') if path.is_file()}}
        with (pending / 'manifest.json').open('x') as output:
            os.fchmod(output.fileno(), 0o600)
            json.dump(manifest, output, sort_keys=True)
            output.flush()
            os.fsync(output.fileno())
        validate(pending)
        for folder in (pending / 'projects', pending / 'registry', pending):
            LEGACY.sync_dir(folder)
        pending.rename(published)
        LEGACY.sync_dir(backups)
    except BaseException:
        shutil.rmtree(pending)
        raise
    complete = sorted((path for path in backups.iterdir() if NAME.fullmatch(path.name)), reverse=True)
    for old in complete[LEGACY.RETAIN:]:
        validate(old)
        shutil.rmtree(old)
    LEGACY.sync_dir(backups)
    return published
