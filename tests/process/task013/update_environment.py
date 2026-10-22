"""Real Go/SQLite operators with explicit simulated systemd/Next boundaries."""
import importlib.util
import json
import os
from pathlib import Path
import sys

REPO = Path(__file__).resolve().parents[3]
sys.path.append(str(REPO / 'tests/process/task011-update'))
spec = importlib.util.spec_from_file_location('task013_prior_environment', REPO / 'tests/process/task011-update/environment.py')
prior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prior)
from fixture import capture, command, rebind, seed

host, upgrade, guard, OLD, NEW = prior.host, prior.upgrade, prior.guard, prior.OLD, prior.NEW

def installed_guard(path):
    spec = importlib.util.spec_from_file_location('task013_installed_guard', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class Environment(prior.Environment):
    def start_go(self):
        installed_guard(self.admin / 'projects/guard.py').require_capable(
            (self.go / 'current/agent-board').resolve(), self.projects)
        super().start_go()

    def run(self, *args):
        if args[0] == 'runuser':
            installed = installed_guard(self.admin / 'projects/guard.py')
            installed.require_capable(self.go / 'releases' / NEW / 'agent-board', self.projects)
            try:
                installed.require_capable(self.go / 'releases' / OLD / 'agent-board', self.projects)
            except RuntimeError:
                pass
            else:
                raise AssertionError('old schema4 accepted before DDL')
            assert (self.state / 'START_REQUIRES_INSPECTION').is_file()
        return super().run(*args)

    def prepare(self):
        old_infra = Path(os.environ.get('ACCORD_TASK013_OLD_INFRA', REPO / '.local/task012/checkout/infra/accord')).resolve()
        helpers = prior.host_ops.helper_manifest()
        previous = {}
        for relative in helpers:
            source, target = old_infra / relative, self.admin / relative
            assert source.is_file() and not source.is_symlink()
            self.directory(target.parent)
            target.write_bytes(source.read_bytes())
            target.chmod(0o644)
            previous[relative] = host.sha(target)
        alias = self.admin / 'web-backup.py'
        alias.write_bytes((self.admin / 'web/backup.py').read_bytes())
        alias.chmod(0o644)
        (self.work / OLD).mkdir(mode=0o700)
        host.save_json(self.work / OLD / 'state.json', {'revision': OLD, 'phase': 'complete', 'helpers': previous})
        baseline = prior.upgrade_verify.baseline_helpers(helpers)
        assert previous['projects/guard.py'] != helpers['projects/guard.py']
        installed_guard(self.admin / 'projects/guard.py').require_capable(self.go / 'releases' / OLD / 'agent-board', self.projects)
        state = {'kind': 'capable-update', 'phase': 'staged', 'revision': NEW, 'helpers': helpers,
                 'installed_helpers': baseline, 'go_sha256': host.sha(self.go / 'releases' / NEW / 'agent-board'),
                 'manifest_sha256': 'synthetic-next-boundary', 'candidate_sha256': host.sha(self.nginx),
                 'nginx_sha256': host.sha(self.nginx), 'preserved': {'fixture': 'unchanged'},
                 'old_go': 'releases/' + OLD, 'old_next': 'releases/' + OLD}
        host.save_json(self.work_release / 'state.json', state)
        return state
