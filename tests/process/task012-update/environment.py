"""TASK011 adapters, with real old helpers replaced under the owned update hold."""
import importlib.util
import json
import os
from pathlib import Path
import sys

REPO = Path(__file__).resolve().parents[3]
sys.path.append(str(REPO / 'tests/process/task011-update'))
spec = importlib.util.spec_from_file_location('task011_update_environment', REPO / 'tests/process/task011-update/environment.py')
prior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prior)
from fixture import capture, command, rebind, seed

host, guard, upgrade = prior.host, prior.guard, prior.upgrade
OLD, NEW = prior.OLD, prior.NEW


def installed_guard(path):
    spec = importlib.util.spec_from_file_location('task012_installed_guard', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class Environment(prior.Environment):
    def start_go(self):
        # The old installed guard must accept the old writer before replacement;
        # every actual restart uses the currently installed helper, including resume.
        installed_guard(self.admin / 'projects/guard.py').require_capable(
            (self.go / 'current/agent-board').resolve(), self.projects)
        super().start_go()

    def run(self, *args):
        if args[0] == 'runuser':
            installed = installed_guard(self.admin / 'projects/guard.py')
            assert host.sha(self.admin / 'projects/guard.py') == host.sha(REPO / 'infra/accord/projects/guard.py')
            installed.require_capable(self.go / 'releases' / NEW / 'agent-board', self.projects)
            try:
                installed.require_capable(self.go / 'releases' / OLD / 'agent-board', self.projects)
                raise AssertionError('dedup floor absent before storage operation')
            except RuntimeError:
                pass
            assert (self.state / 'START_REQUIRES_INSPECTION').is_file()
        return super().run(*args)

    def prepare(self):
        old_infra = Path(os.environ['ACCORD_UPDATE_OLD_INFRA']).resolve()
        assert old_infra.is_dir() and not old_infra.is_symlink()
        assert host.sha(old_infra / 'projects/guard.py') == 'c1703c392fc62891e150e23e52cdc68132f05f1c5dad570c53404f4de07b50ed'
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
        assert previous['projects/guard.py'] != helpers['projects/guard.py'], 'old guard replacement not exercised'
        installed_guard(self.admin / 'projects/guard.py').require_capable(
            self.go / 'releases' / OLD / 'agent-board', self.projects)
        state = {'kind': 'capable-update', 'phase': 'staged', 'revision': NEW, 'helpers': helpers,
                 'installed_helpers': baseline, 'go_sha256': host.sha(self.go / 'releases' / NEW / 'agent-board'),
                 'manifest_sha256': 'synthetic-next-boundary', 'candidate_sha256': host.sha(self.nginx),
                 'nginx_sha256': host.sha(self.nginx), 'preserved': {'fixture': 'unchanged'},
                 'old_go': 'releases/' + OLD, 'old_next': 'releases/' + OLD}
        host.save_json(self.work_release / 'state.json', state)
        return state
