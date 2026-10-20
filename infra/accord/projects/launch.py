"""Verify effective systemd arguments, not only executable capabilities or PID."""
import hashlib
from pathlib import Path
import re
import shlex
import subprocess

UNIT = 'accord.service'
BINARY = '/opt/accord/current/agent-board'
PROJECTS = '/var/lib/accord/projects-data'
ARGV = [BINARY, 'serve', '--db', '/var/lib/accord/board.db', '--listen',
        '127.0.0.1:7892', '--web-trusted-proxy', '--projects-data', PROJECTS]
GUARD = ['/usr/bin/python3', '/opt/accord/admin/projects/guard.py', '--binary',
         BINARY, '--projects-data', PROJECTS, '--unit', UNIT]


def command(*args):
    return subprocess.check_output(args, text=True, timeout=5,
                                   env={'PATH': '/usr/bin:/bin', 'LANG': 'C'})


def require_launch(run=command):
    # systemctl exposes the resolved command after all lexically ordered drop-ins.
    # Matching a file in isolation would miss a later web.conf override.
    text = run('systemctl', 'show', UNIT, '-p', 'ExecStart', '--value').strip()
    entries = re.findall(r'\{ path=([^ ;]+) ; argv\[\]=([^;]+) ; ignore_errors=(yes|no) ;', text)
    if (len(entries) != 1 or text.count('{') != 1 or entries[0][0] != BINARY or
            shlex.split(entries[0][1]) != ARGV or entries[0][2] != 'no'):
        raise RuntimeError('effective service command does not enforce project scope')
    guards = run('systemctl', 'show', UNIT, '-p', 'ExecStartPre', '--value')
    commands = re.findall(r'\{ path=([^ ;]+) ; argv\[\]=([^;]+) ; ignore_errors=(yes|no) ;', guards)
    if not any(path == GUARD[0] and shlex.split(args) == GUARD and ignored == 'no'
               for path, args, ignored in commands):
        raise RuntimeError('effective project startup guard is missing')


def require_backup(run=command):
    text = run('systemctl', 'show', 'accord-web-backup.service', '-p', 'ExecStart', '--value').strip()
    entries = re.findall(r'\{ path=([^ ;]+) ; argv\[\]=([^;]+) ; ignore_errors=(yes|no) ;', text)
    expected = ['/usr/bin/python3', '/opt/accord/admin/projects/backup.py']
    if (len(entries) != 1 or text.count('{') != 1 or entries[0][0] != expected[0] or
            shlex.split(entries[0][1]) != expected or entries[0][2] != 'no'):
        raise RuntimeError('effective backup command does not include all projects')


def require_process(pid, digest, run=command, read=lambda path: path.read_bytes()):
    if not isinstance(pid, int) or pid <= 1:
        raise RuntimeError('owned live service PID required')
    current = lambda: run('systemctl', 'show', UNIT, '-p', 'MainPID', '--value').strip()
    if current() != str(pid):
        raise RuntimeError('service PID changed before process verification')
    proc = Path('/proc') / str(pid)
    if read(proc / 'cmdline') != b'\0'.join(arg.encode() for arg in ARGV) + b'\0':
        raise RuntimeError('running process does not enforce project scope')
    if hashlib.sha256(read(proc / 'exe')).hexdigest() != digest or current() != str(pid):
        raise RuntimeError('running executable or service PID differs')
