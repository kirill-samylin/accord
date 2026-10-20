"""Restore runtime/configuration independently; never restore database files."""
import signal

from host import BASE, GO, NGINX, UNIT, atomic, ready, run, save_json, switch


def rollback(work, state):
    previous = {}
    for name in ('SIGTERM', 'SIGINT', 'SIGHUP'):
        if hasattr(signal, name):
            number = getattr(signal, name)
            previous[number] = signal.signal(number, signal.SIG_IGN)
    try:
        _rollback(work, state)
    finally:
        for number, handler in previous.items():
            signal.signal(number, handler)


def _rollback(work, state):
    errors = []

    def attempt(label, action):
        try:
            action()
        except Exception as error:
            errors.append(label + ': ' + type(error).__name__)

    attempt('stop Next', lambda: run('systemctl', 'stop', 'accord-next.service'))
    attempt('stop Go', lambda: run('systemctl', 'stop', 'accord.service'))
    attempt('Go pointer', lambda: switch(GO, state['old_go']))
    attempt('Next pointer', lambda: switch(BASE, state['old_next']))
    attempt('nginx file', lambda: atomic(NGINX, (work / 'previous-nginx').read_bytes()))

    def restore_unit():
        if state['old_unit']:
            atomic(UNIT, (work / 'previous-unit').read_bytes())
        else:
            UNIT.unlink(missing_ok=True)

    # Failure restoring one file never skips another file or the original writer.
    attempt('Next unit', restore_unit)
    attempt('daemon reload', lambda: run('systemctl', 'daemon-reload'))
    action = 'enable' if state['old_next_enabled'] else 'disable'
    if state['old_next_enabled'] or state['old_unit']:
        attempt('Next enabled state', lambda: run('systemctl', action, 'accord-next.service'))
    else:
        # Disable before deleting the unit may leave a wants link; remove exactly
        # this managed link, never rewrite the whole target wants directory.
        attempt('Next enabled link', lambda: (
            UNIT.parent / 'multi-user.target.wants/accord-next.service').unlink(missing_ok=True))
    attempt('Go start', lambda: run('systemctl', 'start', 'accord.service'))
    attempt('Go readiness', lambda: ready(7892))
    if state['old_next_active']:
        attempt('Next start', lambda: run('systemctl', 'start', 'accord-next.service'))
        attempt('Next readiness', lambda: ready(state['port'], '/app/login', nonce=True))
    attempt('nginx syntax', lambda: run('nginx', '-t'))
    attempt('nginx reload', lambda: run('systemctl', 'reload', 'nginx'))
    state.update(phase='rollback-incomplete' if errors else 'rolled-back', rollback_errors=errors)
    save_json(work / 'state.json', state)
    if errors:
        raise RuntimeError('rollback incomplete: ' + '; '.join(errors))
    print('Previous executable/configuration restored; database history preserved.')
