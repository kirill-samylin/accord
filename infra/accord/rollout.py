#!/usr/bin/env python3
"""Roll a built Accord release out to the server with the capable updater.

Runs on an operator machine or in CI and reaches the server only over SSH:

    upload -> stage -> apply -> smoke -> finalize [-> prune]

The server side stays the reviewed infra/accord/projects/upgrade.py: this
driver never switches release pointers or touches databases itself. Every
phase is idempotent against the server journal, so a rerun continues from the
recorded state instead of repeating a completed step. `prune` removes only old
release directories under /opt/accord; database snapshots under /var/backups
are never touched.
"""
import os
import sys

# infra/accord/token.py shadows the stdlib `token` module whenever this file's
# directory leads sys.path (plain `python3 rollout.py`); drop it first.
sys.path[:] = [entry for entry in sys.path
               if os.path.abspath(entry or os.getcwd()) != os.path.dirname(os.path.abspath(__file__))]

import argparse  # noqa: E402
import hashlib  # noqa: E402
import json  # noqa: E402
import re  # noqa: E402
import shlex  # noqa: E402
import subprocess  # noqa: E402
import tarfile  # noqa: E402
import tempfile  # noqa: E402
import time  # noqa: E402
import urllib.request  # noqa: E402
from pathlib import Path  # noqa: E402

GO_BASE = '/opt/accord'
NEXT_BASE = '/opt/accord/next'
JOURNAL = '/var/backups/accord-projects'
NGINX = '/etc/nginx/conf.d/accord-public.conf'
PHASES = ('upload', 'stage', 'apply', 'smoke', 'finalize')
# upgrade.py journal phase -> phases that this revision already passed.
DONE_AFTER = {
    'staged': ('upload', 'stage'),
    'pending-https-smoke': ('upload', 'stage', 'apply'),
    'complete': PHASES,
}
RESUMABLE = ('activation-started', 'blocked-roll-forward')
EXECUTABLES = ('agent-board', 'node')


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            digest.update(block)
    return digest.hexdigest()


def read_sums(text):
    """Parse `sha256sum` output (`<hex>  <name>`) into {name: hex}."""
    sums = {}
    for line in text.splitlines():
        if not line.strip():
            continue
        match = re.fullmatch(r'([0-9a-f]{64}) [ *](.+)', line)
        if not match:
            raise ValueError('malformed checksum line: ' + line)
        sums[match.group(2)] = match.group(1)
    return sums


def tree_files(root):
    root = Path(root)
    files = {}
    for path in sorted(root.rglob('*')):
        if path.is_symlink():
            raise ValueError('symlink in release tree: ' + str(path))
        if path.is_file():
            files[path.relative_to(root).as_posix()] = sha(path)
    return files


def load_go_release(path):
    """Validate a build-release.sh output directory; return its identity."""
    path = Path(path)
    revision = (path / 'SOURCE').read_text().strip()
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise ValueError('SOURCE must hold the full commit hash')
    sums = read_sums((path / 'SHA256SUMS').read_text())
    actual = tree_files(path)
    actual.pop('SHA256SUMS')
    if actual != sums:
        raise ValueError('Go release files differ from SHA256SUMS')
    expected = dict(sums, SHA256SUMS=sha(path / 'SHA256SUMS'))
    return {'revision': revision, 'files': expected, 'go_sha256': sums['agent-board'],
            'manifest_sha256': expected['SHA256SUMS']}


def load_next_bundle(path, revision):
    """Validate a package.py bundle for `revision`; return its identity."""
    path = Path(path)
    manifest = json.loads((path / 'manifest.json').read_text())
    if manifest.get('format') != 1 or manifest.get('revision') != revision:
        raise ValueError('Next bundle was packaged for another revision')
    actual = tree_files(path)
    actual.pop('manifest.json')
    if actual != manifest['files']:
        raise ValueError('Next bundle files differ from manifest.json')
    expected = dict(manifest['files'])
    expected['manifest.json'] = sha(path / 'manifest.json')
    return {'files': expected, 'manifest_sha256': expected['manifest.json']}


