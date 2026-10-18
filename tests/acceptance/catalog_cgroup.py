"""Observe a server PID's cgroup-v2 leaf files, never infer effective quotas."""
import pathlib
import re


def absolute_path(value):
    """Reject traversal before PurePath can silently normalize proc input."""
    if not value.startswith('/') or '\x00' in value:
        raise ValueError('Non-absolute or invalid cgroup path')
    if value != '/' and any(part in ('', '.', '..') for part in value[1:].split('/')):
        raise ValueError('Non-canonical cgroup path')
    return pathlib.PurePosixPath(value)


def mount_path(value):
    # mountinfo escapes whitespace and backslash; cgroup membership is raw.
    escapes = {'040': ' ', '011': '\t', '012': '\n', '134': '\\'}

    def decode(match):
        code = match.group(1)
        if code not in escapes:
            raise ValueError('Invalid mountinfo path escape')
        return escapes[code]

    decoded = re.sub(r'\\([0-7]{3})', decode, value)
    # A literal backslash is valid only when encoded as \134 in mountinfo.
    if '\\' in re.sub(r'\\(?:040|011|012|134)', '', value):
        raise ValueError('Invalid mountinfo path escape')
    return absolute_path(decoded)


def membership(text):
    entries = []
    for line in text.split('\n'):
        if not line:
            continue
        fields = line.split(':', 2)
        if len(fields) != 3 or not decimal(fields[0]):
            raise ValueError('Malformed cgroup membership record')
        if fields[:2] == ['0', '']:
            entries.append(fields[2])
    if len(entries) != 1 or entries[0].endswith(' (deleted)'):
        raise ValueError('Unified membership missing, ambiguous or deleted')
    return absolute_path(entries[0])


def locate_leaf(path, mountinfo):
    """Map namespace-relative membership through one unambiguous cgroup2 mount."""
    matches = []
    for line in mountinfo.splitlines():
        before, separator, after = line.partition(' - ')
        filesystem = after.split()
        if not separator or not filesystem or filesystem[0] != 'cgroup2':
            continue
        fields = before.split()
        if len(fields) < 6 or len(filesystem) < 3:
            raise ValueError('Malformed cgroup2 mountinfo')
        root, point = mount_path(fields[3]), mount_path(fields[4])
        if path.is_relative_to(root):
            matches.append((root, point, point / path.relative_to(root)))
    if len(matches) != 1:
        raise ValueError('No unique matching cgroup2 mount')
    root, point, leaf = matches[0]
    resolved_point = pathlib.Path(point).resolve(strict=True)
    resolved_leaf = pathlib.Path(leaf).resolve(strict=True)
    if not resolved_leaf.is_relative_to(resolved_point):
        raise ValueError('Cgroup path escapes mountpoint')
    return root, point, resolved_leaf


def decimal(value):
    return value.isascii() and value.isdigit()


def read_limit(leaf, name):
    try:
        value = (leaf / name).read_text().strip()
        if name == 'cpu.max':
            parts = value.split()
            valid = (len(parts) == 2 and decimal(parts[1]) and int(parts[1]) > 0
                     and (parts[0] == 'max' or (decimal(parts[0]) and int(parts[0]) > 0)))
        else:
            valid = value == 'max' or decimal(value)
        if not valid:
            return {'status': 'unavailable', 'reason': 'Invalid controller value'}
        return {'status': 'available', 'value': value}
    except (OSError, UnicodeError) as error:
        return {'status': 'unavailable', 'reason': type(error).__name__}


def cgroup_limits(pid, proc_root=pathlib.Path('/proc')):
    """Snapshot child membership using observer mounts; ancestors are not sampled.

    proc PID cgroup paths and self mountinfo are interpreted in the observer's
    cgroup/mount namespaces. Hidden ancestors, cpusets and competing processes
    prevent these leaf values from proving the PID's effective resource budget.
    No fallback to the hierarchy root or observer's own membership is allowed.
    """
    member_file = proc_root / str(pid) / 'cgroup'
    mount_file = proc_root / 'self' / 'mountinfo'
    result = {
        'status': 'unavailable', 'pid': pid,
        'membership_source': str(member_file), 'mountinfo_source': str(mount_file),
        'scope': 'Server PID leaf cgroup-v2 files, sampled after measured requests; '
                 'paths relative to observer namespaces; no ancestor sampling.',
        'effective_limits': {'status': 'unavailable',
                             'reason': 'Ancestor quotas and hidden namespace ancestors are not evaluated; '
                                       'leaf max does not prove unlimited effective resources.'},
    }
    try:
        path = membership(member_file.read_text())
        result['membership_path'] = str(path)
        root, point, leaf = locate_leaf(path, mount_file.read_text())
        result.update(mount_root=str(root), mount_point=str(point), leaf_path=str(leaf))
        limits = {name: read_limit(leaf, name) for name in ('cpu.max', 'memory.max')}
        if membership(member_file.read_text()) != path:
            raise ValueError('Server cgroup membership changed during snapshot')
        result['leaf_limits'] = limits
        available = sum(value['status'] == 'available' for value in limits.values())
        result['status'] = ('unavailable', 'partial', 'available')[available]
    except (OSError, UnicodeError, ValueError) as error:
        result['reason'] = str(error) if isinstance(error, ValueError) else type(error).__name__
    return result
