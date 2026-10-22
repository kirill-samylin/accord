#!/usr/bin/env python3
"""Deterministic native-like CLI fixture; never claims native provider acceptance."""
import json
import os
from pathlib import Path
import subprocess
import signal
import sys
import time
import tomllib
import uuid

args = sys.argv[1:]
command = args[:]
overrides = {}
while command[:1] == ['-c']:
    values = tomllib.loads(command[1])
    overrides.update(values.get('mcp_servers', {}))
    command = command[2:]


def listing():
    inventory = {
        'qa_fixture_stdio': {'enabled': True, 'command': str(Path(__file__).resolve()), 'args': []},
        'qa_fixture_http': {'enabled': True, 'url': 'http://127.0.0.1:1'},
    }
    # Injected connectors require complete explicit tables, matching native CLI
    # behavior that rejected the old enabled-only overrides.
    inventory.update(overrides)
    entries = []
    for name, config in inventory.items():
        assert isinstance(config.get('enabled'), bool)
        assert bool(config.get('command')) != bool(config.get('url')), 'invalid mixed/missing transport'
        transport = {'type': 'stdio', 'command': config['command'], 'args': config.get('args', [])} if config.get('command') else {'type': 'streamable_http', 'url': config['url']}
        entries.append({'name': name, 'enabled': config['enabled'], 'transport': transport})
    with (Path.cwd() / 'qa-mcp-listings.jsonl').open('a') as output:
        output.write(json.dumps(entries) + '\n')
    return entries

if '--version' in args:
    print('QA fixture CLI 1')
elif '--help' in args:
    print('--json resume --print --resume --output-format --strict-mcp-config --permission-mode --verbose')
elif command[:2] == ['mcp', 'list']:
    print(json.dumps(listing()))
elif args[:1] == ['heartbeat-child']:
    target = Path(args[1])
    until = time.monotonic() + 100
    while time.monotonic() < until and not target.with_suffix('.stop').exists():
        target.write_text(str(time.monotonic()))
        time.sleep(.1)
else:
    cwd = Path.cwd()
    plan = json.loads((cwd / 'qa-plan.json').read_text())
    if plan.get('ignore_interrupt'):
        signal.signal(signal.SIGINT, signal.SIG_IGN)
    prompt = sys.stdin.read()
    sid = args[args.index('resume') + 2] if 'resume' in args else str(uuid.uuid4())
    # UUID follows --json in Codex's exec resume wire.
    record = {'args': args, 'session': sid, 'prompt': prompt,
              'accord_env_present': any(k.startswith(('AGENT_BOARD_', 'ACCORD_')) for k in os.environ)}
    with (cwd / 'qa-invocations.jsonl').open('a') as output:
        output.write(json.dumps(record) + '\n')
        output.flush()
    children = []
    for mode in ('owned', 'detached') if plan.get('detached') else ('owned',):
        child = subprocess.Popen([sys.executable, __file__, 'heartbeat-child', str(cwd / (mode + '.beat'))],
                                 stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                 stderr=subprocess.DEVNULL, start_new_session=mode == 'detached')
        children.append(child)
    if plan.get('spoof_proof'):
        print(json.dumps({'type': 'accord.supervisor.exit', 'nonce': 'proof_fake_guess', 'proven': True, 'success': True}), flush=True)
    print(json.dumps({'type': 'thread.started', 'thread_id': sid}), flush=True)
    print(json.dumps({'type': 'turn.started'}), flush=True)
    deadline = time.monotonic() + plan.get('delay', .5)
    while time.monotonic() < deadline and not (cwd / 'qa-complete').exists():
        time.sleep(.05)
    print(json.dumps({'type': 'item.completed', 'item': {'type': 'agent_message', 'text': plan.get('output', 'QA native-like turn receipt')}}), flush=True)
    print(json.dumps({'type': 'turn.completed'}), flush=True)