def pending_phases(requested, journal_phase, resume=False):
    """Phases of `requested` still to run given the server journal state."""
    if journal_phase is None:
        done = ()
    elif journal_phase in DONE_AFTER:
        done = DONE_AFTER[journal_phase]
    elif journal_phase in RESUMABLE and resume:
        done = ('upload', 'stage')
    elif journal_phase in RESUMABLE:
        raise RuntimeError('journal phase %r: inspect the server, then rerun with --resume' % journal_phase)
    else:
        raise RuntimeError('unknown journal phase %r requires manual inspection' % journal_phase)
    return [phase for phase in requested if phase not in done]


def plan_prune(releases, active, keep, protect=()):
    """Names to delete: all but the active, protected and `keep` newest releases."""
    others = sorted((name for name in releases if name != active and name not in protect),
                    key=lambda name: releases[name], reverse=True)
    return others[max(keep, 0):]


class Server:
    def __init__(self, target, identity=None, known_hosts=None, port=None):
        command = ['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=15',
                   '-o', 'StrictHostKeyChecking=yes', '-o', 'ServerAliveInterval=30']
        if identity:
            command += ['-i', str(identity)]
        if known_hosts:
            command += ['-o', 'UserKnownHostsFile=' + str(known_hosts)]
        if port:
            command += ['-p', str(port)]
        self.command = command + [target]

    def run(self, remote, stdin=None, timeout=900):
        # `stdin` is either bytes or an open file streamed to the remote command.
        feed = {'stdin': stdin} if hasattr(stdin, 'read') else {'input': stdin}
        done = subprocess.run(self.command + [remote], capture_output=True, timeout=timeout, **feed)
        if done.returncode:
            raise RuntimeError('remote command failed (%d): %s\n%s' % (
                done.returncode, remote[:120], done.stderr.decode(errors='replace')[-4000:]))
        return done.stdout

    def python(self, source, timeout=900, **values):
        body = 'V = ' + repr(values) + '\n' + source
        return json.loads(self.run('python3 -I -B -', body.encode(), timeout))


INVENTORY = r'''
import hashlib, json, os, pathlib, shutil
def pointer(base):
    current = pathlib.Path(base) / 'current'
    return os.readlink(current) if current.is_symlink() else None
def releases(base):
    folder = pathlib.Path(base) / 'releases'
    return {d.name: d.stat().st_mtime for d in folder.iterdir() if d.is_dir() and not d.is_symlink()}
journal = {}
for d in pathlib.Path(V['journal']).iterdir():
    state = d / 'state.json'
    if state.is_file():
        try:
            journal[d.name] = json.loads(state.read_text()).get('phase')
        except ValueError:
            journal[d.name] = 'unreadable'
print(json.dumps({'go_current': pointer(V['go']), 'next_current': pointer(V['next']),
    'go_releases': releases(V['go']), 'next_releases': releases(V['next']), 'journal': journal,
    'nginx_sha256': hashlib.sha256(pathlib.Path(V['nginx']).read_bytes()).hexdigest(),
    'free_bytes': shutil.disk_usage(V['go']).free}))
'''

PREPARE = r'''
import json, pathlib, stat
target = pathlib.Path(V['target'])
for parent in target.parents:
    info = parent.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
        raise SystemExit('unsafe release ancestry: ' + str(parent))
if target.is_symlink():
    raise SystemExit('release path is a symlink: ' + str(target))
created = not target.exists()
if created:
    target.mkdir(mode=0o755)
print(json.dumps({'created': created}))
'''

VERIFY = r'''
import hashlib, json, pathlib, stat
target = pathlib.Path(V['target']); expected = V['files']
def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            value.update(block)
    return value.hexdigest()
actual = {}
for path in target.rglob('*'):
    info = path.lstat()
    if stat.S_ISDIR(info.st_mode):
        continue
    if not stat.S_ISREG(info.st_mode):
        raise SystemExit('non-regular release member: ' + str(path))
    actual[path.relative_to(target).as_posix()] = digest(path)
missing = sorted(set(expected) - set(actual)); extra = sorted(set(actual) - set(expected))
wrong = sorted(name for name in expected if name in actual and actual[name] != expected[name])
if not (missing or extra or wrong):
    for path in target.rglob('*'):
        if path.is_dir():
            path.chmod(0o755)
        else:
            path.chmod(0o755 if path.parent == target and path.name in V['executables'] else 0o644)
print(json.dumps({'missing': missing, 'extra': extra, 'wrong': wrong, 'count': len(actual)}))
'''

