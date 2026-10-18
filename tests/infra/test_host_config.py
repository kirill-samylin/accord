"""Real files/fsync; root directory validation is an explicit test double."""
import os
import pathlib
import stat
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2] / 'infra/accord'))
import host_config


class HostConfigTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.config = pathlib.Path(self.temporary.name) / 'config'
        self.path = self.config / 'server.env'
        patch = mock.patch.object(host_config, 'CONFIG', self.config)
        patch.start()
        self.addCleanup(patch.stop)
        original = pathlib.Path.lstat

        def root_owned(path):
            values = list(original(path))
            values[4:6] = (0, 0)
            return os.stat_result(values)
        patch = mock.patch.object(pathlib.Path, 'lstat', root_owned)
        patch.start()
        self.addCleanup(patch.stop)

    def directory(self, path, mode):
        path.mkdir(mode=mode, exist_ok=True)
        self.assertEqual(stat.S_IMODE(path.stat().st_mode), mode)

    def write(self, content):
        self.config.mkdir(mode=0o700, exist_ok=True)
        self.path.write_text(content, encoding='ascii')
        self.path.chmod(0o600)

    def test_rejects_invalid_before_creating_configuration_directory(self):
        for value in ['', '*.example:8443', 'board.example', 'Board.example:8443', 'board.example:08443',
                      'board.example:65536', '127.1:8443', '0x7f000001:8443', 'localhost:8443', 'user@board.example:8443',
                      'board.example:8443/path', 'board.example:8443\n', '[board.example]:8443']:
            with self.assertRaises(RuntimeError):
                host_config.configure(value, self.directory)
            self.assertFalse(self.config.exists())

    def test_atomic_private_file_and_directory_synced(self):
        events = []
        original_fsync = os.fsync

        def fsync(descriptor):
            events.append('directory' if stat.S_ISDIR(os.fstat(descriptor).st_mode) else 'file')
            original_fsync(descriptor)
        with mock.patch.object(host_config.os, 'fsync', side_effect=fsync):
            host_config.configure('board.example:8443', self.directory)
        self.assertEqual(events, ['file', 'directory'])
        self.assertEqual(self.path.read_text(), 'ACCORD_EXTERNAL_HOST=board.example:8443\n')
        self.assertEqual(stat.S_IMODE(self.path.stat().st_mode), 0o600)
        self.assertFalse((self.config / '.server.env.new').exists())

    def test_omission_preserves_configuration_and_valid_explicit_update_works(self):
        host_config.configure(None, self.directory)
        self.assertFalse(self.config.exists())
        self.write('ACCORD_EXTERNAL_HOST=board.example:8443\n')
        inode = self.path.stat().st_ino
        host_config.configure(None, self.directory)
        host_config.configure('board.example:8443', self.directory)
        self.assertEqual(self.path.stat().st_ino, inode)
        host_config.configure('new.example:443', self.directory)
        self.assertEqual(self.path.read_text(), 'ACCORD_EXTERNAL_HOST=new.example:443\n')

    def test_unmanaged_content_permissions_and_symlinks_are_not_overwritten(self):
        for content in ['OTHER_SETTING=1\n', 'ACCORD_EXTERNAL_HOST=board.example:8443\nOTHER=1\n',
                        'ACCORD_EXTERNAL_HOST=*.example:8443\n', '']:
            self.write(content)
            with self.assertRaises(RuntimeError):
                host_config.configure('new.example:8443', self.directory)
            self.assertEqual(self.path.read_text(), content)
        self.write('ACCORD_EXTERNAL_HOST=board.example:8443\n')
        self.path.write_bytes(b'FOREIGN=\xff\n')
        with self.assertRaisesRegex(RuntimeError, 'unmanaged'):
            host_config.configure('board.example:8443', self.directory)
        self.assertEqual(self.path.read_bytes(), b'FOREIGN=\xff\n')
        self.write('ACCORD_EXTERNAL_HOST=board.example:8443\n')
        self.path.chmod(0o644)
        with self.assertRaises(RuntimeError):
            host_config.configure(None, self.directory)
        self.path.unlink()
        self.path.symlink_to(self.config / 'missing')
        with self.assertRaises(RuntimeError):
            host_config.configure('board.example:8443', self.directory)
        self.assertTrue(self.path.is_symlink())

    def test_fsync_failure_preserves_previous_file_and_foreign_staging(self):
        content = 'ACCORD_EXTERNAL_HOST=board.example:8443\n'
        self.write(content)
        with mock.patch.object(host_config.os, 'fsync', side_effect=OSError('synthetic disk failure')):
            with self.assertRaises(OSError):
                host_config.configure('new.example:8443', self.directory)
        self.assertEqual(self.path.read_text(), content)
        staging = self.config / '.server.env.new'
        staging.write_text('foreign')
        with self.assertRaises(FileExistsError):
            host_config.configure('new.example:8443', self.directory)
        self.assertEqual(staging.read_text(), 'foreign')


if __name__ == '__main__':
    unittest.main(verbosity=2)
