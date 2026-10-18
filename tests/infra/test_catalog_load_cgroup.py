"""Synthetic proc/mount/controller fixtures; no Linux or live service required."""
import importlib.util
import pathlib
import tempfile
import unittest
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / 'acceptance' / 'catalog_cgroup.py'
SPEC = importlib.util.spec_from_file_location('catalog_cgroup', MODULE_PATH)
cgroup = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(cgroup)


def escaped(value):
    return (str(value).replace('\\', r'\134').replace(' ', r'\040')
            .replace('\t', r'\011').replace('\n', r'\012'))


class CgroupSnapshotTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = pathlib.Path(temporary.name).resolve()
        self.proc = self.directory / 'proc'
        (self.proc / '123').mkdir(parents=True)
        (self.proc / 'self').mkdir()
        self.mountpoint = self.directory / 'cgroup'
        self.mountpoint.mkdir()
        self.member_file = self.proc / '123' / 'cgroup'
        self.mount_file = self.proc / 'self' / 'mountinfo'
        self.member_file.write_text('0::/system.slice/load.scope\n')
        (self.proc / 'self' / 'cgroup').write_text('0::/unrelated-observer\n')
        self.mount()

    def mount(self, root='/', point=None):
        point = self.mountpoint if point is None else point
        self.mount_file.write_text('42 1 0:30 ' + escaped(root) + ' ' + escaped(point)
                                   + ' rw,nosuid shared:7 - cgroup2 cgroup rw\n')

    def limits(self, relative='', cpu='max 100000', memory='max'):
        leaf = self.mountpoint / relative
        leaf.mkdir(parents=True, exist_ok=True)
        (leaf / 'cpu.max').write_text(cpu + '\n')
        (leaf / 'memory.max').write_text(memory + '\n')
        return leaf

    def snapshot(self):
        return cgroup.cgroup_limits(123, self.proc)

    def test_child_nested_group_does_not_report_unlimited_root(self):
        self.limits()
        leaf = self.limits('system.slice/load.scope', '100000 100000', '536870912')
        result = self.snapshot()
        self.assertEqual(result['status'], 'available')
        self.assertEqual(result['pid'], 123)
        self.assertEqual(result['membership_path'], '/system.slice/load.scope')
        self.assertEqual(result['leaf_path'], str(leaf))
        self.assertEqual(result['leaf_limits']['cpu.max']['value'], '100000 100000')
        self.assertEqual(result['leaf_limits']['memory.max']['value'], '536870912')
        self.assertEqual(result['effective_limits']['status'], 'unavailable')

    def test_subtree_mount_root_is_removed_from_membership(self):
        self.mount(root='/system.slice')
        leaf = self.limits('load.scope', '50000 100000', '268435456')
        result = self.snapshot()
        self.assertEqual(result['leaf_path'], str(leaf))
        self.assertEqual(result['mount_root'], '/system.slice')
        self.assertEqual(result['status'], 'available')

    def test_root_membership_reports_only_observed_root_files(self):
        self.member_file.write_text('0::/\n')
        self.limits()
        result = self.snapshot()
        self.assertEqual(result['leaf_limits']['memory.max']['value'], 'max')
        self.assertEqual(result['membership_path'], '/')
        self.assertEqual(result['effective_limits']['status'], 'unavailable')

    def test_mountinfo_escapes_and_raw_membership_are_distinct(self):
        self.mountpoint = self.directory / 'space tab\tnewline\nback\\slash'
        self.mountpoint.mkdir()
        self.member_file.write_text('0::/slice name/child\\040literal\n')
        self.mount(root='/slice name')
        leaf = self.limits(r'child\040literal')
        result = self.snapshot()
        self.assertEqual(result['status'], 'available')
        self.assertEqual(result['leaf_path'], str(leaf))
        self.assertEqual(result['mount_point'], str(self.mountpoint))

    def test_traversal_and_noncanonical_membership_unavailable(self):
        for path in ('/../outside', '/slice/../outside', '/./scope', '//scope', 'relative'):
            with self.subTest(path=path):
                self.member_file.write_text('0::' + path + '\n')
                result = self.snapshot()
                self.assertEqual(result['status'], 'unavailable')
                self.assertNotIn('leaf_limits', result)

    def test_invalid_mount_paths_unavailable(self):
        for root in ('/../slice', r'/bad\057slice', r'/bad\oops'):
            with self.subTest(root=root):
                self.mount_file.write_text('42 1 0:30 ' + root + ' ' + str(self.mountpoint)
                                           + ' rw - cgroup2 cgroup rw\n')
                self.assertEqual(self.snapshot()['status'], 'unavailable')

    def test_symlink_cannot_escape_mountpoint(self):
        outside = self.directory / 'outside'
        outside.mkdir()
        (self.mountpoint / 'system.slice').symlink_to(outside, target_is_directory=True)
        (outside / 'load.scope').mkdir()
        result = self.snapshot()
        self.assertEqual(result['status'], 'unavailable')
        self.assertIn('escapes mountpoint', result['reason'])

    def test_no_matching_mount_does_not_use_prefix_or_root_fallback(self):
        self.limits()
        for root in ('/unrelated', '/system'):
            with self.subTest(root=root):
                self.mount(root=root)
                result = self.snapshot()
                self.assertEqual(result['status'], 'unavailable')
                self.assertNotIn('leaf_limits', result)

    def test_duplicate_matching_mount_is_ambiguous(self):
        self.mount_file.write_text(self.mount_file.read_text() * 2)
        self.assertEqual(self.snapshot()['status'], 'unavailable')

    def test_missing_ambiguous_deleted_and_v1_membership_unavailable(self):
        for text in ('', '2:cpu:/v1\n', '0::/a\n0::/b\n', '0::/gone (deleted)\n',
                     '0::/path\nraw-newline-suffix\n'):
            with self.subTest(text=text):
                self.member_file.write_text(text)
                self.assertEqual(self.snapshot()['status'], 'unavailable')
        self.member_file.unlink()
        self.assertEqual(self.snapshot()['reason'], 'FileNotFoundError')

    def test_missing_or_malformed_mountinfo_unavailable(self):
        for text in ('', '42 1 - cgroup2 cgroup rw\n',
                     '42 1 0:30 / /tmp rw - tmpfs tmpfs rw\n'):
            with self.subTest(text=text):
                self.mount_file.write_text(text)
                self.assertEqual(self.snapshot()['status'], 'unavailable')
        self.mount_file.unlink()
        self.assertEqual(self.snapshot()['reason'], 'FileNotFoundError')

    def test_unreadable_or_missing_leaf_file_is_explicitly_unavailable(self):
        leaf = self.limits('system.slice/load.scope')
        (leaf / 'memory.max').unlink()
        result = self.snapshot()
        self.assertEqual(result['status'], 'partial')
        self.assertEqual(result['leaf_limits']['memory.max']['status'], 'unavailable')
        original = pathlib.Path.read_text

        def denied(path, *args, **kwargs):
            if path.name == 'cpu.max':
                raise PermissionError('synthetic')
            return original(path, *args, **kwargs)

        with mock.patch.object(pathlib.Path, 'read_text', denied):
            result = self.snapshot()
        self.assertEqual(result['status'], 'unavailable')
        self.assertEqual(result['leaf_limits']['cpu.max']['reason'], 'PermissionError')

    def test_invalid_controller_values_do_not_become_limits(self):
        self.limits('system.slice/load.scope', '0 100000', 'not-a-number')
        result = self.snapshot()
        self.assertEqual(result['status'], 'unavailable')
        self.assertNotIn('value', result['leaf_limits']['cpu.max'])
        self.assertNotIn('value', result['leaf_limits']['memory.max'])

    def test_migration_during_snapshot_discards_leaf_values(self):
        self.limits('system.slice/load.scope')
        original = pathlib.Path.read_text
        reads = 0

        def moved(path, *args, **kwargs):
            nonlocal reads
            if path == self.member_file:
                reads += 1
                if reads == 2:
                    return '0::/moved.scope\n'
            return original(path, *args, **kwargs)

        with mock.patch.object(pathlib.Path, 'read_text', moved):
            result = self.snapshot()
        self.assertEqual(result['status'], 'unavailable')
        self.assertNotIn('leaf_limits', result)


if __name__ == '__main__':
    unittest.main()