PRUNE = r'''
import json, os, pathlib, shutil
removed, freed = [], 0
for base, names in V['targets'].items():
    folder = pathlib.Path(base) / 'releases'
    active = os.readlink(pathlib.Path(base) / 'current')
    for name in names:
        if active.endswith('releases/' + name) or name in V['protect']:
            raise SystemExit('refusing to remove protected release ' + name)
        path = folder / name
        if path.is_symlink() or not path.is_dir():
            continue
        freed += sum(f.lstat().st_size for f in path.rglob('*') if f.is_file() and not f.is_symlink())
        shutil.rmtree(path)
        removed.append(str(path))
print(json.dumps({'removed': removed, 'freed_bytes': freed}))
'''


class Rollout:
    def __init__(self, args):
        self.args = args
        self.server = Server(args.target, args.identity, args.known_hosts, args.port)
        self.go = self.next = None
        if args.go_release:
            self.go = load_go_release(args.go_release)
            if args.next_bundle:
                self.next = load_next_bundle(args.next_bundle, self.go['revision'])
        self.receipts = Path(args.receipts) if args.receipts else None
        if self.receipts:
            self.receipts.mkdir(parents=True, exist_ok=True)

    @property
    def revision(self):
        return self.go['revision']

    def receipt(self, phase, value):
        print(json.dumps({'phase': phase, **value}, sort_keys=True), flush=True)
        if self.receipts:
            path = self.receipts / ('%s-%d.json' % (phase, time.time_ns()))
            with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'w') as stream:
                json.dump({'phase': phase, 'revision': self.go and self.revision, **value}, stream, indent=2, sort_keys=True)

    def inventory(self):
        return self.server.python(INVENTORY, timeout=60, go=GO_BASE, next=NEXT_BASE, journal=JOURNAL, nginx=NGINX)

    def transfer(self, local, target, files):
        prepared = self.server.python(PREPARE, timeout=60, target=target)
        verified = self.server.python(VERIFY, timeout=600, target=target, files=files, executables=list(EXECUTABLES))
        uploaded = False
        # A fresh or partially filled directory is completed; anything that
        # differs from the local release is left for inspection.
        if verified['missing'] and not (verified['extra'] or verified['wrong']):
            with tempfile.NamedTemporaryFile(prefix='accord-release-', suffix='.tar') as archive:
                with tarfile.open(archive.name, 'w') as tar:
                    for name in sorted(files):
                        tar.add(Path(local) / name, arcname=name, recursive=False)
                archive.seek(0)
                self.server.run('umask 022 && tar --no-same-owner --no-same-permissions -C %s -xf -' % shlex.quote(target),
                                stdin=archive, timeout=1800)
            uploaded = True
            verified = self.server.python(VERIFY, timeout=600, target=target, files=files, executables=list(EXECUTABLES))
        if verified['missing'] or verified['extra'] or verified['wrong']:
            raise RuntimeError('%s differs from the local release: %s' % (target, json.dumps(verified)[:2000]))
        return {'target': target, 'created': prepared['created'], 'uploaded': uploaded, 'files': verified['count']}

    def upload(self):
        result = {
            'go': self.transfer(self.args.go_release, GO_BASE + '/releases/' + self.revision, self.go['files']),
            'next': self.transfer(self.args.next_bundle, NEXT_BASE + '/releases/' + self.revision, self.next['files']),
        }
        self.receipt('upload', result)

    def operator(self, mode, extra=()):
        script = '%s/releases/%s/infra/projects/upgrade.py' % (GO_BASE, self.revision)
        remote = ' '.join(['python3', '-B', shlex.quote(script), mode, '--revision', self.revision]
                          + [shlex.quote(item) for item in extra])
        output = self.server.run(remote, timeout=self.args.timeout).decode(errors='replace').strip()
        state = self.inventory()['journal'].get(self.revision)
        self.receipt(mode, {'output': output, 'journal_phase': state})
        return state

    def stage(self):
        before = self.inventory()
        state = self.operator('stage', (
            '--bundle', NEXT_BASE + '/releases/' + self.revision,
            '--manifest-sha256', self.next['manifest_sha256'],
            '--go-binary', GO_BASE + '/releases/' + self.revision + '/agent-board',
            '--go-sha256', self.go['go_sha256'],
            '--expected-nginx-sha256', before['nginx_sha256']))
        if state != 'staged':
            raise RuntimeError('stage did not leave the journal staged: %r' % state)

    def apply(self, resume=False):
        state = self.operator('resume' if resume else 'apply')
        after = self.inventory()
        expected = 'releases/' + self.revision
        if state != 'pending-https-smoke' or after['go_current'] != expected or after['next_current'] != expected:
            raise RuntimeError('apply did not activate %s (journal %r)' % (self.revision, state))

    def smoke(self):
        checks = {}
        for path, needle in (('/readyz', None), ('/app/login', b'/_next/static/')):
            with urllib.request.urlopen(self.args.public_url.rstrip('/') + path, timeout=30) as response:
                body = response.read(2 << 20)
                checks[path] = {'status': response.status, 'ok': response.status == 200 and (needle is None or needle in body)}
        self.receipt('smoke', {'url': self.args.public_url, 'checks': checks})
        if not all(check['ok'] for check in checks.values()):
            raise RuntimeError('external HTTPS smoke failed; the update stays pending-https-smoke')

    def finalize(self):
        state = self.operator('finalize')
        if state != 'complete':
            raise RuntimeError('finalize did not complete the journal: %r' % state)

    def prune(self):
        inventory = self.inventory()
        protect = [self.revision] if self.go else []
        targets = {}
        for base, key, current in ((GO_BASE, 'go_releases', 'go_current'), (NEXT_BASE, 'next_releases', 'next_current')):
            active = (inventory[current] or '').rsplit('/', 1)[-1]
            targets[base] = plan_prune(inventory[key], active, self.args.keep, protect)
        if not any(targets.values()):
            self.receipt('prune', {'removed': [], 'freed_bytes': 0})
            return
        result = self.server.python(PRUNE, timeout=600, targets=targets, protect=protect)
        self.receipt('prune', result)

    def run(self, phase):
        if phase == 'status':
            print(json.dumps(self.inventory(), indent=2, sort_keys=True))
            return
        if phase == 'prune':
            self.prune()
            return
        requested = list(PHASES) if phase == 'all' else [phase]
        journal = self.inventory()['journal'].get(self.revision)
        todo = pending_phases(requested, journal, self.args.resume)
        if not todo:
            self.receipt('noop', {'journal_phase': journal, 'message': 'nothing left to do for this revision'})
        for step in todo:
            if step == 'apply' and journal in RESUMABLE:
                self.apply(resume=True)
            else:
                getattr(self, step)()
        if phase == 'all' and self.args.prune:
            self.prune()


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('phase', choices=('all', 'status') + PHASES + ('prune',))
    parser.add_argument('--target', required=True, help='SSH destination, e.g. root@host')
    parser.add_argument('--identity', help='SSH private key file')
    parser.add_argument('--known-hosts', help='known_hosts file with the server key')
    parser.add_argument('--port', type=int)
    parser.add_argument('--go-release', help='build-release.sh output directory')
    parser.add_argument('--next-bundle', help='build-next.sh / package.py output directory')
    parser.add_argument('--public-url', help='https://host used for the external smoke check')
    parser.add_argument('--receipts', help='directory for per-phase JSON receipts')
    parser.add_argument('--timeout', type=int, default=900, help='seconds per operator phase')
    parser.add_argument('--resume', action='store_true', help='continue an interrupted apply with upgrade.py resume')
    parser.add_argument('--prune', action='store_true', help='after a complete rollout remove old release directories')
    parser.add_argument('--keep', type=int, default=1, help='old releases to keep next to the active one')
    args = parser.parse_args()
    needs_release = args.phase not in ('status', 'prune')
    if needs_release and not (args.go_release and args.next_bundle):
        parser.error('--go-release and --next-bundle are required for ' + args.phase)
    if args.phase in ('all', 'smoke') and not args.public_url:
        parser.error('--public-url is required for the smoke check')
    Rollout(args).run(args.phase)


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, ValueError, OSError, subprocess.TimeoutExpired) as error:
        print('rollout stopped: %s' % error, file=sys.stderr)
        print('No automatic retry or rollback: inspect the server journal under '
              + JOURNAL + ' and rerun the same phase (or --resume).', file=sys.stderr)
        sys.exit(1)
