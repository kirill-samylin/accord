#!/usr/bin/python3
"""Publish only Accord's private HTTPS8443 while preserving every other route."""
import argparse
import copy
import fcntl
import json
import os
import subprocess
import sys
import urllib.request

PORT = '8443'
TARGET = 'http://127.0.0.1:7892'


def status():
    result = subprocess.run(['tailscale', 'serve', 'status', '--json'],
                            check=True, stdout=subprocess.PIPE, text=True)
    return json.loads(result.stdout)


def unrelated(config):
    result = copy.deepcopy(config)
    tcp = result.get('TCP', {})
    tcp.pop(PORT, None)
    if not tcp:
        result.pop('TCP', None)
    web = result.get('Web', {})
    for host in list(web):
        if host.endswith(':' + PORT):
            del web[host]
    if not web:
        result.pop('Web', None)
    return result


def own_route(config):
    expected = {'Handlers': {'/': {'Proxy': TARGET}}}
    web = [value for host, value in config.get('Web', {}).items() if host.endswith(':' + PORT)]
    return config.get('TCP', {}).get(PORT) == {'HTTPS': True} and web == [expected]


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--disable', action='store_true')
    args = parser.parse_args()
    if os.geteuid() != 0 or sys.platform != 'linux':
        raise RuntimeError('run as root on the Linux server')
    fd = os.open('/run/lock/accord-admin.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        before = status()
        if any(before.get('AllowFunnel', {}).values()):
            raise RuntimeError('public Funnel detected; inspect manually before publishing credentials')
        exists = PORT in before.get('TCP', {}) or any(host.endswith(':' + PORT) for host in before.get('Web', {}))
        if exists and not own_route(before):
            raise RuntimeError('port8443 already belongs to another service')
        if args.disable:
            if exists:
                subprocess.run(['tailscale', 'serve', '--https=' + PORT, 'off'],
                               check=True, stdout=subprocess.DEVNULL)
        else:
            client = urllib.request.build_opener(urllib.request.ProxyHandler({}))
            with client.open(TARGET + '/readyz', timeout=5) as response:
                if response.status != 200:
                    raise RuntimeError('Accord is not ready')
            subprocess.run(['tailscale', 'serve', '--bg', '--https=' + PORT, '--yes', TARGET],
                           check=True, stdout=subprocess.DEVNULL)
        after = status()
        if unrelated(before) != unrelated(after):
            raise RuntimeError('unrelated Tailscale configuration changed; inspect manually without reset')
        if not args.disable and not own_route(after):
            raise RuntimeError('Accord route verification failed')
        if any(after.get('AllowFunnel', {}).values()):
            raise RuntimeError('Funnel must remain disabled')
        print('Accord private HTTPS8443', 'disabled.' if args.disable else 'configured; unrelated routes preserved.')


if __name__ == '__main__':
    try:
        run()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print('Accord publication failed:', type(error).__name__, file=sys.stderr)
        sys.exit(1)
