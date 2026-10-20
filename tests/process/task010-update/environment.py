"""Relocate operator I/O; actual Go/SQLite, simulated systemd and Next runtime."""
from contextlib import ExitStack
import json
import os
from pathlib import Path
import shutil
import socket
import stat
import subprocess
import sys
import urllib.request
from unittest.mock import patch

from fixture import Process, command, free_port

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO / 'infra/accord/projects'))
import guard
import host_ops
import launch
import operate
import upgrade
import upgrade_storage
import upgrade_verify

host = host_ops.host
OLD = '5' * 40
NEW = 'c' * 40


class Environment:
    def __init__(self, root, old_binary=None, new_binary=None, existing=False):
        self.root, self.state = root, root / 'state'
        self.go, self.next = root / 'go', root / 'next'
        self.admin, self.work = root / 'admin', root / 'journals'
        self.projects, self.backups = self.state / 'projects-data', root / 'backups'
        self.dropin, self.backup_dropin = root / 'units/zz.conf', root / 'units/backup.conf'
        self.nginx = root / 'ingress.conf'
        metadata = root / 'fixture.json'
        if existing:
            config = json.loads(metadata.read_text())
            self.go_port, self.next_port = config['go_port'], config['next_port']
        else:
            self.go_port, self.next_port = free_port(), free_port()
        self.processes = {}
        self.calls = []
        self.fail = None
        self.work_release = self.work / NEW
        if existing:
            return
        metadata.write_text(json.dumps({'go_port': self.go_port, 'next_port': self.next_port}))
        metadata.chmod(0o600)
        for folder in (self.go, self.next, self.admin, self.work, self.backups, self.dropin.parent):
            folder.mkdir(mode=0o700)
        for revision, binary in ((OLD, old_binary), (NEW, new_binary)):
            folder = self.go / 'releases' / revision
            folder.mkdir(parents=True, mode=0o700)
            shutil.copyfile(binary, folder / 'agent-board')
            (folder / 'agent-board').chmod(0o700)
            (self.next / 'releases' / revision).mkdir(parents=True, mode=0o700)
        (self.go / 'current').symlink_to('releases/' + OLD)
        (self.next / 'current').symlink_to('releases/' + OLD)
        self.nginx.write_bytes(b'synthetic unchanged ingress')
        self.nginx.chmod(0o600)
        self.work_release = self.work / NEW
        self.work_release.mkdir(mode=0o700)
        (self.work_release / 'candidate-nginx').write_bytes(self.nginx.read_bytes())
        (self.work_release / 'candidate-nginx').chmod(0o600)
        self.dropin.write_bytes((host_ops.HERE / 'accord-projects.conf').read_bytes())
        self.backup_dropin.write_bytes((host_ops.HERE / 'accord-projects-backup.conf').read_bytes())

    def bounded(self, value):
        value = Path(value)
        assert value.is_absolute() and self.root in value.parents, 'I/O escaped synthetic root'
        for entry in value.parents:
            if entry == self.root.parent:
                break
            if entry.exists():
                assert entry.is_dir() and not entry.is_symlink(), 'unsafe fixture ancestor'
        return value

    def regular(self, path):
        path = Path(path)
        assert self.root in path.parents or REPO / 'infra/accord' in path.parents, 'regular read escaped fixture/source'
        info = path.lstat()
        assert stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and not info.st_mode & 0o022
        return info

    def directory(self, path, mode=0o755):
        path = self.bounded(path)
        path.mkdir(parents=True, exist_ok=True, mode=mode)
        assert path.is_dir() and not path.is_symlink()
        path.chmod(mode)

    def mapped_path(self, value):
        mapped = {'/var/backups/accord-web': self.backups,
                  '/opt/accord/current/agent-board': self.go / 'current/agent-board'}
        if str(value) in mapped:
            return mapped[str(value)]
        path = Path(value)
        assert path == self.root or self.root in path.parents, 'operator constant escaped relocation'
        return path

    def start_go(self):
        assert 'go' not in self.processes
        assert not (self.state / 'START_REQUIRES_INSPECTION').exists(), 'cold boot marker bypassed'
        self.processes['go'] = Process(self.go / 'current/agent-board', self.state, self.go_port,
                                       f'http://127.0.0.1:{self.go_port}')

    def stop(self, kind):
        process = self.processes.pop(kind, None)
        if process:
            process.stop()

    def run(self, *args):
        self.calls.append(args)
        if args[0] == 'runuser':
            assert args[:4] == ('runuser', '-u', 'accord', '--')
            binary = Path(args[4])
            assert binary == self.go / 'current/agent-board'
            for option in ('--projects-data', '--db', '--web-data'):
                self.bounded(args[args.index(option) + 1])
            assert args[5:7] == ('projects', 'migrate-storage'), 'unauthorized administrative operation'
            return command(binary, args[5:])
        assert args[0] == 'systemctl', 'unrecognized host command'
        if args[1:] == ('daemon-reload',):
            return ''
        assert len(args) == 3 and args[2] in ('accord.service', 'accord-next.service', 'accord-web-backup.service')
        kind = 'go' if args[2] == 'accord.service' else 'next'
        if args[1] == 'stop':
            self.stop(kind)
        elif args[1] == 'start' and kind == 'go':
            if self.fail == 'start-go':
                self.fail = None
                raise RuntimeError('injected owned startup failure')
            self.start_go()
        elif args[1] == 'start' and args[2] == 'accord-web-backup.service':
            from snapshot import snapshot
            self.stop('go')
            snapshot(self.state / 'board.db', self.state / 'web', self.projects, self.backups)
            self.start_go()
        elif args[1] == 'start' and kind == 'next':
            # Next/systemd are explicit simulated boundaries, no fake Next PASS.
            self.processes['next'] = None
        else:
            raise AssertionError('unauthorized synthetic lifecycle command')
        return ''

    def ready(self, port, *_args, **_kwargs):
        assert port in (7892, 7893)
        if port == 7892:
            process = self.processes['go']
            assert process.child.poll() is None
            with urllib.request.urlopen(process.url + '/readyz', timeout=2) as response:
                assert response.status == 200
        else:
            assert 'next' in self.processes

    def port_free(self, port):
        port = {7892: self.go_port, 7893: self.next_port}[port]
        with socket.socket() as sock:
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            sock.bind(('127.0.0.1', port))

    def owned_listener(self, port, unit):
        assert (port, unit) in ((7892, 'accord.service'), (7893, 'accord-next.service'))
        if port == 7893:
            assert 'next' in self.processes
            return -1  # Explicit placeholder is never passed to Go process verifier.
        process = self.processes['go']
        assert process.child.poll() is None
        return process.child.pid

    def require_process(self, pid, digest):
        process = self.processes['go']
        assert process.child.pid == pid and process.child.poll() is None
        assert host.sha(self.go / 'current/agent-board') == digest
        if sys.platform == 'linux':
            proc = Path('/proc') / str(pid)
            assert proc.joinpath('cmdline').read_bytes() == b'\0'.join(a.encode() for a in process.args) + b'\0'
            assert host.sha(proc / 'exe') == digest

    def patches(self):
        stack = ExitStack()
        for module in (host_ops, operate, upgrade, upgrade_storage, upgrade_verify):
            for name, value in (('STATE', self.state), ('PROJECTS', self.projects), ('WORK', self.work),
                                ('ADMIN', self.admin), ('DROPIN', self.dropin), ('BACKUP_DROPIN', self.backup_dropin),
                                ('LEGACY_DROPIN', self.root / 'units/absent-legacy.conf')):
                if hasattr(module, name):
                    stack.enter_context(patch.object(module, name, value))
        for name, value in (('GO', self.go), ('BASE', self.next), ('NGINX', self.nginx)):
            stack.enter_context(patch.object(host, name, value))
        for name in ('regular', 'directory', 'run', 'ready'):
            stack.enter_context(patch.object(host, name, getattr(self, name)))
        stack.enter_context(patch.object(host, 'verify_bundle', self.verify_bundle))
        stack.enter_context(patch.object(operate, 'preserved', lambda: {'fixture': 'unchanged'}))
        for name in ('port_free', 'owned_listener', 'require_process'):
            stack.enter_context(patch.object(upgrade, name, getattr(self, name)))
        stack.enter_context(patch.object(upgrade, 'Path', self.mapped_path))
        # Live systemd introspection is not exercised by this harness.
        stack.enter_context(patch.object(upgrade_verify, 'require_launch', lambda: None))
        stack.enter_context(patch.object(upgrade_verify, 'require_backup', lambda: None))
        return stack

    def verify_bundle(self, folder, digest, revision):
        assert folder == self.next / 'releases' / NEW and revision == NEW
        assert digest == 'synthetic-next-boundary'

    def prepare(self):
        helpers = host_ops.helper_manifest()
        prior = {}
        for relative in helpers:
            if Path(relative).name in ('upgrade.py', 'upgrade_storage.py', 'upgrade_verify.py', 'migration_files.py'):
                continue
            target = self.admin / relative
            self.directory(target.parent)
            target.write_bytes((host_ops.HERE.parent / relative).read_bytes())
            target.chmod(0o644)
            prior[relative] = host.sha(target)
        (self.admin / 'web-backup.py').write_bytes((self.admin / 'web/backup.py').read_bytes())
        (self.admin / 'web-backup.py').chmod(0o644)
        (self.work / OLD).mkdir(mode=0o700)
        host.save_json(self.work / OLD / 'state.json', {'revision': OLD, 'phase': 'complete', 'helpers': prior})
        baseline = upgrade_verify.baseline_helpers(helpers)
        state = {'kind': 'capable-update', 'phase': 'staged', 'revision': NEW, 'helpers': helpers,
                 'installed_helpers': baseline, 'go_sha256': host.sha(self.go / 'releases' / NEW / 'agent-board'),
                 'manifest_sha256': 'synthetic-next-boundary', 'candidate_sha256': host.sha(self.nginx),
                 'nginx_sha256': host.sha(self.nginx), 'preserved': {'fixture': 'unchanged'},
                 'old_go': 'releases/' + OLD, 'old_next': 'releases/' + OLD}
        host.save_json(self.work_release / 'state.json', state)
        return state

    def cleanup(self):
        self.stop('next')
        self.stop('go')
        self.port_free(7892)
        self.port_free(7893)
