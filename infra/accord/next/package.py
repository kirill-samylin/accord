#!/usr/bin/env python3
"""Assemble standalone output, Linux Node and exact lock-pinned sharp binaries.

Build Next before calling this tool. Node is supplied by the operator after
verification against the official Node SHASUMS256 list. No VPS build is needed.
"""
import argparse
import base64
import hashlib
import io
import json
import pathlib
import re
import shutil
import tarfile
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[3]


def sha(path):
    value = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            value.update(block)
    return value.hexdigest()


def linux_package(lock, name, destination):
    entry = lock['packages']['node_modules/' + name]
    url = entry['resolved']
    if not url.startswith('https://registry.npmjs.org/' + name + '/-/'):
        raise RuntimeError('package must come from the exact public npm lock entry')
    with urllib.request.urlopen(url, timeout=60) as response:
        data = response.read(64 << 20)
    integrity = 'sha512-' + base64.b64encode(hashlib.sha512(data).digest()).decode()
    if integrity != entry['integrity']:
        raise RuntimeError('Linux optional package integrity mismatch')
    destination.mkdir(parents=True)
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
        for member in archive:
            relative = pathlib.PurePosixPath(member.name)
            if (relative.is_absolute() or '..' in relative.parts or
                    not relative.parts or relative.parts[0] != 'package' or
                    not (member.isfile() or member.isdir())):
                raise RuntimeError('unsafe npm archive member')
            target = destination.joinpath(*relative.parts[1:])
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                with archive.extractfile(member) as source, target.open('xb') as output:
                    shutil.copyfileobj(source, output)
    return {'version': entry['version'], 'integrity': integrity, 'resolved': url}


def package(web, output, node, node_sha, revision):
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise RuntimeError('full source revision required')
    if not re.fullmatch(r'[0-9a-f]{64}', node_sha) or sha(node) != node_sha:
        raise RuntimeError('operator-verified Linux Node checksum required')
    header = node.read_bytes()[:20]
    if header[:5] != b'\x7fELF\x02' or header[18:20] != b'\x3e\x00':
        raise RuntimeError('Node must be Linux ELF64 x86_64')
    build = web / '.local/next'
    for name in ('standalone/server.js', 'BUILD_ID', 'routes-manifest.json'):
        if not (build / name).is_file():
            raise RuntimeError('complete standalone build required')
    routes = json.loads((build / 'routes-manifest.json').read_text())
    rewrites = routes['rewrites']['afterFiles']
    expected = [{'source': '/web/', 'destination': 'http://127.0.0.1:7892/web/'},
                {'source': '/web/:path*', 'destination': 'http://127.0.0.1:7892/web/:path*'}]
    if [{key: item[key] for key in ('source', 'destination')} for item in rewrites] != expected:
        raise RuntimeError('unexpected external rewrites')
    output.mkdir(parents=True)
    output.chmod(0o755)
    app = output / 'app'
    # Dereference package-manager symlinks. The final immutable bundle has none.
    shutil.copytree(build / 'standalone', app, symlinks=False)
    shutil.copytree(build / 'static', app / '.local/next/static')
    shutil.copytree(web / 'public', app / 'public')
    shutil.copyfile(node, output / 'node')
    license_path = node.parent.parent / 'LICENSE'
    if not license_path.is_file():
        raise RuntimeError('official Node distribution LICENSE required')
    shutil.copyfile(license_path, output / 'NODE-LICENSE')
    (output / 'node').chmod(0o755)
    lock = json.loads((web / 'package-lock.json').read_text())
    image_modules = app / 'node_modules/@img'
    for path in image_modules.glob('*darwin*'):
        shutil.rmtree(path)
    dependencies = {}
    for name in ('@img/sharp-linux-x64', '@img/sharp-libvips-linux-x64'):
        dependencies[name] = linux_package(lock, name, app / 'node_modules' / name)
    # Refuse an unaccounted host binary rather than hoping it is unused.
    for path in app.rglob('*'):
        if path.is_symlink():
            raise RuntimeError('symlink in standalone bundle')
        if path.is_file() and path.suffix in ('.node', '.dylib'):
            header = path.read_bytes()[:20]
            if header[:5] != b'\x7fELF\x02' or header[18:20] != b'\x3e\x00':
                raise RuntimeError('non-Linux native code in standalone bundle')
    metadata = {
        'format': 1, 'revision': revision, 'node_version': 'v24.19.0',
        'node_sha256': node_sha, 'next_version': '16.4.2',
        'build_id': (build / 'BUILD_ID').read_text().strip(),
        'go_origin': 'http://127.0.0.1:7892', 'linux_optional': dependencies,
        'package_lock_sha256': sha(web / 'package-lock.json'),
    }
    (output / 'SOURCE.json').write_text(json.dumps(metadata, sort_keys=True, indent=2) + '\n')
    files = {}
    for path in sorted(output.rglob('*')):
        if path.is_file():
            path.chmod(0o755 if path.name == 'node' and path.parent == output else 0o644)
            files[str(path.relative_to(output))] = sha(path)
        elif path.is_dir():
            path.chmod(0o755)
    manifest = {'format': 1, 'revision': revision, 'files': files}
    (output / 'manifest.json').write_text(json.dumps(manifest, sort_keys=True, indent=2) + '\n')
    (output / 'manifest.json').chmod(0o644)
    print('Bundle manifest SHA256:', sha(output / 'manifest.json'))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--web', type=pathlib.Path, default=ROOT / 'web')
    parser.add_argument('--output', type=pathlib.Path, required=True)
    parser.add_argument('--node', type=pathlib.Path, required=True)
    parser.add_argument('--node-sha256', required=True)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    package(args.web, args.output, args.node, args.node_sha256, args.revision)


if __name__ == '__main__':
    main()
