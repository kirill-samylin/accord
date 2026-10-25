"""Strict historical migration snapshots; preserve bytes without opening them."""
import re

PATTERN = re.compile(r'schema[12345]-[0-9a-f]{64}\.db(?:\.sha256)?')
# Registry copies written before the 1->2 DDL. The 64hex is either a content
# checksum in the name or an opaque identifier with a board-style sidecar.
REGISTRY_PATTERN = re.compile(r'registry1-[0-9a-f]{64}\.db(?:\.sha256)?')
REGISTRY_MIGRATIONS = 'registry.sqlite.migrations'


def validate_migrations(folder, legacy):
    legacy.private(folder, True)
    files = sorted(folder.iterdir())
    names = {path.name for path in files}
    for path in files:
        legacy.private(path)
        if not PATTERN.fullmatch(path.name):
            raise RuntimeError('unknown migration backup entry')
        database = path.name.removesuffix('.sha256')
        if database not in names or database + '.sha256' not in names:
            raise RuntimeError('incomplete migration backup requires inspection')
        if path.name.endswith('.sha256'):
            expected = legacy.sha(folder / database) + '  ' + database + '\n'
            if path.read_text() != expected:
                raise RuntimeError('migration backup checksum differs')
    return files


def validate_registry_migrations(folder, legacy):
    """Every registry1 copy must prove its bytes: sidecar if present, else its name."""
    legacy.private(folder, True)
    files = sorted(folder.iterdir())
    names = {path.name for path in files}
    for path in files:
        legacy.private(path)
        if not REGISTRY_PATTERN.fullmatch(path.name):
            raise RuntimeError('unknown registry migration backup entry')
        database = path.name.removesuffix('.sha256')
        if database not in names:
            raise RuntimeError('incomplete registry migration backup requires inspection')
        if path.name.endswith('.sha256'):
            expected = legacy.sha(folder / database) + '  ' + database + '\n'
            if path.read_text() != expected:
                raise RuntimeError('registry migration backup checksum differs')
        elif database + '.sha256' not in names:
            if legacy.sha(path) != database[len('registry1-'):-len('.db')]:
                raise RuntimeError('registry migration backup checksum differs')
    return files


def copy_migrations(source, target, legacy, copy_file, validate=validate_migrations):
    if not source.exists() and not source.is_symlink():
        return
    files = validate(source, legacy)
    target.mkdir(mode=0o700)
    for path in files:
        copy_file(path, target / path.name)
    legacy.sync_dir(target)
