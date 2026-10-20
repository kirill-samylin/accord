"""Stage a capable Go + Linux Next candidate against an already deployed TASK008."""
import json
from pathlib import Path
import shutil

from guard import require_capable
from host_ops import HERE, PROJECTS, WORK, host, helper_manifest, nginx_check, preserved
from ingress import render


def stage(args):
    host.regular(host.NGINX)
    host.regular(host.UNIT)
    if host.sha(host.NGINX) != args.expected_nginx_sha256:
        raise RuntimeError('ingress changed since private inventory')
    template = (HERE.parent / 'next/accord-next.service.in').read_text()
    if host.UNIT.read_text() != template.replace('@PORT@', '7893'):
        raise RuntimeError('expected existing reviewed Next7893 unit')
    if host.pointer(host.GO) is None or host.pointer(host.BASE) is None:
        raise RuntimeError('existing Go and Next release required')
    host.ready(7892)
    host.ready(7893, '/app/login', nonce=True)
    host.verify_bundle(args.bundle, args.manifest_sha256, args.revision)
    host.regular(args.go_binary)
    if host.sha(args.go_binary) != args.go_sha256:
        raise RuntimeError('Go binary hash differs')
    require_capable(args.go_binary, PROJECTS)
    host.directory(WORK, 0o700)
    host.sync(WORK.parent)
    work = WORK / args.revision
    if work.exists():
        raise RuntimeError('project release journal exists; inspect before proceeding')
    host.directory(work, 0o700)
    host.sync(WORK)
    release = host.BASE / 'releases' / args.revision
    if not release.exists():
        shutil.copytree(args.bundle, release)
    host.verify_bundle(release, args.manifest_sha256, args.revision)
    go_release = host.GO / 'releases' / args.revision
    host.directory(go_release)
    binary = go_release / 'agent-board'
    if not binary.exists():
        host.atomic(binary, args.go_binary.read_bytes(), 0o755)
    host.regular(binary)
    if host.sha(binary) != args.go_sha256:
        raise RuntimeError('immutable Go release differs')
    metadata = {'revision': args.revision, 'phase': 'staging',
                'old_go': host.pointer(host.GO), 'old_next': host.pointer(host.BASE),
                'nginx_sha256': host.sha(host.NGINX), 'go_sha256': args.go_sha256,
                'manifest_sha256': args.manifest_sha256, 'preserved': preserved(),
                'owner_identity': args.owner_identity,
                'helpers': helper_manifest()}
    host.save_json(work / 'state.json', metadata)
    host.atomic(work / 'previous-nginx', host.NGINX.read_bytes(), 0o600)
    host.atomic(work / 'candidate-nginx', render(host.NGINX.read_text()).encode(), 0o600)
    nginx_check(work / 'candidate-nginx', work)
    unit = Path('/run/systemd/system/accord-next-project-stage.service')
    if unit.exists() or unit.is_symlink():
        raise RuntimeError('unrelated stage unit exists')
    source = template.replace('@PORT@', str(args.stage_port)).replace('/opt/accord/next/current', str(release))
    host.atomic(work / 'accord-next-project-stage.service', source.encode(), 0o600)
    host.run('systemd-analyze', 'verify', str(work / 'accord-next-project-stage.service'))
    if host.run(str(release / 'node'), '--version').strip() != 'v24.19.0':
        raise RuntimeError('Linux Node runtime version differs')
    if not host.run('runuser', '-u', 'accord-next', '--', str(release / 'node'), '-e',
                    'console.log(require(process.argv[1]).versions.vips)', str(release / 'app/node_modules/sharp')).strip():
        raise RuntimeError('Linux sharp unavailable')
    host.atomic(unit, source.encode())
    try:
        host.run('systemctl', 'daemon-reload')
        host.run('systemctl', 'start', unit.name)
        host.ready(args.stage_port, '/app/login', nonce=True)
        metadata['stage_runtime'] = host.run('systemctl', 'show', unit.name,
                                            '-p', 'MemoryCurrent', '-p', 'MemoryPeak', '-p', 'NRestarts')
    finally:
        try:
            host.run('systemctl', 'stop', unit.name)
        finally:
            unit.unlink(missing_ok=True)
            host.run('systemctl', 'daemon-reload')
    metadata.update(phase='staged', candidate_sha256=host.sha(work / 'candidate-nginx'))
    host.save_json(work / 'state.json', metadata)
    print('Capable Go and Linux Next staged; active data unchanged.')
