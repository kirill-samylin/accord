import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("rollout", ROOT / "infra/accord/rollout.py")
rollout = importlib.util.module_from_spec(spec)
spec.loader.exec_module(rollout)

REVISION = "a" * 40


def digest(data):
    return hashlib.sha256(data).hexdigest()


class RolloutPlanningTests(unittest.TestCase):
    def test_read_sums_accepts_sha256sum_output(self):
        text = digest(b"x") + "  agent-board\n" + digest(b"y") + " *infra/install.py\n\n"
        self.assertEqual(rollout.read_sums(text), {"agent-board": digest(b"x"), "infra/install.py": digest(b"y")})
        with self.assertRaises(ValueError):
            rollout.read_sums("not a checksum line")

    def test_pending_phases_follow_the_server_journal(self):
        phases = list(rollout.PHASES)
        self.assertEqual(rollout.pending_phases(phases, None), phases)
        self.assertEqual(rollout.pending_phases(phases, "staged"), ["apply", "smoke", "finalize"])
        self.assertEqual(rollout.pending_phases(phases, "pending-https-smoke"), ["smoke", "finalize"])
        self.assertEqual(rollout.pending_phases(phases, "complete"), [])
        self.assertEqual(rollout.pending_phases(["smoke"], "pending-https-smoke"), ["smoke"])

    def test_interrupted_apply_requires_explicit_resume(self):
        phases = list(rollout.PHASES)
        for state in rollout.RESUMABLE:
            with self.assertRaises(RuntimeError):
                rollout.pending_phases(phases, state)
            self.assertEqual(rollout.pending_phases(phases, state, resume=True), ["apply", "smoke", "finalize"])
        with self.assertRaises(RuntimeError):
            rollout.pending_phases(phases, "something-else")

    def test_plan_prune_keeps_active_protected_and_newest(self):
        releases = {"old1": 1.0, "old2": 2.0, "old3": 3.0, "active": 4.0, "candidate": 5.0}
        self.assertEqual(sorted(rollout.plan_prune(releases, "active", 1, ["candidate"])), ["old1", "old2"])
        self.assertEqual(sorted(rollout.plan_prune(releases, "active", 0, ["candidate"])), ["old1", "old2", "old3"])
        self.assertEqual(rollout.plan_prune(releases, "active", 10, ["candidate"]), [])
        self.assertEqual(rollout.plan_prune(releases, "active", -5, ["candidate"]), ["old3", "old2", "old1"])


class ReleaseValidationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def go_release(self, revision=REVISION):
        release = self.root / "go"
        (release / "infra").mkdir(parents=True)
        (release / "agent-board").write_bytes(b"binary")
        (release / "infra/install.py").write_bytes(b"print()")
        (release / "SOURCE").write_text(revision + "\n")
        sums = "".join(digest(data) + "  " + name + "\n" for name, data in
                       [("SOURCE", (revision + "\n").encode()), ("agent-board", b"binary"), ("infra/install.py", b"print()")])
        (release / "SHA256SUMS").write_text(sums)
        return release

    def next_bundle(self, revision=REVISION):
        bundle = self.root / "next"
        (bundle / "app").mkdir(parents=True)
        (bundle / "node").write_bytes(b"node")
        (bundle / "app/server.js").write_bytes(b"js")
        files = {"node": digest(b"node"), "app/server.js": digest(b"js")}
        (bundle / "manifest.json").write_text(json.dumps({"format": 1, "revision": revision, "files": files}))
        return bundle

    def test_go_release_identity(self):
        release = self.go_release()
        info = rollout.load_go_release(release)
        self.assertEqual(info["revision"], REVISION)
        self.assertEqual(info["go_sha256"], digest(b"binary"))
        self.assertEqual(set(info["files"]), {"SOURCE", "agent-board", "infra/install.py", "SHA256SUMS"})
        self.assertEqual(info["manifest_sha256"], digest((release / "SHA256SUMS").read_bytes()))

    def test_go_release_rejects_drift(self):
        release = self.go_release()
        (release / "agent-board").write_bytes(b"other")
        with self.assertRaises(ValueError):
            rollout.load_go_release(release)
        (release / "agent-board").write_bytes(b"binary")
        (release / "extra").write_bytes(b"")
        with self.assertRaises(ValueError):
            rollout.load_go_release(release)

    def test_next_bundle_identity_and_revision(self):
        bundle = self.next_bundle()
        info = rollout.load_next_bundle(bundle, REVISION)
        self.assertEqual(set(info["files"]), {"node", "app/server.js", "manifest.json"})
        self.assertEqual(info["manifest_sha256"], digest((bundle / "manifest.json").read_bytes()))
        with self.assertRaises(ValueError):
            rollout.load_next_bundle(bundle, "b" * 40)
        (bundle / "app/server.js").write_bytes(b"changed")
        with self.assertRaises(ValueError):
            rollout.load_next_bundle(bundle, REVISION)


if __name__ == "__main__":
    unittest.main()
