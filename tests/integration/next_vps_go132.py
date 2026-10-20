#!/usr/bin/env python3
"""Root executes this reviewed payload on Linux; it never reads production state."""
import hashlib
import http.cookiejar
import json
import os
import pathlib
import socket
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request


def main(binary_name, expected_sha):
    binary = pathlib.Path(binary_name)
    if not binary.is_absolute() or binary.is_symlink() or not binary.is_file():
        raise RuntimeError('regular absolute candidate executable required')
    digest = hashlib.sha256(binary.read_bytes()).hexdigest()
    if digest != expected_sha:
        raise RuntimeError('candidate executable hash differs')
    port = 7895
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', port))
    origin = 'http://127.0.0.1:' + str(port)
    result = {'result': 'FAIL', 'go_sha256': digest, 'checks': []}
    with tempfile.TemporaryDirectory(prefix='accord-qa132-', dir='/run') as folder:
        os.chmod(folder, 0o700)
        root = pathlib.Path(folder)
        database, token = root / 'board.db', root / 'operator.token'
        subprocess.run([str(binary), 'token', 'create', '--db', str(database),
                        '--owner', 'isolated-linux-qa132', '--machine', 'isolated-qa132',
                        '--output', str(token)], check=True, capture_output=True, timeout=20)
        if token.stat().st_mode & 0o777 != 0o600:
            raise RuntimeError('isolated token permissions differ')
        with (root / 'go.log').open('wb') as log:
            child = subprocess.Popen([str(binary), 'serve', '--db', str(database),
                                      '--listen', '127.0.0.1:' + str(port),
                                      '--web-origin', origin, '--web-data', str(root / 'web')],
                                     stdin=subprocess.DEVNULL, stdout=log, stderr=log)
            try:
                cookies = http.cookiejar.CookieJar()
                client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies))

                def request(path, fields=None):
                    headers = {'Accept': 'application/json'}
                    body = None
                    if fields is not None:
                        headers.update({'Content-Type': 'application/x-www-form-urlencoded', 'Origin': origin})
                        body = urllib.parse.urlencode(fields).encode()
                    req = urllib.request.Request(origin + path, data=body, headers=headers)
                    with client.open(req, timeout=5) as response:
                        if response.status != 200:
                            raise RuntimeError('isolated response status differs')
                        return response.read()

                until = time.monotonic() + 15
                while True:
                    if child.poll() is not None:
                        raise RuntimeError('candidate Go exited before readiness')
                    try:
                        request('/readyz')
                        break
                    except OSError:
                        if time.monotonic() >= until:
                            raise RuntimeError('isolated readiness timeout') from None
                        time.sleep(0.1)
                result['checks'].append('candidate-linux-readyz')
                page = json.loads(request('/web/login'))
                if not page.get('ok') or page.get('page', {}).get('name') != 'login':
                    raise RuntimeError('JSON login projection missing')
                csrf = page['page']['csrf']
                login = json.loads(request('/web/login', {'csrf': csrf, 'token': token.read_text().strip()}))
                if not login.get('ok') or login.get('redirect') != '/web/':
                    raise RuntimeError('isolated JSON login rejected')
                overview = json.loads(request('/web/'))
                if not overview.get('ok') or overview.get('page', {}).get('name') != 'overview':
                    raise RuntimeError('isolated authenticated projection missing')
                result['checks'].extend(['candidate-json-login', 'candidate-cookie-overview'])
            finally:
                if child.poll() is None:
                    child.terminate()
                    try:
                        child.wait(timeout=8)
                    except subprocess.TimeoutExpired:
                        child.kill()
                        child.wait(timeout=3)
                if child.returncode != 0:
                    raise RuntimeError('isolated Go did not stop cleanly')
    result.update(result='PASS', fixture_removed=not pathlib.Path(folder).exists(), process_stopped=True)
    print(json.dumps(result, sort_keys=True))


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2])
