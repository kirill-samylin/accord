"""Strict historical migration snapshots; preserve bytes without opening them."""
import re

PATTERN = re.compile(r'schema[12]-[0-9a-f]{64}\.db(?:\.sha256)?')


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


def copy_migrations(source, target, legacy, copy_file):
    if not source.exists() and not source.is_symlink():
        return
    files = validate_migrations(source, legacy)
    target.mkdir(mode=0o700)
    for path in files:
        copy_file(path, target / path.name)
    legacy.sync_dir(target)
