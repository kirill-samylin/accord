"""Install immutable candidates and exercise Linux runtime before activation."""
import os
import grp
import pathlib
import pwd
import shutil
import subprocess

from host import BASE, GO, NGINX, UNIT, BACKUPS, atomic, directory, pointer, ready
from host import regular, run, save_json, sha, verify_bundle
from render import render

HERE = pathlib.Path(__file__).resolve().parent


def account():
    try:
        user = pwd.getpwnam('accord-next')
    except KeyError:
        run('useradd', '--system', '--user-group', '--home-dir', '/nonexistent',
            '--shell', '/usr/sbin/nologin', 'accord-next')
        user = pwd.getpwnam('accord-next')
    groups = os.getgrouplist(user.pw_name, user.pw_gid)
    accord = pwd.getpwnam('accord')
    if (user.pw_uid in (0, accord.pw_uid) or user.pw_dir != '/nonexistent'
            or user.pw_shell != '/usr/sbin/nologin'
            or grp.getgrgid(user.pw_gid).gr_name != 'accord-next'
            or set(groups) != {user.pw_gid}):
        raise RuntimeError('unexpected existing accord-next account')


def unit(release, port):
    return (HERE / 'accord-next.service.in').read_text().replace('@PORT@', str(port)).replace(
        '/opt/accord/next/current', str(release)).encode()


def preserved():
    paths = [pathlib.Path('/etc/accord/server.env'), pathlib.Path('/etc/accord/web.env'),
             pathlib.Path('/etc/systemd/system/accord.service'), pathlib.Path('/etc/nginx/nginx.conf')]
    for folder in ('/etc/systemd/system/accord.service.d', '/etc/systemd/system'):
        base = pathlib.Path(folder)
        if folder.endswith('.d'):
            paths.extend(sorted(base.glob('*.conf')))
        else:
            paths.extend(sorted(base.glob('accord*.timer')))
            paths.extend(path for path in sorted(base.glob('accord*.service'))
                         if path.name != 'accord-next.service' and path not in paths)
    return {str(path): sha(path) for path in paths if path.is_file()}


def nginx_check(candidate, work):
    main = pathlib.Path('/etc/nginx/nginx.conf').read_text()
    old = 'include /etc/nginx/conf.d/accord-public.conf;'
    if main.count(old) != 1:
        raise RuntimeError('unexpected nginx include topology')
    temporary = work / 'nginx-check.conf'
    atomic(temporary, main.replace(old, 'include ' + str(candidate) + ';').encode(), 0o600)
    run('nginx', '-t', '-c', str(temporary))


def stage(args):
    regular(NGINX)
    if sha(NGINX) != args.expected_nginx_sha256:
        raise RuntimeError('nginx changed since reviewed inventory')
    verify_bundle(args.bundle, args.manifest_sha256, args.revision)
    regular(args.go_binary)
    if sha(args.go_binary) != args.go_sha256:
        raise RuntimeError('Go binary checksum mismatch')
    if pointer(GO) is None:
        raise RuntimeError('existing healthy Go service required')
    ready(7892)
    directory(BASE)
    directory(BASE / 'releases')
    directory(BACKUPS, 0o700)
    work = BACKUPS / args.revision
    if work.exists():
        raise RuntimeError('deployment journal already exists; inspect/rollback first')
    directory(work, 0o700)
    release = BASE / 'releases' / args.revision
    if release.exists():
        verify_bundle(release, args.manifest_sha256, args.revision)
    else:
        shutil.copytree(args.bundle, release)
        verify_bundle(release, args.manifest_sha256, args.revision)
    go_release = GO / 'releases' / args.revision
    directory(go_release)
    binary = go_release / 'agent-board'
    if binary.exists():
        regular(binary)
        if sha(binary) != args.go_sha256:
            raise RuntimeError('existing immutable Go release differs')
    else:
        atomic(binary, args.go_binary.read_bytes(), 0o755)
    old_next = pointer(BASE)
    enabled = subprocess.run(['systemctl', 'is-enabled', '--quiet', 'accord-next.service'],
                             stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0
    active = subprocess.run(['systemctl', 'is-active', '--quiet', 'accord-next.service'],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0
    metadata = {'revision': args.revision, 'manifest_sha256': args.manifest_sha256,
                'go_sha256': args.go_sha256, 'old_go': pointer(GO), 'old_next': old_next,
                'old_next_enabled': enabled, 'old_next_active': active,
                'nginx_sha256': sha(NGINX), 'preserved': preserved(), 'phase': 'staging'}
    atomic(work / 'previous-nginx', NGINX.read_bytes(), 0o600)
    if UNIT.exists():
        regular(UNIT)
        atomic(work / 'previous-unit', UNIT.read_bytes(), 0o600)
    metadata['old_unit'] = UNIT.exists()
    save_json(work / 'state.json', metadata)
    atomic(work / 'candidate-nginx', render(NGINX.read_text(), args.port).encode(), 0o600)
    atomic(work / 'accord-next.service', unit(BASE / 'current', args.port))
    nginx_check(work / 'candidate-nginx', work)
    verify = work / 'verify'
    directory(verify, 0o700)
    atomic(verify / 'accord-next.service', unit(release, args.port))
    run('systemd-analyze', 'verify', str(verify / 'accord-next.service'))
    account()
    if run(str(release / 'node'), '--version').strip() != 'v24.19.0':
        raise RuntimeError('Linux Node runtime version differs')
    # Force Linux native module load, even though UI currently has no next/image.
    native = run('runuser', '-u', 'accord-next', '--', str(release / 'node'), '-e',
                 'console.log(require(process.argv[1]).versions.vips)',
                 str(release / 'app/node_modules/sharp')).strip()
    if not native:
        raise RuntimeError('Linux sharp runtime unavailable')
    stage_unit = pathlib.Path('/run/systemd/system/accord-next-stage.service')
    if stage_unit.exists():
        raise RuntimeError('unrelated staging unit already exists')
    atomic(stage_unit, unit(release, args.stage_port))
    try:
        run('systemctl', 'daemon-reload')
        run('systemctl', 'start', 'accord-next-stage.service')
        ready(args.stage_port, '/app/login', nonce=True)
        metadata['stage_runtime'] = run('systemctl', 'show', 'accord-next-stage.service',
                                       '-p', 'MemoryCurrent', '-p', 'MemoryPeak', '-p', 'NRestarts')
    finally:
        try:
            run('systemctl', 'stop', 'accord-next-stage.service')
        finally:
            stage_unit.unlink(missing_ok=True)
            run('systemctl', 'daemon-reload')
    metadata.update(phase='staged', port=args.port, stage_port=args.stage_port,
                    candidate_sha256=sha(work / 'candidate-nginx'),
                    unit_sha256=sha(work / 'accord-next.service'))
    save_json(work / 'state.json', metadata)
    print('Linux runtime staged; activation remains a separate command.')
